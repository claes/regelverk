package regelverk

import (
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/sj14/astral/pkg/astral"
)

type TimeOfDay int

const (
	Nighttime TimeOfDay = iota
	MorningAstronomicalTwilight
	MorningNauticalTwilight
	MorningCivilTwilight
	Daytime
	EveningCivilTwilight
	EveningNauticalTwilight
	EveningAstronomcialTwilight
)

func (t TimeOfDay) String() string {
	switch t {
	case Nighttime:
		return "Nighttime"
	case MorningAstronomicalTwilight:
		return "Morning Astronomical Twilight"
	case MorningNauticalTwilight:
		return "Morning Nautical Twilight"
	case MorningCivilTwilight:
		return "Morning Civil Twilight"
	case Daytime:
		return "Daytime"
	case EveningCivilTwilight:
		return "Evening Civil Twilight"
	case EveningNauticalTwilight:
		return "Evening Nautical Twilight"
	case EveningAstronomcialTwilight:
		return "Evening Astronomical Twilight"
	default:
		return "Unknown TimeOfDay"
	}
}

func ComputeTimeOfDay(currentTime time.Time, lat, long float64) TimeOfDay {

	observer := astral.Observer{
		Latitude:  lat,
		Longitude: long,
		Elevation: 0.0,
	}

	location := currentTime.Location()
	localMidnight := time.Date(
		currentTime.Year(),
		currentTime.Month(),
		currentTime.Day(),
		0, 0, 0, 0,
		location,
	)

	midnight := localMidnight
	nextMidnight := midnight.Add(24 * time.Hour)

	startAstronomicalTwilight, _ := astral.Dawn(observer, midnight, astral.DepressionAstronomical)
	startNauticalTwilight, _ := astral.Dawn(observer, midnight, astral.DepressionNautical)
	startCivilTwilight, _ := astral.Dawn(observer, midnight, astral.DepressionCivil)
	sunrise, _ := astral.Sunrise(observer, midnight)
	sunset, _ := astral.Sunset(observer, midnight)
	endCivilTwilight, _ := astral.Dusk(observer, midnight, astral.DepressionCivil)
	endNauticalTwilight, _ := astral.Dusk(observer, midnight, astral.DepressionNautical)
	endAstronomicalTwilight, _ := astral.Dusk(observer, midnight, astral.DepressionAstronomical)

	var phase TimeOfDay
	switch {
	case currentTime.After(midnight) && currentTime.Before(startAstronomicalTwilight):
		phase = Nighttime
	case currentTime.After(startAstronomicalTwilight) && currentTime.Before(startNauticalTwilight):
		phase = MorningAstronomicalTwilight
	case currentTime.After(startNauticalTwilight) && currentTime.Before(startCivilTwilight):
		phase = MorningNauticalTwilight
	case currentTime.After(startCivilTwilight) && currentTime.Before(sunrise):
		phase = MorningCivilTwilight
	case currentTime.After(sunrise) && currentTime.Before(sunset):
		phase = Daytime
	case currentTime.After(sunset) && currentTime.Before(endCivilTwilight):
		phase = EveningCivilTwilight
	case currentTime.After(endCivilTwilight) && currentTime.Before(endNauticalTwilight):
		phase = EveningNauticalTwilight
	case currentTime.After(endNauticalTwilight) && currentTime.Before(endAstronomicalTwilight):
		phase = EveningAstronomcialTwilight
	case currentTime.After(endAstronomicalTwilight) && currentTime.Before(nextMidnight):
		phase = Nighttime
	}

	// fmt.Printf("Astronomical twilight start %v\n", startAstronomicalTwilight.In(location))
	// fmt.Printf("Nautical twilight start %v\n", startNauticalTwilight.In(location))
	// fmt.Printf("Civil twilight start %v\n", startCivilTwilight.In(location))
	// fmt.Printf("Sunrise, day start %v\n", sunrise.In(location))
	// fmt.Printf("Sunset %v\n", sunset.In(location))
	// fmt.Printf("Civil twilight end %v\n", endCivilTwilight.In(location))
	// fmt.Printf("Nautical twilight end %v\n", endNauticalTwilight.In(location))
	// fmt.Printf("Astronomical twilight end, night start %v\n", endAstronomicalTwilight.In(location))
	// fmt.Printf("Phase of the day for %v is %s\n", currentTime.In(location).Format("2006-01-02 15:04:05 MST"), phase)
	return phase
}

