package regelverk

import (
	"fmt"
	"math"
	"time"
)

type BayesianRule struct {
	Name string

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
}

type Observation struct {
	Name      string
	Matched   bool      // Whether the evidence was observed (true) or absent (false)
	Timestamp time.Time // When the observation occurred
}

// Converts a probability to log-odds (logit):
// logit(p) = ln(p / (1 - p))
func LogOdds(p float64) float64 {
	return math.Log(p / (1 - p))
}

// Converts log-odds back to a probability using the sigmoid function:
// sigmoid(x) = 1 / (1 + e^(-x))
func Sigmoid(logit float64) float64 {
	return 1 / (1 + math.Exp(-logit))
}

// Applies exponential decay to a probability based on how old the observation is.
// Converts half-life to minutes internally for exponential math.
// This reduces the influence of stale evidence over time.
func ApplyTimeDecay(p float64, age time.Duration, halfLife time.Duration) float64 {
	if halfLife <= 0 {
		return p // No decay applied
	}
	ageMin := age.Minutes()
	halfLifeMin := halfLife.Minutes()
	decay := math.Exp(-math.Ln2 * ageMin / halfLifeMin)
	return 1 - (1-p)*decay
}

// Performs one Bayesian update in log-odds space, applying a weight to control the influence of this observation.
func ApplyWeightedBayes(prior float64, rule BayesianRule, matched bool, age time.Duration) float64 {
	// Invert probabilities if the observation was NOT matched (absence of event)
	pTrue := rule.ProbGivenTrue
	pFalse := rule.ProbGivenFalse
	if !matched {
		pTrue = 1 - pTrue
		pFalse = 1 - pFalse
	}

	// Apply decay to both likelihoods
	pTrue = ApplyTimeDecay(pTrue, age, rule.HalfLife)
	pFalse = ApplyTimeDecay(pFalse, age, rule.HalfLife)

	// Convert prior belief to log-odds
	priorLogOdds := LogOdds(prior)

	// Likelihood ratio in log form
	likelihoodLog := math.Log(pTrue / pFalse)

	// Apply weight to control this observation's influence
	weightedLogOdds := priorLogOdds + rule.Weight*likelihoodLog

	// Convert back to a probability (posterior)
	return Sigmoid(weightedLogOdds)
}

// Applies all observations in sequence, updating belief each time.
func ApplyBayesianInference(
	prior float64,
	rules map[string]BayesianRule,
	observations []Observation,
	threshold float64,
	verbose bool,
) (float64, bool) {
	now := time.Now()
	p := prior

	for _, obs := range observations {
		rule, ok := rules[obs.Name]
		if !ok {
			if verbose {
				fmt.Printf("⚠️ No rule for '%s' — skipping\n", obs.Name)
			}
			continue
		}

		age := now.Sub(obs.Timestamp)
		updated := ApplyWeightedBayes(p, rule, obs.Matched, age)

		if verbose {
			fmt.Printf("🔎 Observation: %s\n", obs.Name)
			fmt.Printf("  Matched: %v, Age: %.1f min, Weight: %.2f\n", obs.Matched, age.Minutes(), rule.Weight)
			fmt.Printf("  Decayed P(E|H): %.3f, P(E|~H): %.3f\n", rule.ProbGivenTrue, rule.ProbGivenFalse)
			fmt.Printf("  Posterior: %.4f → %.4f\n\n", p, updated)
		}

		p = updated
	}

	return p, p >= threshold
}

func main() {
	prior := 0.6
	threshold := 0.9
	now := time.Now()

	rules := map[string]BayesianRule{
		"phone": {
			Name:           "phone",
			ProbGivenTrue:  0.9,              // If home, phone detected 90% of the time
			ProbGivenFalse: 0.2,              // If not home, phone still shows up 20% of the time
			HalfLife:       60 * time.Minute, // Evidence fades slowly
			Weight:         1.0,              // Highly trusted
		},
		"motion": {
			Name:           "motion",
			ProbGivenTrue:  0.8, // If home, motion detected 80% of the time
			ProbGivenFalse: 0.3, // If not home, motion falsely triggered 30% of the time
			HalfLife:       15 * time.Minute,
			Weight:         0.5, // Less trusted
		},
		"door": {
			Name:           "door",
			ProbGivenTrue:  0.6, // If home, door open 60% of the time
			ProbGivenFalse: 0.4, // Even if not home, 40% chance door is open
			HalfLife:       30 * time.Minute,
			Weight:         0.8,
		},
	}

	observations := []Observation{
		{Name: "phone", Matched: true, Timestamp: now.Add(-5 * time.Minute)},
		{Name: "motion", Matched: false, Timestamp: now.Add(-20 * time.Minute)}, // No motion
		{Name: "door", Matched: true, Timestamp: now.Add(-2 * time.Minute)},
	}

	posterior, decision := ApplyBayesianInference(prior, rules, observations, threshold, true)

	fmt.Printf("🧠 Final probability: %.4f\n", posterior)
	if decision {
		fmt.Println("✅ Decision: Someone is home.")
	} else {
		fmt.Println("❌ Decision: No one is home.")
	}
}
