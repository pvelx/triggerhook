package util

import "github.com/satori/go.uuid"

func NewId() string {
	return uuid.NewV4().String()
}

func IsIdValid(id string) bool {
	if _, err := uuid.FromString(id); err != nil {
		return false
	}

	return true
}
