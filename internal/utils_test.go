package regelverk

import (
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	// fix nowFunc to a constant instant
	fixed := time.Date(2025, 7, 27, 12, 0, 0, 0, time.UTC)
	nowFunc = func() time.Time { return fixed }
	os.Exit(m.Run())
}

// makeStateValue sets only the timestamp matching the current value.
// Useful for continuously*/edge tests.
func makeStateValue(val bool, ago time.Duration) StateValue {
	sv := StateValue{value: val, isDefined: true}
	t0 := nowFunc().Add(-ago)
	if val {
		sv.lastSetTrue = t0
	} else {
		sv.lastSetFalse = t0
	}
	return sv
}

// makeStateValueTrue always sets lastSetTrue, regardless of val.
// Useful for recentlyTrue scenarios.
func makeStateValueTrue(val bool, lastSetTrue time.Duration) StateValue {
	sv := StateValue{value: val, isDefined: true}
	sv.lastSetTrue = nowFunc().Add(-lastSetTrue)
	return sv
}

// makeStateValueFalse always sets lastSetFalse, regardless of val.
// Useful for recentlyFalse scenarios.
func makeStateValueFalse(val bool, lastSetFalse time.Duration) StateValue {
	sv := StateValue{value: val, isDefined: true}
	sv.lastSetFalse = nowFunc().Add(-lastSetFalse)
	return sv
}

/*----------------------------------------------------------------------*/
/* 2.  “Currently true/false”                                            */

func TestStateValue_Currently(t *testing.T) {

	sv := StateValue{value: true}
	if got := sv.requireCurrentlyTrue(); got != true {
		t.Errorf("CurrentlyTrue = %v, want %v", got, true)
	}
	if got := sv.requireCurrentlyFalse(); got != false {
		t.Errorf("CurrentlyTrue = %v, want %v", got, false)
	}

	sv = StateValue{value: false}
	if got := sv.requireCurrentlyTrue(); got != false {
		t.Errorf("CurrentlyTrue = %v, want %v", got, false)
	}
	if got := sv.requireCurrentlyFalse(); got != true {
		t.Errorf("CurrentlyTrue = %v, want %v", got, true)
	}
}

/*----------------------------------------------------------------------*/
/* 3.  ContinuouslyTrue / ContinuouslyFalse                              */

