package regelverk

import (
	"testing"
	"time"
)

// TestRequireTrue verifies the behavior of requireTrue
func TestRequireTrue(t *testing.T) {
	// Initialize StateValueMap
	stateMap := NewStateValueMap()

	// Test with no keys set
	if result := stateMap.requireTrue("nonexistent"); result != false {
		t.Errorf("Expected false for nonexistent key, got %v", result)
	}

	// Test with a key set to true
	stateMap.setState("key1", true)
	if result := stateMap.requireTrue("key1"); result != true {
		t.Errorf("Expected true for key1, got %v", result)
	}

	// Test with a key set to false
	stateMap.setState("key2", false)
	if result := stateMap.requireTrue("key2"); result != false {
		t.Errorf("Expected false for key2, got %v", result)
	}
}

// TestConcurrency ensures requireTrue works correctly under concurrent access
func TestRequireTrue_Concurrency(t *testing.T) {
	stateMap := NewStateValueMap()

	// Initialize keys in a separate goroutine
	go func() {
		stateMap.setState("key1", true)
		stateMap.setState("key2", false)
	}()

	// Allow goroutine to execute
	time.Sleep(10 * time.Millisecond)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			if result := stateMap.requireTrue("key1"); !result {
				t.Errorf("Concurrent access: expected true for key1, got %v", result)
			}
			if result := stateMap.requireTrue("key2"); result {
				t.Errorf("Concurrent access: expected false for key2, got %v", result)
			}
		}()
	}
}

// TestRequireTrueSince validates the behavior of requireTrueSince
func TestRequireTrueSince(t *testing.T) {
	stateMap := NewStateValueMap()

	// Test with no key set
	if result := stateMap.requireTrueSince("nonexistent", 1*time.Second); result != false {
		t.Errorf("Expected false for nonexistent key, got %v", result)
	}

	// Set a key to true and validate immediately
	stateMap.setState("key1", true)
	if result := stateMap.requireTrueSince("key1", 1*time.Second); result != false {
		t.Errorf("Expected false for key1 as not enough time has passed, got %v", result)
	}

	// Wait for the duration to pass and validate again
	time.Sleep(2 * time.Second)
	if result := stateMap.requireTrueSince("key1", 1*time.Second); result != true {
		t.Errorf("Expected true for key1 after sufficient time has passed, got %v", result)
	}

	// Change key1 to false and validate
	stateMap.setState("key1", false)
	if result := stateMap.requireTrueSince("key1", 1*time.Second); result != false {
		t.Errorf("Expected false for key1 after being set to false, got %v", result)
	}

	// Set a key to false initially and validate
	stateMap.setState("key2", false)
	if result := stateMap.requireTrueSince("key2", 1*time.Second); result != false {
		t.Errorf("Expected false for key2 initially set to false, got %v", result)
	}

	// Set key2 to true and validate after some delay
	stateMap.setState("key2", true)
	time.Sleep(500 * time.Millisecond)
	if result := stateMap.requireTrueSince("key2", 1*time.Second); result != false {
		t.Errorf("Expected false for key2 as not enough time has passed, got %v", result)
	}

	time.Sleep(1 * time.Second)
	if result := stateMap.requireTrueSince("key2", 1*time.Second); result != true {
		t.Errorf("Expected true for key2 after sufficient time has passed, got %v", result)
	}
}

// TestRequireTrueSinceEdgeCases tests edge cases for requireTrueSince
func TestRequireTrueSinceEdgeCases(t *testing.T) {
	stateMap := NewStateValueMap()

	// Set key to true and check at the exact threshold
	stateMap.setState("key3", true)
	time.Sleep(1 * time.Second)
	if result := stateMap.requireTrueSince("key3", 1*time.Second); result != true {
		t.Errorf("Expected true for key3 at the exact threshold, got %v", result)
	}

	// Check with zero duration
	if result := stateMap.requireTrueSince("key3", 0*time.Second); result != true {
		t.Errorf("Expected true for key3 with zero duration, got %v", result)
	}

	// Check with negative duration
	if result := stateMap.requireTrueSince("key3", -1*time.Second); result != true {
		t.Errorf("Expected true for key3 with negative duration, got %v", result)
	}
}

