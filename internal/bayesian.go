package regelverk

import (
	"log/slog"
	"math"
	"time"
)

type BayesianModel struct {
	Prior       float64
	Threshold   float64
	Likelihoods map[StateKey][]LikelihoodModel
}

type LikelihoodModel struct {

	// ProbGivenTrue = P(E | H):
	//   The probability of seeing this evidence (E)
	//   when the hypothesis (H) is true.
	// Example: If someone is home (H), there's a 90% chance
	// that the phone (E) is detected as connected to Wi-Fi.
	ProbGivenTrue float64

	// ProbGivenFalse = P(E | ~H):
	//   The probability of seeing this evidence (E)
	//   when the hypothesis (H) is false.
	// Example: Even if nobody is home (~H), there is still a 20%
	// chance that the phone shows up as "home" due to GPS errors.
	ProbGivenFalse float64

	// HalfLife defines how quickly this evidence decays.
	// A shorter half-life means old observations lose their influence faster.
	// Using time.Duration keeps this semantically correct and type-safe.
	HalfLife time.Duration

	// Weight controls how strongly this observation influences the final result.
	// A weight > 1.0 increases its impact, and a weight < 1.0 reduces it.
	Weight float64

	// Compute the value to use for the given StateValue
	// Returns the value and the age of the value.
	// The age will be used to apply decay. If <= 0 then no decay is applied.
	StateValueEvaluator func(StateValue) (bool, time.Duration)
}

func (likelihoodModel LikelihoodModel) plusComplement() []LikelihoodModel {
	complement := LikelihoodModel{
		ProbGivenTrue:  (1 - likelihoodModel.ProbGivenTrue),
		ProbGivenFalse: (1 - likelihoodModel.ProbGivenFalse),
		HalfLife:       likelihoodModel.HalfLife,
		Weight:         likelihoodModel.Weight,
		StateValueEvaluator: func(sv StateValue) (bool, time.Duration) {
			b, duration := likelihoodModel.StateValueEvaluator(sv)
			return !b, duration
		},
	}
	return []LikelihoodModel{likelihoodModel, complement}
}

var currentlyTrue = func(value StateValue) (bool, time.Duration) {
	return value.currentlyTrue(), 0
}

var currentlyFalse = func(value StateValue) (bool, time.Duration) {
	return value.currentlyFalse(), 0
}

type Observation struct {
	Name      string
	Matched   bool      // Whether the evidence was observed (true) or absent (false)
	Timestamp time.Time // When the observation occurred
}

// Applies exponential decay to a probability based on how old the observation is.
// Converts half-life to minutes internally for exponential math.
// This reduces the influence of stale evidence over time.
func applyTimeDecay(p float64, age time.Duration, halfLife time.Duration) float64 {
	if halfLife <= 0 {
		return p // No decay applied
	}
	ageMin := age.Minutes()
	halfLifeMin := halfLife.Minutes()
	decay := math.Exp(-math.Ln2 * ageMin / halfLifeMin)
	return 1 - (1-p)*decay
}

func clipProb(p float64) float64 {
	const eps = 1e-12
	if p < eps {
		return eps
	}
	if p > 1-eps {
		return 1 - eps
	}
	return p
}

// Performs one Bayesian update in log-odds space, applying a weight to control the influence of this observation.
func applyBayes(prior float64, likelihood LikelihoodModel, matched bool, age time.Duration) float64 {

	// Time‑decay the conditional probabilities.
	pTrue := applyTimeDecay(likelihood.ProbGivenTrue, age, likelihood.HalfLife)
	pFalse := applyTimeDecay(likelihood.ProbGivenFalse, age, likelihood.HalfLife)

	var posterior float64
	// Perform calculation in log-odds space to apply weighting
	// log likelihood ratio (LR) and weighting:
	LR := clipProb(pTrue) / clipProb(pFalse)          // LR
	logLRWeighted := likelihood.Weight * math.Log(LR) // (log LR) * w

	// Prior in log‑odds
	logOddsPrior := math.Log(prior) - math.Log(1-prior)

	// Posterior in log‑odds + conversion back to probability.
	// For numerical stability, clamp logOddsPost to interval where Exp will not overflow
	logOddsPost := logOddsPrior + logLRWeighted
	if matched {
		const expClamp = 700.0
		if logOddsPost > expClamp {
			logOddsPost = expClamp
		}
		if logOddsPost < -expClamp {
			logOddsPost = -expClamp
		}
		posterior = 1.0 / (1.0 + math.Exp(-logOddsPost))
	} else {
		posterior = prior
	}

	slog.Debug("Posterior calculation",
		"matched", matched, "prior", prior,
		"probGivenTrue", likelihood.ProbGivenTrue,
		"probGivenFalse", likelihood.ProbGivenFalse,
		"pTrue", pTrue,
		"pFalse", pFalse,
		"age_minutes", age.Minutes(),
		"weight", likelihood.Weight,
		"posterior", posterior)

	return posterior
}

func inferPosterior(bayesianModel BayesianModel, stateValueMap *StateValueMap) (float64, bool) {

	now := time.Now()
	p := bayesianModel.Prior

	for key, likelihoods := range bayesianModel.Likelihoods {

		state, found := stateValueMap.getState(key)
		if found {
			for _, likelihood := range likelihoods {
				var value bool
				var age time.Duration
				if likelihood.StateValueEvaluator != nil {
					value, age = likelihood.StateValueEvaluator(state)
				} else {
					value = state.value
					age = now.Sub(state.lastUpdate)
				}

				updatedPosterior := applyBayes(p, likelihood, value, age)
				p = updatedPosterior
			}
		} else {
			slog.Debug("Observation update, state not found",
				"observation", key,
			)
		}
	}
	return p, p >= bayesianModel.Threshold
}