func TestStateValue_ContinuouslyTrue(t *testing.T) {
	d := 5 * time.Second
	cases := []struct {
		name string
		sv   StateValue
		want bool
	}{
		{"long enough", makeStateValue(true, d+time.Millisecond), true},
		{"change too recent", makeStateValue(true, d-time.Millisecond), false},
		{"wrong current", makeStateValue(false, d+time.Millisecond), false},
		{"never true", StateValue{}, false},
		{"exact boundary (strict <)", makeStateValue(true, d), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.sv.continuouslyTrue(d); got != tc.want {
				t.Errorf("continuouslyTrue = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStateValue_ContinuouslyFalse(t *testing.T) {
	d := 5 * time.Second
	cases := []struct {
		name string
		sv   StateValue
		want bool
	}{
		{"long enough", makeStateValue(false, d+time.Millisecond), true},
		{"too recent", makeStateValue(false, d-time.Millisecond), false},
		{"wrong current", makeStateValue(true, d+time.Millisecond), false},
		{"never false", StateValue{value: false}, false},
		{"exact boundary (strict <)", makeStateValue(false, d), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.sv.continuouslyFalse(d); got != tc.want {
				t.Errorf("continuouslyFalse = %v, want %v", got, tc.want)
			}
		})
	}
}

/*----------------------------------------------------------------------*/
/* 4.  RecentlyTrue / RecentlyFalse                                      */

func TestStateValue_RecentlyTrue(t *testing.T) {
	d := 5 * time.Second
	cases := []struct {
		name string
		sv   StateValue
		want bool
	}{
		{"shortcut true now", makeStateValueTrue(true, 10*time.Minute), true},
		{"inside window", makeStateValueTrue(false, d-time.Millisecond), true},
		{"outside window", makeStateValueTrue(false, d+time.Millisecond), false},
		{"never true", StateValue{}, false},
		{"exact boundary (inclusive ≥)", makeStateValueTrue(false, d), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.sv.recentlyTrue(d); got != tc.want {
				t.Errorf("recentlyTrue = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStateValue_RecentlyFalse(t *testing.T) {
	d := 5 * time.Second
	cases := []struct {
		name string
		sv   StateValue
		want bool
	}{
		{"shortcut false now", makeStateValueFalse(false, 10*time.Minute), true},
		{"inside window", makeStateValueFalse(true, d-time.Millisecond), true},
		{"outside window", makeStateValueFalse(true, d+time.Millisecond), false},
		{"never false", StateValue{}, false},
		{"exact boundary (inclusive ≥)", makeStateValueFalse(true, d), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.sv.recentlyFalse(d); got != tc.want {
				t.Errorf("recentlyFalse = %v, want %v", got, tc.want)
			}
		})
	}
}

/*----------------------------------------------------------------------*/
/* 5.  Edge‑case durations: zero & negative                              */

func TestEdgeDurations(t *testing.T) {
	zero := 0 * time.Second
	neg := -3 * time.Second

	stateValueTrue := makeStateValue(true, 2*time.Second)
	stateValueFalse := makeStateValue(false, 2*time.Second)

	cases := []struct {
		name string
		got  bool
		want bool
	}{
		// d==0 → continuously* true, recently* false
		{"contTrue d0", stateValueTrue.continuouslyTrue(zero), true},
		{"contFalse d0", stateValueFalse.continuouslyFalse(zero), true},
		{"recentTrue d0", stateValueTrue.recentlyTrue(zero), true},
		{"recentFalse d0", stateValueFalse.recentlyFalse(zero), true},

		{"contTrue reverse d0", stateValueTrue.continuouslyFalse(zero), false},
		{"contFalse reverse d0", stateValueFalse.continuouslyTrue(zero), false},
		{"recentTrue reverse d0", stateValueTrue.recentlyFalse(zero), false},
		{"recentFalse reverse d0", stateValueFalse.recentlyTrue(zero), false},

		// d<0 → cut in future: continuously* true, recently* false
		{"contTrue neg", stateValueTrue.continuouslyTrue(neg), true},
		{"contFalse neg", stateValueFalse.continuouslyFalse(neg), true},
		{"recentTrue neg", stateValueTrue.recentlyTrue(neg), true},
		{"recentFalse neg", stateValueFalse.recentlyFalse(neg), true},

		{"contTrue reverse neg", stateValueTrue.continuouslyFalse(neg), false},
		{"contFalse reverse neg", stateValueFalse.continuouslyTrue(neg), false},
		{"recentTrue reverse neg", stateValueTrue.recentlyFalse(neg), false},
		{"recentFalse reverse neg", stateValueFalse.recentlyTrue(neg), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("%s: got %v, want %v", tc.name, tc.got, tc.want)
			}
		})
	}
}

/*----------------------------------------------------------------------*/
/* 6.  StateValueMap adapter                                             */

func TestStateValueMap(t *testing.T) {
	key := StateKey("k")

	// empty map → all require… return false
	m := StateValueMap{svMap: map[StateKey]StateValue{}}
	if m.requireCurrentlyTrue(key) {
		t.Error("missing key → true")
	}
	if m.requireContinuouslyTrue(key, time.Second) {
		t.Error("missing key → true")
	}
	if m.requireRecentlyTrue(key, time.Second) {
		t.Error("missing key → true")
	}

	// now put a value
	m.svMap[key] = makeStateValueTrue(true, 10*time.Second)
	if !m.requireCurrentlyTrue(key) {
		t.Error("should be true now")
	}
	if !m.requireContinuouslyTrue(key, 5*time.Second) {
		t.Error("should be continuous true")
	}
	if m.requireContinuouslyFalse(key, 5*time.Second) {
		t.Error("should not be continuous false")
	}
	if !m.requireRecentlyTrue(key, 5*time.Second) {
		t.Error("should be recently true")
	}
}

/*----------------------------------------------------------------------*/
/* 7.  Symmetry                                                          */

func TestSymmetry(t *testing.T) {
	d := 3 * time.Second
	stateValue := makeStateValue(false, d+time.Second) // last false long ago

	if stateValue.continuouslyTrue(d) {
		t.Error("continuouslyTrue should be false")
	}
	if !stateValue.continuouslyFalse(d) {
		t.Error("continuouslyFalse should be true")
	}
	if stateValue.recentlyTrue(d) {
		t.Error("recentlyTrue should be false")
	}
	if !stateValue.recentlyFalse(d) {
		t.Error("recentlyFalse should be true")
	}
}
