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

type Observation struct {
	Name      string
	Matched   bool      // Whether the evidence was observed (true) or absent (false)
	Timestamp time.Time // When the observation occurred
}

// Converts a probability to log-odds (logit):
// logit(p) = ln(p / (1 - p))
func logOdds(p float64) float64 {
	return math.Log(p / (1 - p))
}

// Converts log-odds back to a probability using the sigmoid function:
// sigmoid(x) = 1 / (1 + e^(-x))
func sigmoid(logit float64) float64 {
	return 1 / (1 + math.Exp(-logit))
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

// Performs one Bayesian update in log-odds space, applying a weight to control the influence of this observation.
func applyWeightedBayes(prior float64, likelihood LikelihoodModel, matched bool, age time.Duration) float64 {
	// Invert probabilities if the observation was NOT matched (absence of event)
	pTrue := likelihood.ProbGivenTrue
	pFalse := likelihood.ProbGivenFalse
	if !matched {
		pTrue = 1 - pTrue
		pFalse = 1 - pFalse
	}

	// Apply decay to both likelihoods
	pTrue = applyTimeDecay(pTrue, age, likelihood.HalfLife)
	pFalse = applyTimeDecay(pFalse, age, likelihood.HalfLife)

	// Convert prior belief to log-odds
	priorLogOdds := logOdds(prior)

	// Likelihood ratio in log form
	likelihoodLog := math.Log(pTrue / pFalse)

	// Apply weight to control this observation's influence
	weightedLogOdds := priorLogOdds + likelihood.Weight*likelihoodLog

	// Convert back to a probability (posterior)
	return sigmoid(weightedLogOdds)
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

				updated := applyWeightedBayes(p, likelihood, value, age)

				slog.Debug("Observation update",
					"observation", key,
					"value", value,
					"age_minutes", age.Minutes(),
					"weight", likelihood.Weight,
					"decayed_P(E|H)", likelihood.ProbGivenTrue,
					"decayed_P(E|~H)", likelihood.ProbGivenFalse,
					"posterior_before", p,
					"posterior_after", updated,
				)
				p = updated
			}
		} else {
			slog.Debug("Observation update, state not found",
				"observation", key,
			)
		}
	}
	return p, p >= bayesianModel.Threshold
}
