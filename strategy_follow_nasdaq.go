package main

import "time"

type FollowNasdaqStrategy struct {
	dayEntry          map[string]bool
	maxDayLossPercent float64
}

func NewFollowNasdaqStrategy() *FollowNasdaqStrategy {
	return &FollowNasdaqStrategy{
		dayEntry:          map[string]bool{},
		maxDayLossPercent: 0.0015,
	}
}

func (s *FollowNasdaqStrategy) Run(now time.Time, ds Datasource, b Broker, om *OrderManager) error {
	var symId int = 32959
	var qty int64 = 100

	if now.Hour() < 10 {
		return nil
	}

	if now.Hour() >= 15 && now.Minute() >= 50 {
		if err := om.CancelAllStops(symId); err != nil {
			return err
		}
		if err := om.CancelAllLimits(symId); err != nil {
			return err
		}
		if err := om.Ensure(symId, 0); err != nil {
			return err
		}
		return nil
	}

	day := now.Format("2006-01-02")
	if _, ok := s.dayEntry[day]; !ok {
		startOfDay := now.Add(-12 * time.Hour).Round(24 * time.Hour).Add((9.5 * 60) * time.Minute)
		candles, err := ds.Candles(symId, startOfDay, now, CandleIntervalOneMinute)
		if err != nil {
			return err
		}
		if len(candles) < 1 {
			return nil
		}

		if err := om.Ensure(symId, qty); err != nil {
			return err
		}

		stopPrice := candles[len(candles)-1].Close * (1.0 - s.maxDayLossPercent)
		if _, err := b.CreateOrder(
			symId, OrderActionSell, OrderTypeStop, stopPrice, qty,
		); err != nil {
			return err
		}

		s.dayEntry[day] = true
	}

	return nil
}
