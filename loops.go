package main

var loops = []controlLoop{
	//&tvLoop{},
	&mpdLoop{},
	&presenceLoop{},
	&kitchenLoop{},
	&cecLoop{},
	&webLoop{},
}

func initLoops(msgHandler *mqttMessageHandler) {
	for _, l := range loops {
		l.Init(msgHandler)
	}
}