func TestRequireTrueRecently(t *testing.T) {
	stateMap := NewStateValueMap()

	// Test with no key set
	if result := stateMap.requireTrueRecently("nonexistent", 1*time.Second); result != false {
		t.Errorf("Expected false for nonexistent key, got %v", result)
	}

	stateMap.setState("key1", true)
	if result := stateMap.requireTrueRecently("key1", 1*time.Second); result != true {
		t.Errorf("Expected true for key1 immediately after being set, got %v", result)
	}

	time.Sleep(2 * time.Second)
	stateMap.setState("key1", false)
	stateMap.setState("key1", true)
	time.Sleep(1 * time.Second)
	stateMap.setState("key1", false)
	if result := stateMap.requireTrueRecently("key1", 2*time.Second); result != true {
		t.Errorf("Expected true for key1 , got %v", result)
	}

	stateMap.setState("key1", false)
	stateMap.setState("key1", true)
	time.Sleep(1 * time.Second)
	stateMap.setState("key1", false)
	time.Sleep(2 * time.Second)
	if result := stateMap.requireTrueRecently("key1", 1*time.Second); result != false {
		t.Errorf("Expected false for key1 , got %v", result)
	}
}

func TestRequireTrueRecentlyEdgeCases(t *testing.T) {
	stateMap := NewStateValueMap()

	// Set a key to true and validate at the exact threshold
	stateMap.setState("key3", true)
	time.Sleep(1 * time.Second)
	if result := stateMap.requireTrueRecently("key3", 1*time.Second); result != true {
		t.Errorf("Expected true for key3 at the exact threshold, got %v", result)
	}

	// Check behavior with zero duration
	if result := stateMap.requireTrueRecently("key3", 0*time.Second); result != true {
		t.Errorf("Expected true for key3 with zero duration, got %v", result)
	}

	// Check behavior with a negative duration
	if result := stateMap.requireTrueRecently("key3", -1*time.Second); result != true {
		t.Errorf("Expected true for key3 with negative duration, got %v", result)
	}

	// Reset the key to false and validate
	stateMap.setState("key3", false)
	if result := stateMap.requireTrueRecently("key3", 1*time.Second); result != false {
		t.Errorf("Expected false for key3 after being reset to false, got %v", result)
	}
}

func TestRequireTrueNotRecently(t *testing.T) {
	stateMap := NewStateValueMap()

	// Test with no key set
	if result := stateMap.requireTrueRecently("nonexistent", 1*time.Second); result != false {
		t.Errorf("Expected false for nonexistent key, got %v", result)
	}

	stateMap.setState("key1", true)
	if result := stateMap.requireTrueNotRecently("key1", 1*time.Second); result != false {
		t.Errorf("Expected false for key1 immediately after being set, got %v", result)
	}

	time.Sleep(2 * time.Second)
	stateMap.setState("key1", false)
	stateMap.setState("key1", true)
	time.Sleep(1 * time.Second)
	stateMap.setState("key1", false)
	if result := stateMap.requireTrueNotRecently("key1", 2*time.Second); result != false {
		t.Errorf("Expected false for key1 , got %v", result)
	}

	stateMap.setState("key1", false)
	stateMap.setState("key1", true)
	time.Sleep(1 * time.Second)
	stateMap.setState("key1", false)
	time.Sleep(2 * time.Second)
	if result := stateMap.requireTrueNotRecently("key1", 1*time.Second); result != true {
		t.Errorf("Expected true for key1 , got %v", result)
	}

}

//

