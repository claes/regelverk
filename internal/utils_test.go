package regelverk

import (
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

// seedTrue simulates a true update at now-ago using the code's internal method
func seedTrue(m *StateValueMap, key StateKey, ago time.Duration) {
	origNowFunc := nowFunc()
	nowFunc = func() time.Time { return origNowFunc.Add(-ago) }
	m.updateStateUnsafe(key, true)
	nowFunc = func() time.Time { return origNowFunc }
}

// seedFalse simulates a false update at now-ago using the code's internal method
func seedFalse(m *StateValueMap, key StateKey, ago time.Duration) {
	origNowFunc := nowFunc()
	nowFunc = func() time.Time { return origNowFunc.Add(-ago) }
	m.updateStateUnsafe(key, false)
	nowFunc = func() time.Time { return origNowFunc }
}

func TestRequireCurrently(t *testing.T) {
	m := &StateValueMap{svMap: make(map[StateKey]StateValue)}
	key := StateKey("key1")

	// missing key
	if m.requireCurrentlyTrue(key) {
		t.Error("missing key: requireCurrentlyTrue should be false")
	}
	if m.requireCurrentlyFalse(key) {
		t.Error("missing key: requireCurrentlyFalse should be false")
	}

	// seed true
	seedTrue(m, key, 10*time.Second)
	if !m.requireCurrentlyTrue(key) {
		t.Error("true now: requireCurrentlyTrue should be true")
	}
	if m.requireCurrentlyFalse(key) {
		t.Error("true now: requireCurrentlyFalse should be false")
	}

	// seed false
	seedFalse(m, key, 5*time.Second)
	if m.requireCurrentlyTrue(key) {
		t.Error("false now: requireCurrentlyTrue should be false")
	}
	if !m.requireCurrentlyFalse(key) {
		t.Error("false now: requireCurrentlyFalse should be true")
	}
}

func TestRequireContinuously(t *testing.T) {
	m := &StateValueMap{svMap: make(map[StateKey]StateValue)}
	key := StateKey("key2")
	d := 5 * time.Second

	// missing
	if m.requireContinuouslyTrue(key, d) || m.requireContinuouslyFalse(key, d) {
		t.Error("missing key: continuously should be false")
	}

	// true long enough
	seedTrue(m, key, d+time.Millisecond)
	sv := m.svMap[key]
	if !m.requireContinuouslyTrue(key, d) || !sv.continuouslyTrue(d) {
		t.Error("true long enough: continuouslyTrue should be true")
	}
	if m.requireContinuouslyFalse(key, d) || sv.continuouslyFalse(d) {
		t.Error("true long enough: continuouslyFalse should be false")
	}

	// true too recent
	seedTrue(m, key, d-time.Millisecond)
	sv = m.svMap[key]
	if m.requireContinuouslyTrue(key, d) || sv.continuouslyTrue(d) {
		t.Error("true too recent: continuouslyTrue should be false")
	}

	// false long enough
	seedFalse(m, key, d+time.Millisecond)
	sv = m.svMap[key]
	if !m.requireContinuouslyFalse(key, d) || !sv.continuouslyFalse(d) {
		t.Error("false long enough: continuouslyFalse should be true")
	}
	if m.requireContinuouslyTrue(key, d) || sv.continuouslyTrue(d) {
		t.Error("false long enough: continuouslyTrue should be false")
	}

	// boundary strict
	seedTrue(m, key, d)
	sv = m.svMap[key]
	if m.requireContinuouslyTrue(key, d) || sv.continuouslyTrue(d) {
		t.Error("boundary true: continuouslyTrue should be false")
	}
	seedFalse(m, key, d)
	sv = m.svMap[key]
	if m.requireContinuouslyFalse(key, d) || sv.continuouslyFalse(d) {
		t.Error("boundary false: continuouslyFalse should be false")
	}
}

func TestRequireRecently(t *testing.T) {
	m := &StateValueMap{svMap: make(map[StateKey]StateValue)}
	key := StateKey("key3")
	d := 5 * time.Second

	// missing
	if m.requireRecentlyTrue(key, d) || m.requireRecentlyFalse(key, d) {
		t.Error("missing key: recently should be false")
	}

	// true now shortcut
	seedTrue(m, key, 10*time.Minute)
	sv := m.svMap[key]
	if !m.requireRecentlyTrue(key, d) || !sv.recentlyTrue(d) {
		t.Error("true now: recentlyTrue should be true")
	}
	if m.requireRecentlyFalse(key, d) || sv.recentlyFalse(d) {
		t.Error("true now: recentlyFalse should be false")
	}

	// inside window
	seedTrue(m, key, d-time.Millisecond)
	sv = m.svMap[key]
	if !m.requireRecentlyTrue(key, d) || !sv.recentlyTrue(d) {
		t.Error("inside window: recentlyTrue should be true")
	}

	// boundary inclusive
	seedTrue(m, key, d)
	sv = m.svMap[key]
	if !m.requireRecentlyTrue(key, d) || !sv.recentlyTrue(d) {
		t.Error("boundary true: recentlyTrue should be true")
	}

	// outside window (value still true) => should still be true
	seedTrue(m, key, d+time.Millisecond)
	sv = m.svMap[key]
	if !m.requireRecentlyTrue(key, d) || !sv.recentlyTrue(d) {
		t.Error("too old: recentlyTrue should be true due to current value")
	}

	// false now shortcut
	seedFalse(m, key, 10*time.Minute)
	sv = m.svMap[key]
	if !m.requireRecentlyFalse(key, d) || !sv.recentlyFalse(d) {
		t.Error("false now: recentlyFalse should be true")
	}
	if m.requireRecentlyTrue(key, d) || sv.recentlyTrue(d) {
		t.Error("false now: recentlyTrue should be false")
	}

	// inside window for false
	seedFalse(m, key, d-time.Millisecond)
	sv = m.svMap[key]
	if !m.requireRecentlyFalse(key, d) || !sv.recentlyFalse(d) {
		t.Error("inside window: recentlyFalse should be true")
	}

	// boundary inclusive for false
	seedFalse(m, key, d)
	sv = m.svMap[key]
	if !m.requireRecentlyFalse(key, d) || !sv.recentlyFalse(d) {
		t.Error("boundary false: recentlyFalse should be true")
	}

	// outside window for false (value still false) => should still be true
	seedFalse(m, key, d+time.Millisecond)
	sv = m.svMap[key]
	if !m.requireRecentlyFalse(key, d) || !sv.recentlyFalse(d) {
		t.Error("too old: recentlyFalse should be true due to current false")
	}
}

func TestRequireEdgeDurations(t *testing.T) {
	m := &StateValueMap{svMap: make(map[StateKey]StateValue)}
	key := StateKey("key4")
	zero := 0 * time.Second
	neg := -3 * time.Second

	// seed true 2s ago
	seedTrue(m, key, 2*time.Second)
	sv := m.svMap[key]
	// d==0
	if !m.requireContinuouslyTrue(key, zero) || !sv.continuouslyTrue(zero) {
		t.Error("edge d0 contTrue")
	}
	if !m.requireRecentlyTrue(key, zero) || !sv.recentlyTrue(zero) {
		t.Error("edge d0 recentTrue")
	}
	// d<0
	if !m.requireContinuouslyTrue(key, neg) || !sv.continuouslyTrue(neg) {
		t.Error("edge neg contTrue")
	}
	if !m.requireRecentlyTrue(key, neg) || !sv.recentlyTrue(neg) {
		t.Error("edge neg recentTrue should be true due to current value")
	}

	// seed false 2s ago
	seedFalse(m, key, 2*time.Second)
	sv = m.svMap[key]
	if !m.requireContinuouslyFalse(key, zero) || !sv.continuouslyFalse(zero) {
		t.Error("edge d0 contFalse")
	}
	if !m.requireRecentlyFalse(key, zero) || !sv.recentlyFalse(zero) {
		t.Error("edge d0 recentFalse")
	}
	// d<0
	if !m.requireContinuouslyFalse(key, neg) || !sv.continuouslyFalse(neg) {
		t.Error("edge neg contFalse")
	}
	if !m.requireRecentlyFalse(key, neg) || !sv.recentlyFalse(neg) {
		t.Error("edge neg recentFalse should be true due to current false")
	}
}

func TestRequireSymmetry(t *testing.T) {
	m := &StateValueMap{svMap: make(map[StateKey]StateValue)}
	key := StateKey("key5")
	d := 3 * time.Second

	// seed false long ago
	seedFalse(m, key, d+time.Second)
	sv := m.svMap[key]
	// symmetric expectations
	if m.requireContinuouslyTrue(key, d) || sv.continuouslyTrue(d) {
		t.Error("sym contTrue should be false")
	}
	if !m.requireContinuouslyFalse(key, d) || !sv.continuouslyFalse(d) {
		t.Error("sym contFalse should be true")
	}
	if m.requireRecentlyTrue(key, d) || sv.recentlyTrue(d) {
		t.Error("sym recentTrue should be false")
	}
	if !m.requireRecentlyFalse(key, d) || !sv.recentlyFalse(d) {
		t.Error("sym recentFalse should be true")
	}
}
