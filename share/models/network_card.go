package models

import (
	"fmt"
	"strconv"
)

const errorPrefix = "network card"

type NetworkCard struct {
	Name     string `json:"name"`
	MaxSpeed int    `json:"max_speed"`
}

func DecodeCard(sCard []string) (*NetworkCard, error) {
	if len(sCard) != 2 {
		return nil, fmt.Errorf("%s description illegal", errorPrefix)
	}
	if sCard[0] == "" {
		return nil, fmt.Errorf("%s device name missing", errorPrefix)
	}
	if sCard[1] == "" {
		return nil, fmt.Errorf("%s max speed missing", errorPrefix)
	}
	maxSpeed, err := strconv.Atoi(sCard[1])
	if err != nil {
		return nil, fmt.Errorf("%s max speed illegal: %s", errorPrefix, sCard[1])
	}
	if maxSpeed < 1 {
		return nil, fmt.Errorf("%s max speed low: %d", errorPrefix, maxSpeed)
	}

	card := &NetworkCard{
		Name:     sCard[0],
		MaxSpeed: maxSpeed,
	}

	return card, nil
}
