package regelverk

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	// Freeze nowFunc for deterministic tests
	fixed := time.Date(2025, 7, 27, 12, 0, 0, 0, time.UTC)
	nowFunc = func() time.Time { return fixed }
	os.Exit(m.Run())
}

// stateErrorString returns a formatted error message including the StateValue details.
// Timestamps printed in RFC3339Nano for clarity.
func stateErrorString(msg string, m *StateValueMap, key StateKey) string {
	sv, _ := m.getState(key)
	lastTrue := "<nil>"
	if !sv.lastSetTrue.IsZero() {
		lastTrue = sv.lastSetTrue.Format(time.RFC3339Nano)
	}
	lastFalse := "<nil>"
	if !sv.lastSetFalse.IsZero() {
		lastFalse = sv.lastSetFalse.Format(time.RFC3339Nano)
	}
	return fmt.Sprintf("%s; value=%v, lastSetTrue=%s, lastSetFalse=%s", msg, sv.value, lastTrue, lastFalse)
}

// seedTrue sets value true at (now - ago) via internal updateStateUnsafe
func seedTrue(m *StateValueMap, key StateKey, ago time.Duration) {
	base := nowFunc()
	nowFunc = func() time.Time { return base.Add(-ago) }
	m.updateStateUnsafe(key, true)
	nowFunc = func() time.Time { return base }
}

// seedFalse sets value false at (now - ago) via internal updateStateUnsafe
func seedFalse(m *StateValueMap, key StateKey, ago time.Duration) {
	base := nowFunc()
	nowFunc = func() time.Time { return base.Add(-ago) }
	m.updateStateUnsafe(key, false)
	nowFunc = func() time.Time { return base }
}

func TestRequireCurrently(t *testing.T) {
	m := NewStateValueMap()
	key := StateKey("cur")

	// missing key
	if m.requireCurrentlyTrue(key) {
		t.Error(stateErrorString("missing key: requireCurrentlyTrue should be false", &m, key))
	}
	if m.requireCurrentlyFalse(key) {
		t.Error(stateErrorString("missing key: requireCurrentlyFalse should be false", &m, key))
	}

	// seed true
	seedTrue(&m, key, 10*time.Second)
	if !m.requireCurrentlyTrue(key) {
		t.Error(stateErrorString("true now: requireCurrentlyTrue should be true", &m, key))
	}
	if m.requireCurrentlyFalse(key) {
		t.Error(stateErrorString("true now: requireCurrentlyFalse should be false", &m, key))
	}

	// seed false
	seedFalse(&m, key, 5*time.Second)
	if m.requireCurrentlyTrue(key) {
		t.Error(stateErrorString("false now: requireCurrentlyTrue should be false", &m, key))
	}
	if !m.requireCurrentlyFalse(key) {
		t.Error(stateErrorString("false now: requireCurrentlyFalse should be true", &m, key))
	}
}

func TestRequireContinuously(t *testing.T) {
	m := NewStateValueMap()
	key := StateKey("cont")
	d := 5 * time.Second

	// missing key
	if m.requireContinuouslyTrue(key, d) || m.requireContinuouslyFalse(key, d) {
		t.Error(stateErrorString("missing key: continuously should be false", &m, key))
	}

	// true long enough
	seedTrue(&m, key, d+time.Millisecond)
	sv, _ := m.getState(key)
	if !m.requireContinuouslyTrue(key, d) || !sv.continuouslyTrue(d) {
		t.Error(stateErrorString("true long enough: continuouslyTrue should be true", &m, key))
	}
	if m.requireContinuouslyFalse(key, d) || sv.continuouslyFalse(d) {
		t.Error(stateErrorString("true long enough: continuouslyFalse should be false", &m, key))
	}

	// call to set true again without a false in between
	seedTrue(&m, key, d-time.Millisecond)
	sv, _ = m.getState(key)
	if !m.requireContinuouslyTrue(key, d) || !sv.continuouslyTrue(d) {
		t.Error(stateErrorString("true too recent: continuouslyTrue should be false", &m, key))
	}

	// false long enough
	seedFalse(&m, key, d+time.Millisecond)
	sv, _ = m.getState(key)
	if !m.requireContinuouslyFalse(key, d) || !sv.continuouslyFalse(d) {
		t.Error(stateErrorString("false long enough: continuouslyFalse should be true", &m, key))
	}
	if m.requireContinuouslyTrue(key, d) || sv.continuouslyTrue(d) {
		t.Error(stateErrorString("false long enough: continuouslyTrue should be false", &m, key))
	}

	// boundary strict
	seedTrue(&m, key, d)
	sv, _ = m.getState(key)
	if m.requireContinuouslyTrue(key, d) || sv.continuouslyTrue(d) {
		t.Error(stateErrorString("boundary true: continuouslyTrue should be false", &m, key))
	}
	seedFalse(&m, key, d)
	sv, _ = m.getState(key)
	if m.requireContinuouslyFalse(key, d) || sv.continuouslyFalse(d) {
		t.Error(stateErrorString("boundary false: continuouslyFalse should be false", &m, key))
	}
}

