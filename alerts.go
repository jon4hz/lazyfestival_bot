package main

import (
	"encoding/json"
	"os"
	"time"
)

type Alert struct {
	Band  string
	Time  time.Time
	Min5  bool
	Min15 bool
	Min30 bool
	Hour1 bool
	Hour2 bool
}

func loadAlerts() (map[int64][]*Alert, error) {
	data, err := os.ReadFile("alerts.json")
	if err != nil {
		return nil, err
	}
	alerts := make(map[int64][]*Alert)
	if err := json.Unmarshal(data, &alerts); err != nil {
		return nil, err
	}
	return alerts, nil
}

func storeAlerts(alerts map[int64][]*Alert) error {
	data, err := json.Marshal(alerts)
	if err != nil {
		return err
	}
	if err := os.WriteFile("alerts.json", data, 0644); err != nil {
		return err
	}
	return nil
}
