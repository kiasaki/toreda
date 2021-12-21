package main

import (
	"fmt"
	"time"
)

type GapNGoStrategy struct {
	symbolIds  []int
	cash       float64
	daysTraded map[string]float64
}

func NewGapNGoStrategy() *GapNGoStrategy {
	var tqqq = 32959
	//var sqqq = 16271758

	return &GapNGoStrategy{
		symbolIds:  []int{tqqq},
		cash:       10000,
		daysTraded: map[string]float64{},
	}
}

func (s *GapNGoStrategy) Run(now time.Time, ds Datasource, b Broker, om *OrderManager) error {
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
		// Check if we already traded today
		if _, ok := s.daysTraded[now.Format(dateFormat)]; ok {
			continue
		}
		// Check if we are already in a position
		position := om.CurrentPositionForIncludingPending(symbolId)
		if position.OpenQuantity > 0 {
			continue
		}

		startOfDay, err := time.ParseInLocation(dateTimeFormat, now.Format(dateFormat)+" 09:30:00", timeLocation)
		if err != nil {
			return err
		}
		tenAM := startOfDay.Add(30 * time.Minute)
		candles, err := ds.Candles(symbolId, startOfDay, now, CandleIntervalOneMinute)
		if err != nil {
			return err
		}
		if len(candles) <= 0 {
			continue
		}

		positionSize := s.cash / float64(len(s.symbolIds))
		lastPrice := candles[len(candles)-1].Close
		targetQty := int64(positionSize / lastPrice)

		openCandles := candlesRange(candles, startOfDay, tenAM)
		if len(openCandles) <= 0 {
			continue
		}
		openLow, openHigh := closeLowHigh(openCandles)

		if lastPrice > openHigh {
			stopPrice := openLow + ((openHigh - openLow) * 0.75)
			fmt.Println(startOfDay.Format(dateTimeFormat), lastPrice, openLow, openHigh, stopPrice)
			if err := om.Buy(symbolId, targetQty, stopPrice); err != nil {
				return err
			}
			s.daysTraded[now.Format(dateFormat)] = lastPrice
		}
	}

	return nil
}
