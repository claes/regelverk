package main

var loops = []controlLoop{
	//&tvLoop{},
	&mpdLoop{},
	&tvAudioLoop{},
	&cecLoop{},
	&rotelHttpLoop{},
}

func initLoops(msgHandler *mqttMessageHandler) {
	for _, l := range loops {
		l.Init(msgHandler)
	}
}
