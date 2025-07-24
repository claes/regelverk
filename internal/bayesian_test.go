package regelverk

import (
	"math"
	"testing"
	"time"
)

func floatEquals(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}

func TestApplyTimeDecay(t *testing.T) {
	baseProb := 0.8
	age := 30 * time.Minute
	halfLife := 30 * time.Minute // Now using time.Duration

	decayed := ApplyTimeDecay(baseProb, age, halfLife)

	expected := 1 - (1-baseProb)*0.5 // 1 half-life → 50% decay
	if !floatEquals(decayed, expected, 0.0001) {
		t.Errorf("Expected %.4f, got %.4f", expected, decayed)
	}
}

func TestApplyWeightedBayes(t *testing.T) {
	prior := 0.5
	rule := LikelihoodModel{
		Name:           "motion",
		ProbGivenTrue:  0.9,
		ProbGivenFalse: 0.1,
		HalfLife:       60 * time.Minute, // Using time.Duration
		Weight:         1.0,
	}
	age := 0 * time.Minute

	posterior := ApplyWeightedBayes(prior, rule, true, age)
	//expected := 0.9 // Should be close given likelihood ratio is strong

	if posterior <= prior {
		t.Errorf("Posterior %.4f should be greater than prior %.4f", posterior, prior)
	}
}

func TestApplyBayesianInferenceWithDuration(t *testing.T) {
	now := time.Now()
	rules := map[string]LikelihoodModel{
		"motion": {
			Name:           "motion",
			ProbGivenTrue:  0.9,
			ProbGivenFalse: 0.1,
			HalfLife:       60 * time.Minute,
			Weight:         1.0,
		},
	}

	observations := []Observation{
		{
			Name:      "motion",
			Matched:   true,
			Timestamp: now.Add(-30 * time.Minute),
		},
	}

	prior := 0.5
	threshold := 0.7

	bayesianModel := BayesianModel{
		Prior:       prior,
		Threshold:   threshold,
		Likelihoods: rules,
	}
	posterior, decision := ApplyBayesianInference(bayesianModel, observations, false)

	if !decision {
		t.Errorf("Expected decision to be true with posterior %.4f", posterior)
	}

	if posterior <= prior {
		t.Errorf("Posterior %.4f should be greater than prior %.4f", posterior, prior)
	}
}
