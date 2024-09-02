package main

var loops = []controlLoop{
	//&tvLoop{},
	&mpdLoop{},
	&tvAudioLoop{},
	&cecLoop{},
	&webLoop{},
}

func initLoops(msgHandler *mqttMessageHandler) {
	for _, l := range loops {
		l.Init(msgHandler)
	}
}
