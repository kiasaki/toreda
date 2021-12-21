package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

func findSymbolDetailsByName(name string) *SymbolDetails {
	for _, s := range allSymbols {
		if s.Symbol == name {
			return s
		}
	}
	return nil
}

func findSymbolDetailsById(id int) *SymbolDetails {
	for _, s := range allSymbols {
		if s.SymbolId == id {
			return s
		}
	}
	return nil
}

func loadAllSymbols() {
	log.Println("Loading symbol details")
	days := 0
	yesterday := time.Now().In(timeLocation).Add(-16 * time.Hour)
	allSymbols = []*SymbolDetails{}
load:
	err := readJsonFile(
		fmt.Sprintf("data/details/%s.json", yesterday.Format("2006-01-02")),
		&allSymbols,
	)
	if err != nil && days < 30 {
		// Try 1 day before
		days += 1
		yesterday = yesterday.Add(-24 * time.Hour)
		goto load
	} else if err != nil {
		// Ok nevermind, we can't find 1 details file in the past month
		log.Fatalln(err)
	}
}

func loadAllSymbolNames() []string {
	f, err := os.Open("data/all_symbols.csv")
	if err != nil {
		log.Fatalln(err)
	}

	r := csv.NewReader(f)
	lines, err := r.ReadAll()
	if err != nil {
		log.Fatalln(err)
	}

	allSymbols := []string{}
	for _, line := range lines {
		allSymbols = append(allSymbols, strings.Trim(line[0], " "))
	}
	return allSymbols
}

func fetchDailyDetails(symbolNames []string) {
	ds := NewQTDatasource(NewConsoleLogger())

	symbolDetails := []*SymbolDetails{}

	for i := 0; i < len(symbolNames); i += 50 {
		log.Printf("Fetching daily details %d to %d\n", i, i+50)
		symbols, err := ds.BatchDetails(symbolNames[i:min(i+50, len(symbolNames))])
		if err != nil {
			log.Fatalln(err)
		}
		symbolDetails = append(symbolDetails, symbols...)
	}

	fileName := fmt.Sprintf("data/details/%s.json", time.Now().Add((-9.5*60)*time.Minute).Format("2006-01-02"))
	if err := writeJsonFile(fileName, symbolDetails); err != nil {
		log.Fatalln(err)
	}

	log.Printf("Wrote: %s\n", fileName)
	log.Println("Fetched daily details for all symbols")
}
