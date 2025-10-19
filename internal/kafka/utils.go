package kafka

import "strings"

func isKafkaManagedTopic(topic string) bool {
	for _, prefix := range []string{"temperatures_", "humidities_", "ph_levels_"} {
		if strings.HasPrefix(topic, prefix) {
			return true
		}
	}
	return false
}
