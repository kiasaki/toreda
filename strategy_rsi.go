package main

import "time"

type RsiStrategy struct {
	lastStopOrder *BrokerOrder
}

func NewRsiStrategy() *RsiStrategy {
	return &RsiStrategy{}
}

func (s *RsiStrategy) Run(now time.Time, ds Datasource, b Broker, om *OrderManager) error {
	var longSymId = 32959
	var shortSymId = 16271758
	var longTargetQty int64 = 100  // TQQQ runs at around 100$ a share so ~ 10k
	var shortTargetQty int64 = 300 // SQQQ runs at around 33$ a share so ~ 10k
	var rsiSize = 15

	if canTrade, err := dayTradePrecheck(om, now, []int{longSymId, shortSymId}); err != nil {
		return err
	} else if !canTrade {
		// Make sure to cancel our stop if outside trading hours
		if err := om.CancelAllStops(longSymId); err != nil {
			return err
		}
		if err := om.CancelAllLimits(longSymId); err != nil {
			return err
		}
		if err := om.CancelAllStops(shortSymId); err != nil {
			return err
		}
		if err := om.CancelAllLimits(shortSymId); err != nil {
			return err
		}
		return nil
	}

	positions := b.LastPositions()
	for _, p := range positions {
		if p.OpenQuantity == 0 {
			// Ensure all limits are cancelled if OpenQuantity drops to 0
			if err := om.CancelAllLimits(p.SymbolId); err != nil {
				return err
			}
		} else if s.lastStopOrder != nil && p.OpenQuantity < s.lastStopOrder.TotalQuantity {
			// Ensure all stops are adjusted when limits are reached
			if err := om.CancelAllStops(p.SymbolId); err != nil {
				return err
			}
			if o, err := b.CreateOrder(
				s.lastStopOrder.SymbolId, OrderActionSell, OrderTypeStop,
				s.lastStopOrder.StopPrice, p.OpenQuantity,
			); err != nil {
				return err
			} else {
				s.lastStopOrder = o
			}
		}
	}

	startOfDay := now.Add(-12 * time.Hour).Round(24 * time.Hour).Add((9.5 * 60) * time.Minute)
	candles, err := ds.Candles(longSymId, startOfDay, now, CandleIntervalOneMinute)
	if err != nil {
		return err
	}
	shortCandles, err := ds.Candles(shortSymId, startOfDay, now, CandleIntervalOneMinute)
	if err != nil {
		return err
	}

	lastIndex := len(candles) - 1

	// Missing bars for RSI
	if lastIndex <= rsiSize {
		return nil
	}

	currentCandleRSI := rsi(candles[lastIndex-rsiSize : lastIndex])

	// Go long when oversold
	if currentCandleRSI > 60 {
		stopPrice := candles[len(candles)-1].Close - 1
		limitPrice := candles[len(candles)-1].Close + 1
		if err := s.buy(b, om, longSymId, longTargetQty, stopPrice, limitPrice); err != nil {
			return err
		}

		if err := s.sell(om, shortSymId); err != nil {
			return err
		}
	}
	// Go short when overbought
	if currentCandleRSI < 40 {
		if err := s.sell(om, longSymId); err != nil {
			return err
		}

		stopPrice := shortCandles[len(shortCandles)-1].Close - 1
		limitPrice := shortCandles[len(shortCandles)-1].Close + 1
		if err := s.buy(b, om, shortSymId, shortTargetQty, stopPrice, limitPrice); err != nil {
			return err
		}
	}

	return nil
}

func (s *RsiStrategy) buy(
	b Broker, om *OrderManager, symId int, qty int64,
	stopPrice float64, limitPrice float64,
) error {
	if err := om.Ensure(symId, qty); err != nil {
		return err
	}
	if err := om.CancelAllStops(symId); err != nil {
		return err
	}
	if err := om.CancelAllLimits(symId); err != nil {
		return err
	}
	if o, err := b.CreateOrder(
		symId, OrderActionSell, OrderTypeStop, stopPrice, qty,
	); err != nil {
		return err
	} else {
		s.lastStopOrder = o
	}
	if _, err := b.CreateOrder(
		symId, OrderActionSell, OrderTypeLimit, limitPrice, qty/2,
	); err != nil {
		return err
	}
	return nil
}

func (s *RsiStrategy) sell(om *OrderManager, symId int) error {
	if err := om.Ensure(symId, 0); err != nil {
		return err
	}
	if err := om.CancelAllStops(symId); err != nil {
		return err
	}
	if err := om.CancelAllLimits(symId); err != nil {
		return err
	}
	return nil
}