func TestRequireRecently(t *testing.T) {
	m := NewStateValueMap()
	key := StateKey("recent")
	d := 5 * time.Second

	// missing key
	if m.requireRecentlyTrue(key, d) || m.requireRecentlyFalse(key, d) {
		t.Error(stateErrorString("missing key: recently should be false", &m, key))
	}

	// true now shortcut
	seedTrue(&m, key, 10*time.Minute)
	sv, _ := m.getState(key)
	if !m.requireRecentlyTrue(key, d) || !sv.recentlyTrue(d) {
		t.Error(stateErrorString("true now: recentlyTrue should be true", &m, key))
	}
	if m.requireRecentlyFalse(key, d) || sv.recentlyFalse(d) {
		t.Error(stateErrorString("true now: recentlyFalse should be false", &m, key))
	}

	// inside window
	seedTrue(&m, key, d-time.Millisecond)
	sv, _ = m.getState(key)
	if !m.requireRecentlyTrue(key, d) || !sv.recentlyTrue(d) {
		t.Error(stateErrorString("inside window: recentlyTrue should be true", &m, key))
	}

	// boundary inclusive
	seedTrue(&m, key, d)
	sv, _ = m.getState(key)
	if !m.requireRecentlyTrue(key, d) || !sv.recentlyTrue(d) {
		t.Error(stateErrorString("boundary true: recentlyTrue should be true", &m, key))
	}

	// outside window (value still true)
	seedTrue(&m, key, d+time.Millisecond)
	sv, _ = m.getState(key)
	if !m.requireRecentlyTrue(key, d) || !sv.recentlyTrue(d) {
		t.Error(stateErrorString("too old: recentlyTrue should be true due to current value", &m, key))
	}
}

func TestRequireEdgeDurations(t *testing.T) {
	m := NewStateValueMap()
	key := StateKey("edge")
	zero := 0 * time.Second
	neg := -3 * time.Second

	// seed true 2s ago
	seedTrue(&m, key, 2*time.Second)
	sv, _ := m.getState(key)
	if !m.requireContinuouslyTrue(key, zero) || !sv.continuouslyTrue(zero) {
		t.Error(stateErrorString("edge d0 contTrue", &m, key))
	}
	if !m.requireRecentlyTrue(key, zero) || !sv.recentlyTrue(zero) {
		t.Error(stateErrorString("edge d0 recentTrue", &m, key))
	}
	if !m.requireContinuouslyTrue(key, neg) || !sv.continuouslyTrue(neg) {
		t.Error(stateErrorString("edge neg contTrue", &m, key))
	}
	if !m.requireRecentlyTrue(key, neg) || !sv.recentlyTrue(neg) {
		t.Error(stateErrorString("edge neg recentTrue should be true due to current value", &m, key))
	}

	// seed false 2s ago
	seedFalse(&m, key, 2*time.Second)
	sv, _ = m.getState(key)
	if !m.requireContinuouslyFalse(key, zero) || !sv.continuouslyFalse(zero) {
		t.Error(stateErrorString("edge d0 contFalse", &m, key))
	}
	if !m.requireRecentlyFalse(key, zero) || !sv.recentlyFalse(zero) {
		t.Error(stateErrorString("edge d0 recentFalse", &m, key))
	}
	if !m.requireContinuouslyFalse(key, neg) || !sv.continuouslyFalse(neg) {
		t.Error(stateErrorString("edge neg contFalse", &m, key))
	}
	if !m.requireRecentlyFalse(key, neg) || !sv.recentlyFalse(neg) {
		t.Error(stateErrorString("edge neg recentFalse should be true due to current false", &m, key))
	}
}

func TestRequireSymmetry(t *testing.T) {
	m := NewStateValueMap()
	key := StateKey("sym")
	d := 3 * time.Second

	seedFalse(&m, key, d+time.Second)
	sv, _ := m.getState(key)
	if m.requireContinuouslyTrue(key, d) || sv.continuouslyTrue(d) {
		t.Error(stateErrorString("sym contTrue should be false", &m, key))
	}
	if !m.requireContinuouslyFalse(key, d) || !sv.continuouslyFalse(d) {
		t.Error(stateErrorString("sym contFalse should be true", &m, key))
	}
	if m.requireRecentlyTrue(key, d) || sv.recentlyTrue(d) {
		t.Error(stateErrorString("sym recentTrue should be false", &m, key))
	}
	if !m.requireRecentlyFalse(key, d) || !sv.recentlyFalse(d) {
		t.Error(stateErrorString("sym recentFalse should be true", &m, key))
	}
}
