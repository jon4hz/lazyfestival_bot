package main

import (
	"encoding/json"
	"os"
	"sort"
	"time"
)

type BandInternal struct {
	Date  string `json:"date"`
	Time  string `json:"time"`
	Band  string `json:"band"`
	Stage string `json:"stage"`
}

type Band struct {
	Date  time.Time
	Name  string
	Stage string
}

func load() ([]Band, error) {
	data, err := os.ReadFile("data.json")
	if err != nil {
		return nil, err
	}

	bandsInternal := make([]BandInternal, 0)
	if err := json.Unmarshal(data, &bandsInternal); err != nil {
		return nil, err
	}

	bands := make([]Band, len(bandsInternal))
	for i, bandInternal := range bandsInternal {
		date, err := time.Parse("2006 January 02 15:04 MST", "2024 "+bandInternal.Date+" "+bandInternal.Time+" CEST")
		if err != nil {
			return nil, err
		}
		bands[i] = Band{
			Date:  date,
			Name:  bandInternal.Band,
			Stage: bandInternal.Stage,
		}
	}
	sort.Slice(bands, func(i, j int) bool {
		return bands[i].Date.Before(bands[j].Date)
	})
	return bands, nil
}
