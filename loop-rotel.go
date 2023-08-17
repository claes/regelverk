package main

import (
	"fmt"
)

type rotelLoop struct {
	statusLoop
	hasMuted bool
}

func (l *rotelLoop) ProcessEvent(ev MQTTEvent) []MQTTPublish {
	fmt.Printf("Rotelloop topic %v, payload %v \n", ev.Topic, ev.Payload)

	var doMute bool

	switch ev.Topic {
	case "rotel/volume":
		if string(ev.Payload.([]byte)) == "10" {
			fmt.Printf("volume = 10, should mute")
			doMute = true
		} else {
			fmt.Printf("volume != 10, should not mute")
			doMute = false
			l.hasMuted = false
		}
	default:
		return nil // did not influence state
	}

	if doMute && !l.hasMuted {
		l.hasMuted = true
		return []MQTTPublish{
			{
				Topic:    "rotel/mute/set",
				Payload:  "on",
				Retained: false,
			},
		}
	} else {
		return nil
	}
}