func foo() {
	today := time.Now()
	location, _ := time.LoadLocation("CET")
	midnight := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, location)

	for i := 0; i < 24*4; i++ {
		currentHour := midnight.Add(time.Duration(i*15) * time.Minute)
		timeOfDay := ComputeTimeOfDay(currentHour, 59, 18)
		fmt.Printf("Phase of the day for %v is %s\n", currentHour.In(location).Format("2006-01-02 15:04:05 MST"), timeOfDay)
	}
}

type StateKey string

const (
	NoKey StateKey = ""
)

type StateValue struct {
	value        bool
	isDefined    bool
	lastUpdate   time.Time // Last time this state was updated (incl refreshed even if value was not changed)
	lastChange   time.Time // Last time the state was changed (value was changed differently than before)
	lastSetTrue  time.Time
	lastSetFalse time.Time
}

type StateValueMap struct {
	svMap             map[StateKey]StateValue
	mu                sync.RWMutex
	observerCallbacks []func(key StateKey, value, new, updated bool)
	mutatorCallbacks  []func(key StateKey) (StateKey, bool)
}

func NewStateValueMap() StateValueMap {
	return StateValueMap{
		svMap: make(map[StateKey]StateValue),
	}
}

func (s *StateValueMap) registerObserverCallback(callback func(key StateKey, value, new, updated bool)) {
	s.observerCallbacks = append(s.observerCallbacks, callback)
}

func (s *StateValueMap) registerMutatorCallback(callback func(key StateKey) (StateKey, bool)) {
	s.mutatorCallbacks = append(s.mutatorCallbacks, callback)
}

func (s *StateValueMap) setStateValue(key StateKey, value StateValue) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.svMap[key] = value
}

func (s *StateValueMap) setState(key StateKey, value bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.updateStateUnsafe(key, value)

	for _, callback := range s.mutatorCallbacks {
		dependentKey, associatedValue := callback(key)
		s.updateStateUnsafe(dependentKey, associatedValue)
	}
}

// Don't call this from outside, use setState instead
func (s *StateValueMap) updateStateUnsafe(key StateKey, value bool) {

	if key == NoKey {
		return
	}

	existingState, exists := s.svMap[key]

	now := time.Now()
	var updatedState StateValue
	stateNew := false
	stateUpdate := false
	if exists {
		if existingState.value == value {
			// don't change value
		} else {
			existingState.value = value
			existingState.lastChange = now
			stateUpdate = true
		}
		existingState.lastUpdate = now
		updatedState = existingState
	} else {
		// Not exists
		updatedState = StateValue{
			value:      value,
			isDefined:  true,
			lastUpdate: now,
			lastChange: now,
		}
		stateNew = true
	}

	if value {
		updatedState.lastSetTrue = now
	} else {
		updatedState.lastSetFalse = now
	}

	for _, callback := range s.observerCallbacks {
		callback(key, value, stateNew, stateUpdate)
	}

	s.svMap[key] = updatedState
}

func (s *StateValueMap) getState(key StateKey) (StateValue, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stateValue, exists := s.svMap[key]
	stateValue.isDefined = exists
	return stateValue, exists
}

func (s *StateValueMap) requireTrue(key StateKey) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stateValue, exists := s.svMap[key]
	if !exists {
		return false
	} else {
		return stateValue.requireTrue()
	}
}

func (s *StateValueMap) requireFalse(key StateKey) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stateValue, exists := s.svMap[key]
	if !exists {
		return false
	} else {
		return stateValue.requireFalse()
	}
}

// Require it has consistently been true
func (s *StateValueMap) requireTrueSince(key StateKey, duration time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stateValue, exists := s.svMap[key]
	if !exists {
		return false
	} else {
		return stateValue.requireTrueSince(duration)
	}
}

// Require it has been true at some point during duration
func (s *StateValueMap) requireTrueRecently(key StateKey, duration time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stateValue, exists := s.svMap[key]
	if !exists {
		return false
	} else {
		return stateValue.requireTrueRecently(duration)
	}
}

// Require that it must not have been true at any point during duration
func (s *StateValueMap) requireTrueNotRecently(key StateKey, duration time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stateValue, exists := s.svMap[key]
	if !exists {
		return false
	} else {
		return stateValue.requireTrueNotRecently(duration)
	}
}

// Require it has consistently been false
func (s *StateValueMap) requireFalseSince(key StateKey, duration time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stateValue, exists := s.svMap[key]
	if !exists {
		return false
	}
	return stateValue.requireFalseSince(duration)
}

// Require it has been false at some point during duration
func (s *StateValueMap) requireFalseRecently(key StateKey, duration time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stateValue, exists := s.svMap[key]
	if !exists {
		return false
	}
	return stateValue.requireFalseRecently(duration)
}

// Require that it must not have been false at any point during duration
func (s *StateValueMap) requireFalseNotRecently(key StateKey, duration time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stateValue, exists := s.svMap[key]
	if !exists {
		return false
	}
	return stateValue.requireFalseNotRecently(duration)
}

