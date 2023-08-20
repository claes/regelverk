package main

import (
	"encoding/json"
	"fmt"
)

func parseJSONPayload(ev MQTTEvent) map[string]interface{} {
	var payload interface{}
	payloadJson := string(ev.Payload.([]byte))
	err := json.Unmarshal([]byte(payloadJson), &payload)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	m := payload.(map[string]interface{})
	return m
}

// func createMQTTPayload(data map[string]interface{}) []byte {
// 	payloadBytes, err := json.Marshal(data)
// 	if err != nil {
// 		fmt.Println(err)
// 		return nil
// 	}
// 	return payloadBytes
// }
