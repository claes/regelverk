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
	s.lastUpdate = nowFunc().Add(-30 * time.Minute)
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

	// logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	// 	Level: slog.LevelDebug,
	// }))
	// slog.SetDefault(logger)

	// Test case from  https://docs.google.com/spreadsheets/d/16u9RVKRUVjTraX7J26rvuaLKQGxwUN-0pbal97TRY5w/edit?gid=0#gid=0
	// Originally from https://docs.google.com/spreadsheets/d/1sV5WHM0GTG9oXGuO7QMOOHZDVdWVY0D9bTVLUmSM4co/edit?gid=0#gid=0
	houseOccupiedLikelihoods := map[StateKey][]LikelihoodModel{
		"tv": {
			{
				ProbGivenTrue:       4.0 / 14, //Probability of measuring TV on when house occupied
				ProbGivenFalse:      0.1 / 10, //Probability of measuring TV on when house not occupied
				HalfLife:            0,
				Weight:              1.0,
				StateValueEvaluator: currentlyTrue, // TV on
			},
			{
				ProbGivenTrue:       10.0 / 14, // Probability of measuring TV off when house occupied
				ProbGivenFalse:      9.9 / 10,  // Probability of measuring TV off when house not occupied
				HalfLife:            0,
				Weight:              1.0,
				StateValueEvaluator: currentlyFalse, // TV off
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
		Likelihoods: houseOccupiedLikelihoods,
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

func TestApplyBayesianInferenceWithDuration3(t *testing.T) {

	// logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	// 	Level: slog.LevelDebug,
	// }))
	// slog.SetDefault(logger)

	// Test case from  https://docs.google.com/spreadsheets/d/1pcsifVnXngaSsyH4zPPVYuF-13gTvgSMZGnCqFI1auE/edit?gid=982845486#gid=982845486
	// Originally from https://docs.google.com/spreadsheets/d/17aDaO8Na2FiLXdlBmpJA1AGsGEGnGaZG24eJTSz1gko/edit?gid=982845486#gid=982845486
	inBedLikelihoods := map[StateKey][]LikelihoodModel{
		"livingroom_motion": LikelihoodModel{
			ProbGivenTrue:       0.3 / 6,
			ProbGivenFalse:      3.6 / (24 - 6),
			HalfLife:            0,
			Weight:              1.0,
			StateValueEvaluator: currentlyTrue,
		}.plusComplement(),
		"basement_motion": LikelihoodModel{
			ProbGivenTrue:       3.0 / 6,
			ProbGivenFalse:      5.4 / (24 - 6),
			HalfLife:            0,
			Weight:              1.0,
			StateValueEvaluator: currentlyTrue,
		}.plusComplement(),
		"bedroom_motion": LikelihoodModel{
			ProbGivenTrue:       3.0 / 6,
			ProbGivenFalse:      1.8 / (24 - 6),
			HalfLife:            0,
			Weight:              1.0,
			StateValueEvaluator: currentlyTrue,
		}.plusComplement(),
		"sun": LikelihoodModel{
			ProbGivenTrue:       4.2 / 6,
			ProbGivenFalse:      8.1 / (24 - 6),
			HalfLife:            0,
			Weight:              1.0,
			StateValueEvaluator: currentlyTrue,
		}.plusComplement(),
		"android": LikelihoodModel{
			ProbGivenTrue:       5.7 / 6,
			ProbGivenFalse:      1.8 / (24 - 6),
			HalfLife:            0,
			Weight:              1.0,
			StateValueEvaluator: currentlyTrue,
		}.plusComplement(),
	}

	observations := NewStateValueMap()

	observations.setState(StateKey("livingroom_motion"), false)
	observations.setState(StateKey("basement_motion"), false)
	observations.setState(StateKey("bedroom_motion"), false)
	observations.setState(StateKey("sun"), true)
	observations.setState(StateKey("android"), true)

	bayesianModel := BayesianModel{
		Prior:       6.0 / 24,
		Threshold:   0.5,
		Likelihoods: inBedLikelihoods,
	}
	posterior, decision := inferPosterior(bayesianModel, &observations)

	if !decision {
		t.Errorf("Expected decision to be true with posterior %.4f", posterior)
	}

	if posterior < 0.69 || posterior > 0.70 {
		t.Errorf("Posterior %.4f should be between 0.69 and 0.70", posterior)
	}

}
