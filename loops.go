package main

var loops = []controlLoop{
	&tvLoop{},
	&mpdLoop{},
	&tvAudioLoop{},
	&rotelHttpLoop{},
}

func initLoops(msgHandler *mqttMessageHandler) {
	for _, l := range loops {
		l.Init(msgHandler)
	}
}
