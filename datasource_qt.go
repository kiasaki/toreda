package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const qtDateTimeFormat = "2006-01-02T15:04:05-07:00"

type QTDatasource struct {
	logger Logger
	api    *QTApi
}

func NewQTDatasource(logger Logger) *QTDatasource {
	return &QTDatasource{
		logger: logger,
		api:    NewQTApi(),
	}
}

type QTApiDetailsResponse struct {
	Symbols []*SymbolDetails `json:"symbols"`
}

type QTApiQuoteResponse struct {
	Quotes []*SymbolQuote `json:"quotes"`
}

type QTApiCandlesResponse struct {
	Candles []*SymbolCandle `json:"candles"`
}

func (ds *QTDatasource) Details(symbolName string) (*SymbolDetails, error) {
	response := QTApiDetailsResponse{}
	err := ds.api.Request(APIRequest{
		Path: "v1/symbols?names=" + symbolName,
	}, &response)
	if err != nil {
		return nil, err
	}
	if len(response.Symbols) == 0 {
		return nil, errors.New(fmt.Sprintf("No symbol named: %s", symbolName))
	}
	return response.Symbols[0], nil
}

func (ds *QTDatasource) BatchDetails(symbols []string) ([]*SymbolDetails, error) {
	response := QTApiDetailsResponse{}
	err := ds.api.Request(APIRequest{
		Path: "v1/symbols?names=" + strings.Join(symbols, ","),
	}, &response)
	return response.Symbols, err
}

func (ds *QTDatasource) Quote(id int) (*SymbolQuote, error) {
	response := QTApiQuoteResponse{}
	err := ds.api.Request(APIRequest{
		Path: "v1/markets/quotes?ids=" + strconv.Itoa(id),
	}, &response)
	// TODO check length
	return response.Quotes[0], err
}

func (ds *QTDatasource) BatchQuote(ids []int) ([]*SymbolQuote, error) {
	response := QTApiQuoteResponse{}
	err := ds.api.Request(APIRequest{
		Path: "v1/markets/quotes?ids=" + intArrayToString(ids),
	}, &response)
	return response.Quotes, err
}

func (ds *QTDatasource) Candles(id int, start, end time.Time, interval CandleInterval) ([]*SymbolCandle, error) {
	end = end.Truncate(time.Minute)
	response := QTApiCandlesResponse{}
	path := fmt.Sprintf(
		"v1/markets/candles/%d?startTime=%s&endTime=%s&interval=%s",
		id, start.Format(qtDateTimeFormat), end.Format(qtDateTimeFormat), interval,
	)
	symbolDetails := findSymbolDetailsById(id)
	if symbolDetails == nil {
		return nil, errors.New("datasource: can't find details for given symbol id")
	}
	ds.logger.LogDebug(
		"datasource", "fetching_candles,qt,%s,%s,%s,%s",
		symbolDetails.Symbol, start.Format(dateTimeFormat), end.Format(dateTimeFormat), interval,
	)
	err := ds.api.Request(APIRequest{Path: path}, &response)
	return response.Candles, err
}