// TestRequireTrueRecently tests the requireTrueRecently method.
func TestRequireTrueRecentlyStateValue(t *testing.T) {
	base := time.Now()
	tests := []struct {
		name     string
		state    StateValue
		duration time.Duration
		want     bool
	}{
		{
			name:     "Currently true",
			state:    StateValue{value: true, lastSetTrue: base.Add(-10 * time.Minute)},
			duration: 5 * time.Minute,
			want:     true,
		},
		{
			name:     "Was true recently, now false",
			state:    StateValue{value: false, lastSetTrue: base.Add(-3 * time.Minute)},
			duration: 5 * time.Minute,
			want:     true,
		},
		{
			name:     "Was true long ago, now false",
			state:    StateValue{value: false, lastSetTrue: base.Add(-10 * time.Minute)},
			duration: 5 * time.Minute,
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.requireTrueRecently(tt.duration); got != tt.want {
				t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestRequireFalseRecently tests the requireFalseRecently method.
func TestRequireFalseRecentlyStateValue(t *testing.T) {
	base := time.Now()
	tests := []struct {
		name     string
		state    StateValue
		duration time.Duration
		want     bool
	}{
		{
			name:     "Currently false",
			state:    StateValue{value: false, lastSetFalse: base.Add(-10 * time.Minute)},
			duration: 5 * time.Minute,
			want:     true,
		},
		{
			name:     "Was false recently, now true",
			state:    StateValue{value: true, lastSetFalse: base.Add(-3 * time.Minute)},
			duration: 5 * time.Minute,
			want:     true,
		},
		{
			name:     "Was false long ago, now true",
			state:    StateValue{value: true, lastSetFalse: base.Add(-10 * time.Minute)},
			duration: 5 * time.Minute,
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.requireFalseRecently(tt.duration); got != tt.want {
				t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestRequireTrueNotRecently tests the requireTrueNotRecently method.
func TestRequireTrueNotRecentlyStateValue(t *testing.T) {
	base := time.Now()
	tests := []struct {
		name     string
		state    StateValue
		duration time.Duration
		want     bool
	}{
		{
			name:     "Currently false, set true long ago",
			state:    StateValue{value: false, lastSetTrue: base.Add(-10 * time.Minute)},
			duration: 5 * time.Minute,
			want:     true,
		},
		{
			name:     "Currently false, set true recently",
			state:    StateValue{value: false, lastSetTrue: base.Add(-3 * time.Minute)},
			duration: 5 * time.Minute,
			want:     false,
		},
		{
			name:     "Currently true",
			state:    StateValue{value: true, lastSetTrue: base.Add(-10 * time.Minute)},
			duration: 5 * time.Minute,
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.requireTrueNotRecently(tt.duration); got != tt.want {
				t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestRequireFalseNotRecently tests the requireFalseNotRecently method.
func TestRequireFalseNotRecentlyStateValue(t *testing.T) {
	base := time.Now()
	tests := []struct {
		name     string
		state    StateValue
		duration time.Duration
		want     bool
	}{
		{
			name:     "Currently true, set false long ago",
			state:    StateValue{value: true, lastSetFalse: base.Add(-10 * time.Minute)},
			duration: 5 * time.Minute,
			want:     true,
		},
		{
			name:     "Currently true, set false recently",
			state:    StateValue{value: true, lastSetFalse: base.Add(-3 * time.Minute)},
			duration: 5 * time.Minute,
			want:     false,
		},
		{
			name:     "Currently false",
			state:    StateValue{value: false, lastSetFalse: base.Add(-10 * time.Minute)},
			duration: 5 * time.Minute,
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.requireFalseNotRecently(tt.duration); got != tt.want {
				t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestRequireTrueSince tests the requireTrueSince method.
func TestRequireTrueSinceStateValue(t *testing.T) {
	base := time.Now()
	tests := []struct {
		name     string
		state    StateValue
		duration time.Duration
		want     bool
	}{
		{
			name:     "Currently true, set true long ago",
			state:    StateValue{value: true, lastSetTrue: base.Add(-10 * time.Minute)},
			duration: 5 * time.Minute,
			want:     true,
		},
		{
			name:     "Currently true, set true recently",
			state:    StateValue{value: true, lastSetTrue: base.Add(-3 * time.Minute)},
			duration: 5 * time.Minute,
			want:     false,
		},
		{
			name:     "Currently false",
			state:    StateValue{value: false, lastSetTrue: base.Add(-10 * time.Minute)},
			duration: 5 * time.Minute,
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.requireTrueSince(tt.duration); got != tt.want {
				t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestRequireTrueAndRequireFalse tests the requireTrue and requireFalse methods.
func TestRequireTrueAndRequireFalse(t *testing.T) {
	tests := []struct {
		name      string
		state     StateValue
		wantTrue  bool
		wantFalse bool
	}{
		{"Value true", StateValue{value: true}, true, false},
		{"Value false", StateValue{value: false}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name+"/True", func(t *testing.T) {
			if got := tt.state.requireTrue(); got != tt.wantTrue {
				t.Errorf("%s: requireTrue got %v, want %v", tt.name, got, tt.wantTrue)
			}
		})
		t.Run(tt.name+"/False", func(t *testing.T) {
			if got := tt.state.requireFalse(); got != tt.wantFalse {
				t.Errorf("%s: requireFalse got %v, want %v", tt.name, got, tt.wantFalse)
			}
		})
	}
}