func (stateValue *StateValue) requireTrue() bool {
	return stateValue.value
}

func (stateValue *StateValue) requireFalse() bool {
	return !stateValue.value
}

func (stateValue *StateValue) requireTrueSince(duration time.Duration) bool {
	return stateValue.value && time.Since(stateValue.lastSetTrue) > duration
}

func (stateValue *StateValue) requireTrueRecently(duration time.Duration) bool {
	return stateValue.value || time.Since(stateValue.lastSetTrue) < duration
}

func (stateValue *StateValue) requireTrueNotRecently(duration time.Duration) bool {
	return !stateValue.value && time.Since(stateValue.lastSetTrue) > duration
}

func (stateValue *StateValue) requireFalseSince(duration time.Duration) bool {
	return !stateValue.value && time.Since(stateValue.lastSetFalse) > duration
}

func (stateValue *StateValue) requireFalseNotRecently(duration time.Duration) bool {
	return stateValue.value && time.Since(stateValue.lastSetFalse) > duration
}

func (stateValue *StateValue) requireFalseRecently(duration time.Duration) bool {
	return !stateValue.value || time.Since(stateValue.lastSetFalse) < duration
}

// ContinuouslyTrue reports whether the signal has been true
// for the entire interval (now−d , now].
func (s *StateValue) ContinuouslyTrue(d time.Duration) bool {
	if !s.value || s.lastSetTrue.IsZero() {
		return false
	}
	cut := time.Now().Add(-d)
	return s.lastSetTrue.Before(cut) // state switched to true strictly before window start
}

// ContinuouslyFalse is the dual of ContinuouslyTrue.
func (s *StateValue) ContinuouslyFalse(d time.Duration) bool {
	if s.value || s.lastSetFalse.IsZero() {
		return false
	}
	cut := time.Now().Add(-d)
	return s.lastSetFalse.Before(cut)
}

// AnyTrue reports whether the signal was ever true in (now−d , now].
func (s *StateValue) AnyTrue(d time.Duration) bool {
	if s.lastSetTrue.IsZero() {
		return false
	}
	cut := time.Now().Add(-d)
	return !s.lastSetTrue.Before(cut) // lastSetTrue ≥ cut
}

// AnyFalse is the dual of AnyTrue.
func (s *StateValue) AnyFalse(d time.Duration) bool {
	if s.lastSetFalse.IsZero() {
		return false
	}
	cut := time.Now().Add(-d)
	return !s.lastSetFalse.Before(cut)
}

// Negations for convenience.
func (s *StateValue) NoTrue(d time.Duration) bool {
	return !s.AnyTrue(d)
}

func (s *StateValue) NoFalse(d time.Duration) bool {
	return !s.AnyFalse(d)
}

func (s *StateValueMap) LogState() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()

	var params [][]any
	for key, stateValue := range s.svMap {

		secondsSinceLastUpdate := int64(-1)
		if !stateValue.lastUpdate.IsZero() {
			secondsSinceLastUpdate = int64(now.Sub(stateValue.lastUpdate).Seconds())
		}

		secondsSinceLastSetTrue := int64(-1)
		if !stateValue.lastSetTrue.IsZero() {
			secondsSinceLastSetTrue = int64(now.Sub(stateValue.lastSetTrue).Seconds())
		}

		secondsSinceLastSetFalse := int64(-1)
		if !stateValue.lastSetFalse.IsZero() {
			secondsSinceLastSetFalse = int64(now.Sub(stateValue.lastSetFalse).Seconds())
		}

		secondsSinceLastChange := int64(-1)
		if !stateValue.lastChange.IsZero() {
			secondsSinceLastChange = int64(now.Sub(stateValue.lastChange).Seconds())
		}

		params = append(params, []any{"key", key,
			"value", stateValue.value,
			"isDefined", stateValue.isDefined,
			"lastUpdate", stateValue.lastUpdate,
			"secondsSinceLastUpdate", secondsSinceLastUpdate,
			"lastChange", stateValue.lastChange,
			"secondsSinceLastChange", secondsSinceLastChange,
			"lastSetTrue", stateValue.lastSetTrue,
			"secondsSinceLastSetTrue", secondsSinceLastSetTrue,
			"lastSetFalse", stateValue.lastSetFalse,
			"secondsSinceLastSetFalse", secondsSinceLastSetFalse})
	}

	sort.Slice(params, func(i, j int) bool {
		return params[i][1].(string) < params[j][1].(string)
	})

	for _, p := range params {
		slog.Info("StateValue entry", p...)
	}
}
