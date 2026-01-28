package models

import "encoding/json"

// ToJSON converts an object to a JSON string, returns empty string on error
func ToJSON(obj interface{}) string {
	if obj == nil {
		return ""
	}
	b, err := json.Marshal(obj)
	if err != nil {
		return ""
	}
	return string(b)
}// FromPtr returns the value of a string pointer, or empty string if nil
func FromPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
