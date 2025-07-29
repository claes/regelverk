package regelverk

import (
	"log/slog"
	"math"
	"os"
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

	decayed := applyTimeDecay(baseProb, age, halfLife)

	expected := 1 - (1-baseProb)*0.5 // 1 half-life → 50% decay
	if !floatEquals(decayed, expected, 0.0001) {
		t.Errorf("Expected %.4f, got %.4f", expected, decayed)
	}
}

func TestApplyWeightedBayes(t *testing.T) {
	prior := 0.5
	likelihood := LikelihoodModel{
		ProbGivenTrue:  0.9,
		ProbGivenFalse: 0.1,
		HalfLife:       60 * time.Minute, // Using time.Duration
		Weight:         1.0,
	}
	age := 0 * time.Minute

	posterior := applyBayes(prior, likelihood, true, age)
	expected := 0.9 // Should be close given likelihood ratio is strong

	if !floatEquals(posterior, expected, 0.0001) {
		t.Errorf("Expected %.4f, got %.4f", expected, posterior)
	}

	if posterior <= prior {
		t.Errorf("Posterior %.4f should be greater than prior %.4f", posterior, prior)
	}
}

func TestApplyBayesianInferenceWithDuration(t *testing.T) {
	likelihoods := map[StateKey][]LikelihoodModel{
		"motion": {
			{
				ProbGivenTrue:  0.9,
				ProbGivenFalse: 0.1,
				HalfLife:       60 * time.Minute,
				Weight:         1.0,
			},
		},
	}

	observations := NewStateValueMap()
	observations.setState(StateKey("motion"), true)
	s, _ := observations.getState(StateKey("motion"))
	s.lastUpdate = time.Now().Add(-30 * time.Minute)
	observations.setStateValue(StateKey("motion"), s)

	bayesianModel := BayesianModel{
		Prior:       0.5,
		Threshold:   0.7,
		Likelihoods: likelihoods,
	}
	posterior, decision := inferPosterior(bayesianModel, &observations)

	if !decision {
		t.Errorf("Expected decision to be true with posterior %.4f", posterior)
	}

	if posterior <= bayesianModel.Prior {
		t.Errorf("Posterior %.4f should be greater than prior %.4f", posterior, bayesianModel.Prior)
	}
}

func TestApplyBayesianInferenceWithDuration2(t *testing.T) {

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// Test case from https://docs.google.com/spreadsheets/d/16u9RVKRUVjTraX7J26rvuaLKQGxwUN-0pbal97TRY5w/edit?gid=0#gid=0

	likelihoods := map[StateKey][]LikelihoodModel{
		"tv": {
			{
				ProbGivenTrue:       4.0 / 14,
				ProbGivenFalse:      0.1 / 10,
				HalfLife:            0,
				Weight:              1.0,
				StateValueEvaluator: currentlyTrue,
			},
			{
				ProbGivenTrue:       10.0 / 14,
				ProbGivenFalse:      9.9 / 10,
				HalfLife:            0,
				Weight:              1.0,
				StateValueEvaluator: currentlyFalse,
			},
		},
		"lights": {
			{
				ProbGivenTrue:       3.0 / 14,
				ProbGivenFalse:      0.1 / 10,
				HalfLife:            0,
				Weight:              1.0,
				StateValueEvaluator: currentlyTrue,
			},
			{
				ProbGivenTrue:       10.0 / 14,
				ProbGivenFalse:      9.9 / 10,
				HalfLife:            0,
				Weight:              1.0,
				StateValueEvaluator: currentlyFalse,
			},
		},
		"carHome": {
			{
				ProbGivenTrue:       10.0 / 14,
				ProbGivenFalse:      4.0 / 10,
				HalfLife:            0,
				Weight:              1.0,
				StateValueEvaluator: currentlyTrue,
			},
			{
				ProbGivenTrue:       6.0 / 14,
				ProbGivenFalse:      6.0 / 10,
				HalfLife:            0,
				Weight:              1.0,
				StateValueEvaluator: currentlyFalse,
			},
		},
	}

	observations := NewStateValueMap()

	observations.setState(StateKey("tv"), false)
	observations.setState(StateKey("lights"), true)
	observations.setState(StateKey("carHome"), true)

	bayesianModel := BayesianModel{
		Prior:       14.0 / 24,
		Threshold:   0.8,
		Likelihoods: likelihoods,
	}
	posterior, decision := inferPosterior(bayesianModel, &observations)

	if !decision {
		t.Errorf("Expected decision to be true with posterior %.4f", posterior)
	}

	// check if posterior is between 0.974 and 0.975
	if posterior < 0.974 || posterior > 0.975 {
		t.Errorf("Posterior %.4f should be between 0.974 and 0.975", posterior)
	}

}
