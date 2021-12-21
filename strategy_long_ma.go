package main

import (
	"time"
)

// var aapl = 8049
// var amd = 6770
// var amzn = 7410
// var bgu = 0         // big cap
// var cure = 605427   // medical
// var czm = 0         // china
// var erx = 16130     // energy
// var ery = 3575753   // energy bear
// var fas = 1261174   // finance
// var gasx = 14888422 // natural gaz bear
// var intc = 23205
// var jnug = 17478823 // junior gold miners
// var jpnl = 4064867  // japan
// var ko = 14126
// var ms = 28020
// var msft = 27426
// var mu = 27454
// var nke = 29251
// var nvda = 29814
// var pfe = 31867
// var rusl = 13285014 // russia
// var sqqq = 16271758 // nasdaq 100 bear
// var udow = 32958    // dow 30
// var umdd = 32957    // mid-cap 400
// var wfc = 41279
// var xiv = 15121     // volatility
// var yinn = 16126    // china
// var edc = 13285015  // emerging markets

type LongMAStrategy struct {
	symbolIds    []int
	fastMASize   int
	slowMASize   int
	startingCash float64
	closes       map[string][]float64
	lastMinute   int
}

func NewLongMAStrategy() *LongMAStrategy {
	// var drn = 16124     // real estate
	// var soxl = 16114    // semiconductor
	// var spy = 34987     // S&P 1x
	// var tqqq = 32959    // nasdaq 100
	// var ubio = 11831068 // biotech
	var drip = 14888420 // oil & gas services s&p bear
	var dwt = 15968521  // crude oil bear
	var labu = 13285018 // biotech

	return &LongMAStrategy{
		symbolIds:    []int{labu, drip, dwt},
		fastMASize:   8,
		slowMASize:   21,
		startingCash: 1000,
		closes:       map[string][]float64{},
		lastMinute:   -1,
	}
}

func (s *LongMAStrategy) Run(now time.Time, ds Datasource, b Broker, om *OrderManager) error {
	if canTrade, err := dayTradePrecheck(om, now, s.symbolIds); err != nil {
		return err
	} else if !canTrade {
		// Just make sure we don't have any pending orders
		if err := om.CancelPendingOrders(); err != nil {
			return err
		}
		for _, symbolId := range s.symbolIds {
			if err := om.SellAll(symbolId); err != nil {
				return err
			}
		}
		return nil
	}

	for _, symbolId := range s.symbolIds {
		position := om.CurrentPositionForIncludingPending(symbolId)
		if position.OpenQuantity > 0 {
			quantity := position.OpenQuantity
			entryPrice := position.CurrentPrice
			currentPrice := position.AverageEntryPrice
			if currentPrice/entryPrice > 1.0035 {
				if err := om.CancelAllStops(symbolId); err != nil {
					return err
				}
				if _, err := om.broker.CreateOrder(
					symbolId, OrderActionSell, OrderTypeStop,
					position.AverageEntryPrice+0.03, quantity,
				); err != nil {
					return err
				}
			}
		}

		candles, err := ds.Candles(symbolId, now.Add(-180*time.Minute), now, CandleIntervalFiveMinute)
		if err != nil {
			return err
		}
		// Check that we have enough bars
		if len(candles) < s.slowMASize {
			continue
		}

		positionSize := s.startingCash / float64(len(s.symbolIds))
		lastPrice := candles[len(candles)-1].Close

		currentCandleFastMA := wmaValue(candles[s.slowMASize-s.fastMASize:])
		currentCandleSlowMA := wmaValue(candles)
		crossoverSize := (currentCandleFastMA - currentCandleSlowMA) / lastPrice

		if crossoverSize > 0.005 {
			position := om.CurrentPositionForIncludingPending(symbolId)
			if position.OpenQuantity == 0 {
				quote, err := ds.Quote(symbolId)
				if err != nil {
					return err
				}
				lastPrice = quote.AskPrice
				targetQty := int64(positionSize / lastPrice)
				stopSpread := fmax(lastPrice*0.0025, 0.02) // minimum 0.02$ stop
				stopPrice := lastPrice - stopSpread
				if err := om.Buy(symbolId, targetQty, stopPrice); err != nil {
					return err
				}
			}
		}
		if crossoverSize < 0.001 {
			if err := om.SellAll(symbolId); err != nil {
				return err
			}
		}
	}

	return nil
}
