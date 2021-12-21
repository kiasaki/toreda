package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

var (
	allSymbols                  = []*SymbolDetails{}
	timeLocation *time.Location = nil
)

func main() {
	var err error
	if timeLocation, err = time.LoadLocation("America/New_York"); err != nil {
		log.Fatalln(err)
	}

	if len(os.Args) == 2 && os.Args[1] == "server" {
		loadAllSymbols()
		startServer()
	}

	if len(os.Args) == 2 && os.Args[1] == "fetch-daily" {
		allSymbols := loadAllSymbolNames()
		fetchDailyDetails(allSymbols)
		return
	}

	fmt.Println(`Usage: toreda [command]

  server       Starts trading web server
  fetch-daily  Downloads daily details for all known symbols
`)
}
