package main

import (
	"log"
	"os"
)

func main() {
	bands, err := load()
	if err != nil {
		log.Fatal(err)
	}

	var bandsByDay = make([][]Band, 1)
	currDay := bands[0].Date.Day()
	i := 0
	for _, band := range bands {
		if band.Date.Day() != currDay {
			i++
			currDay = band.Date.Day()
			bandsByDay = append(bandsByDay, make([]Band, 0))
		}
		bandsByDay[i] = append(bandsByDay[i], band)
	}

	b, err := NewClient(os.Getenv("BOTTOKEN"), bandsByDay)
	if err != nil {
		log.Fatal(err)
	}
	if err := b.Run(); err != nil {
		log.Fatal(err)
	}
}
