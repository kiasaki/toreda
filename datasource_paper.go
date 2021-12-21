package main

import (
	"errors"
	"fmt"
	"os"
	"time"
)

type PaperDatasource struct {
	logger     Logger
	tm         TradingManager
	candleData map[string][]*SymbolCandle
}

func NewPaperDatasource(tm TradingManager, logger Logger) *PaperDatasource {
	return &PaperDatasource{
		logger:     logger,
		tm:         tm,
		candleData: map[string][]*SymbolCandle{},
	}
}

func (ds *PaperDatasource) Details(sym string) (*SymbolDetails, error) {
	panic("not implemented")
}
func (ds *PaperDatasource) Quote(id int) (*SymbolQuote, error) {
	price, err := ds.currentPrice(id)
	if err != nil {
		return nil, err
	}
	symbolDetails := findSymbolDetailsById(id)
	if symbolDetails == nil {
		return nil, errors.New("broker_paper: quote: can't find symbol")
	}
	return &SymbolQuote{
		Symbol:         symbolDetails.Symbol,
		SymbolId:       id,
		BidPrice:       price * 0.999,
		AskPrice:       price * 1.001,
		LastTradePrice: price,
	}, nil
}

func (ds *PaperDatasource) currentPrice(symId int) (float64, error) {
	var err error
	var candles []*SymbolCandle
	var daysToGoBack = 3
	var day = 0
	for len(candles) == 0 && day <= daysToGoBack {
		// Maybe try yesterday, sometimes there's no candle at 9:30 sharp
		startDiff := time.Duration(-1*(7+(day*24))) * time.Hour
		candles, err = ds.Candles(symId, ds.tm.Now().Add(startDiff), ds.tm.Now(), CandleIntervalOneMinute)
		if err != nil {
			return 0, err
		}
		day++
	}

	if len(candles) == 0 {
		return 0, errors.New(fmt.Sprintf("currentPrice: 0 candles found for %d at %s", symId, ds.tm.Now().Format("2006-01-02 15:04")))
	}
	return candles[len(candles)-1].Close, nil
}

func (ds *PaperDatasource) Candles(
	symId int, start, end time.Time, interval CandleInterval,
) ([]*SymbolCandle, error) {
	if interval != CandleIntervalOneMinute && interval != CandleIntervalFiveMinute {
		return nil, errors.New("Candle interval requested not implemented")
	}
	symbolDetails := findSymbolDetailsById(symId)
	if symbolDetails == nil {
		return nil, errors.New(fmt.Sprintf("Can't find symbol details for id: %d", symId))
	}

	ds.logger.LogDebug(
		"datasource", "fetching_candles,paper,%s,%s,%s,%s",
		symbolDetails.Symbol, start.Format(dateTimeFormat), end.Format(dateTimeFormat), interval,
	)

	intervalFolder := "1m-1d"
	if interval == CandleIntervalFiveMinute {
		intervalFolder = "5m-1d"
	}

	dataSliceName := intervalFolder + "/" + symbolDetails.Symbol + "/" + start.Format("2006-01-02")
	if candles, ok := ds.candleData[dataSliceName]; ok {
		return ds.filterCandles(candles, start, end), nil
	}
	var candles []*SymbolCandle
	err := readJsonFile("data/"+dataSliceName+".json", &candles)
	if err != nil {
		// Try fetching from kibot
		candles, err = callKibotHistory(symbolDetails.Symbol, interval, start, end)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll("data/"+intervalFolder+"/"+symbolDetails.Symbol, 0755); err != nil {
			return nil, err
		}
		err = writeJsonFile("data/"+dataSliceName+".json", candles)
		if err != nil {
			return nil, err
		}
	}
	ds.candleData[dataSliceName] = candles
	return ds.filterCandles(candles, start, end), nil
}

func (ds *PaperDatasource) filterCandles(candles []*SymbolCandle, start, end time.Time) []*SymbolCandle {
	filteredCandles := []*SymbolCandle{}
	for _, c := range candles {
		if c.Start.After(start) && c.Start.Before(end.Add(time.Second)) {
			filteredCandles = append(filteredCandles, c)
		}
	}
	return filteredCandles
}
