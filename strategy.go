package main

import (
	"time"
)

type Strategy interface {
	Run(time.Time, Datasource, Broker, *OrderManager) error
}

func candleForRange(candles []*SymbolCandle, start, end time.Time) *SymbolCandle {
	candle := &SymbolCandle{}
	var lastCandle *SymbolCandle
	for _, c := range candles {
		if candle.Open == 0 {
			candle.Open = c.Open
		}
		if c.Start.After(start) && c.End.Before(end) {
			candle.High = fmax(candle.High, c.High)
			candle.Low = fmin(candle.Low, c.Low)
		}
		lastCandle = c
	}
	candle.Close = lastCandle.Close
	return candle
}

func closeLowHigh(candles []*SymbolCandle) (float64, float64) {
	var low float64 = 1000000
	var high float64 = -1000000
	for _, c := range candles {
		low = fmin(fmin(low, c.Open), c.Open)
		high = fmax(fmax(high, c.Open), c.Close)
	}
	return low, high
}

func candlesRange(candles []*SymbolCandle, start, end time.Time) []*SymbolCandle {
	filteredCandles := []*SymbolCandle{}
	for _, c := range candles {
		if c.Start.After(start) && c.Start.Before(end) {
			filteredCandles = append(filteredCandles, c)
		}
	}
	return filteredCandles
}

func avgClose(candles []*SymbolCandle) float64 {
	var sum float64
	for _, c := range candles {
		sum += c.Close
	}
	return sum / float64(len(candles))
}

func wmaValue(candles []*SymbolCandle) float64 {
	var denominator = float64((len(candles) * (len(candles) + 1)) / 2)
	var total float64
	for i, c := range candles {
		total += c.Close * (float64(i+1) / denominator)
	}
	return total
}

func fwmaValue(values []float64) float64 {
	var denominator = float64((len(values) * (len(values) + 1)) / 2)
	var total float64
	for i, v := range values {
		total += v * (float64(i+1) / denominator)
	}
	return total
}

func avg(prices []float64) float64 {
	var sum float64
	for _, p := range prices {
		sum += p
	}
	return sum / float64(len(prices))
}

func rsi(candles []*SymbolCandle) float64 {
	gains := []float64{}
	losses := []float64{}
	for i := 1; i < len(candles); i++ {
		curr := candles[i].Close
		last := candles[i-1].Close
		if curr > last {
			gains = append(gains, curr-last)
		} else if last > curr {
			losses = append(losses, last-curr)
		}
	}
	return 100 - (100 / (1 + (avg(gains) / avg(losses))))
}

// Returns true (& ensure all traded symbols are sold) when before 10am or
// close to market close
func dayTradePrecheck(om *OrderManager, now time.Time, symbolsTraded []int) (bool, error) {
	// Wait till 10 am to start
	if now.Hour() < 10 {
		for _, s := range symbolsTraded {
			if err := om.Ensure(s, 0); err != nil {
				return false, err
			}
		}
		return false, nil
	}

	// End of Day, exit all positions
	if (now.Hour() >= 15 && now.Minute() >= 50) || now.Hour() >= 16 {
		for _, s := range symbolsTraded {
			if err := om.Ensure(s, 0); err != nil {
				return false, err
			}
		}
		return false, nil
	}

	return true, nil
}
