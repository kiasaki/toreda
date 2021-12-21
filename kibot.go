package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const KIBOT_BASE_URL = "http://api.kibot.com/"

func callKibotHistory(
	symbol string, interval CandleInterval, start time.Time, end time.Time,
) ([]*SymbolCandle, error) {
	var retries int = 0
	var kibotInterval string
	if interval == CandleIntervalOneMinute {
		kibotInterval = "1"
	} else if interval == CandleIntervalFiveMinute {
		kibotInterval = "5"
	} else {
		return nil, errors.New(
			"callKibot: unsupported interval: " + string(interval),
		)
	}

	url := fmt.Sprintf(
		"%s?action=history&splitadjusted=1&symbol=%s&interval=%s&startdate=%s&enddate=%s",
		KIBOT_BASE_URL, symbol, kibotInterval, start.Format("2006-01-02"), end.Format("2006-01-02"),
	)
	log.Println("GET " + url)
request:
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	body := string(bytes)

	// Login if needed
	if body == "401 Not Logged In" && retries < 3 {
		// http://api.kibot.com/?action=login&user=guest&password=guest
		resp, err := http.Get(fmt.Sprintf(
			"%s?action=login&user=%s&password=%s",
			KIBOT_BASE_URL,
			os.Getenv("KIBOT_USERNAME"),
			os.Getenv("KIBOT_PASSWORD"),
		))
		if err != nil {
			return nil, err
		}
		resp.Body.Close()
		retries++
		goto request
	}

	if body[:3] == "405" {
		// No data
		return []*SymbolCandle{}, nil
	}

	// Parse
	records, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	if err != nil {
		return nil, errors.New("callKibot: error reading response: " + strings.Split(body, "\n")[0])
	}

	candles := []*SymbolCandle{}
	for _, record := range records {
		date, err := time.ParseInLocation("01/02/2006 15:04", record[0]+" "+record[1], timeLocation)
		if err != nil {
			return nil, err
		}
		candles = append(candles, &SymbolCandle{
			Start:  date,
			End:    date.Add(time.Minute),
			Open:   mustParseFloat(record[2]),
			High:   mustParseFloat(record[3]),
			Low:    mustParseFloat(record[4]),
			Close:  mustParseFloat(record[5]),
			Volume: int64(mustParseFloat(record[6])),
		})
	}
	return candles, nil
}
