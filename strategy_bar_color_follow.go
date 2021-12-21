package main

import (
	"time"
)

type BarColorFollowStrategy struct{}

func NewBarColorFollowStrategy() *BarColorFollowStrategy {
	return &BarColorFollowStrategy{}
}

func (s *BarColorFollowStrategy) Run(now time.Time, ds Datasource, b Broker, om *OrderManager) error {
	var longSymId = 32959
	//var shortSymId = 16271758
	var longTargetQty int64 = 100
	//var shortTargetQty int64 = 300

	if canTrade, err := dayTradePrecheck(om, now, []int{longSymId}); err != nil {
		return err
	} else if !canTrade {
		// Make sure to cancel our stop if outside trading hours
		if err := om.CancelAllStops(longSymId); err != nil {
			return err
		}
		return nil
	}

	candles, err := ds.Candles(longSymId, now.Add(-5*time.Minute), now, CandleIntervalOneMinute)
	if err != nil {
		return err
	}

	if len(candles) < 3 {
		return nil
	}

	//prevprevCandle := candles[len(candles)-3]
	previousCandle := candles[len(candles)-2]
	currentCandle := candles[len(candles)-1]
	currentPosition := om.CurrentPositionFor(longSymId)

	if currentPosition.OpenQuantity > 0 {
		if currentCandle.Red() && previousCandle.Red() {
			if err := om.CancelAllStops(longSymId); err != nil {
				return err
			}
			if err := om.Ensure(longSymId, 0); err != nil {
				return err
			}
		}
	} else {
		if currentCandle.Green() && previousCandle.Green() &&
			currentCandle.Volume > previousCandle.Volume &&
			currentCandle.Close-previousCandle.Open > 0.25 &&
			previousCandle.Volume > 800 {
			if err := om.Ensure(longSymId, longTargetQty); err != nil {
				return err
			}
			if _, err := b.CreateOrder(
				longSymId, OrderActionSell, OrderTypeStop, currentCandle.Close-0.2, longTargetQty,
			); err != nil {
				return err
			}
		}
	}

	return nil
}
