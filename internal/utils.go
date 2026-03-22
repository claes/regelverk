package regelverk

import (
	"fmt"
	"time"

	"github.com/sj14/astral/pkg/astral"
)

type DayPhase struct {
	Phase    PhaseOfDay `json:"phase"`
	Meridiem Meridiem   `json:"meridiem"`
}

type PhaseOfDay int

type Meridiem int

const (
	Nighttime PhaseOfDay = iota
	MorningAstronomicalTwilight
	MorningNauticalTwilight
	MorningCivilTwilight
	Daytime
	EveningCivilTwilight
	EveningNauticalTwilight
	EveningAstronomcialTwilight
)

const (
	AnteMeridiem Meridiem = iota
	PostMeridiem
)

func (t PhaseOfDay) String() string {
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

func (t Meridiem) String() string {
	switch t {
	case AnteMeridiem:
		return "AM"
	case PostMeridiem:
		return "PM"
	default:
		return "Unknown Meridieu"
	}
}

func ComputePhaseOfDay(currentTime time.Time, lat, long float64) DayPhase {

	observer := astral.Observer{
		Latitude:  lat,
		Longitude: long,
		Elevation: 0.0,
	}

	var meridiem Meridiem
	if currentTime.Before(astral.Noon(observer, currentTime)) {
		meridiem = AnteMeridiem
	} else {
		meridiem = PostMeridiem
	}

	// TODO should we instead use Astral function here
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

	var phase PhaseOfDay

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
	return DayPhase{Phase: phase, Meridiem: meridiem}
}

func foo() {
	today := nowFunc()
	location, _ := time.LoadLocation("CET")
	midnight := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, location)

	for i := 0; i < 24*4; i++ {
		currentHour := midnight.Add(time.Duration(i*15) * time.Minute)
		timeOfDay := ComputePhaseOfDay(currentHour, 59, 18)
		fmt.Printf("Phase of the day for %v is %s\n", currentHour.In(location).Format("2006-01-02 15:04:05 MST"), timeOfDay.Phase)
	}
}
