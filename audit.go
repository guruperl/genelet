package genelet

import "log"

func LogInfo(message string, fields map[string]interface{}) {
	log.Printf("%s: %#v", message, fields)
}
