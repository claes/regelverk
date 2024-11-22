package regelverk

import (
	"fmt"
	"log/slog"
	"sort"
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

type StateValue struct {
	value        bool
	isDefined    bool
	lastUpdate   time.Time
	lastSetTrue  time.Time
	lastSetFalse time.Time
}

func (f StateValue) Age() time.Duration {
	return time.Since(f.lastUpdate)
}

type StateValueMap struct {
	stateValueMap map[string]StateValue
}

func NewStateValueMap() StateValueMap {
	return StateValueMap{
		stateValueMap: make(map[string]StateValue),
	}
}
func (s *StateValueMap) setState(key string, value bool) {
	existingState, exists := s.stateValueMap[key]

	if !exists || existingState.value != value {
		newState := StateValue{
			value:        value,
			isDefined:    true,
			lastUpdate:   time.Now(),
			lastSetTrue:  existingState.lastSetTrue,
			lastSetFalse: existingState.lastSetFalse,
		}
		now := time.Now()
		if value {
			newState.lastSetTrue = now
		} else {
			newState.lastSetFalse = now
		}
		s.stateValueMap[key] = newState
	}
}

func (s *StateValueMap) getState(key string) StateValue {
	stateValue, exists := s.stateValueMap[key]
	stateValue.isDefined = exists
	return stateValue
}

func (s *StateValueMap) require(key string) bool {
	stateValue, exists := s.stateValueMap[key]
	if !exists {
		return false
	} else {
		return stateValue.value
	}
}

func (s *StateValueMap) requireNot(key string) bool {
	stateValue, exists := s.stateValueMap[key]
	if !exists {
		return false
	} else {
		return !stateValue.value
	}
}

func (s *StateValueMap) requireRecently(key string, duration time.Duration) bool {
	stateValue, exists := s.stateValueMap[key]
	if !exists {
		return false
	} else {
		return stateValue.value || time.Since(stateValue.lastSetTrue) < duration
	}
}

func (s *StateValueMap) requireNotRecently(key string, duration time.Duration) bool {
	stateValue, exists := s.stateValueMap[key]
	if !exists {
		return false
	} else {
		return !stateValue.value && time.Since(stateValue.lastSetTrue) > duration
	}
}

func (s *StateValueMap) LogState() {
	now := time.Now()

	var params [][]any
	for key, stateValue := range s.stateValueMap {

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
		params = append(params, []any{"key", key,
			"value", stateValue.value,
			"isDefined", stateValue.isDefined,
			"lastUpdate", stateValue.lastUpdate,
			"secondsSinceLastUpdate", secondsSinceLastUpdate,
			"lastSetTrue", stateValue.lastSetTrue,
			"secondsSinceLastSetTrue", secondsSinceLastSetTrue,
			"lastSetFalse", stateValue.lastSetFalse,
			"secondsSinceLastSetFalse", secondsSinceLastSetFalse})
	}

	sort.Slice(params, func(i, j int) bool {
		return params[i][1].(string) < params[j][1].(string)
	})

	for _, p := range params {
		slog.Debug("StateValue entry", p...)
	}
}
