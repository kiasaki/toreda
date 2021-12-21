package main

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

type PaperBroker struct {
	tm         TradingManager
	logger     Logger
	cash       float64
	positions  []*BrokerPosition
	executions []*BrokerExecution
	orders     []*BrokerOrder
}

func NewPaperBroker(tradingManager TradingManager, logger Logger) *PaperBroker {
	return &PaperBroker{
		tm:         tradingManager,
		logger:     logger,
		cash:       25000,
		positions:  []*BrokerPosition{},
		executions: []*BrokerExecution{},
		orders:     []*BrokerOrder{},
	}
}

func (b *PaperBroker) CreateOrder(
	symId int, action OrderAction, typ OrderType, limitOrStopPrice float64, quantity int64,
) (*BrokerOrder, error) {
	var err error
	var execution *BrokerExecution = &BrokerExecution{}
	symbolDetails := findSymbolDetailsById(symId)
	if symbolDetails == nil {
		return nil, errors.New(fmt.Sprintf("Can't find symbol details for id: %d", symId))
	}

	order := &BrokerOrder{
		Id:               rand.Int(),
		Symbol:           symbolDetails.Symbol,
		SymbolId:         symId,
		CreationTime:     b.tm.Now(),
		UpdateTime:       b.tm.Now(),
		TotalQuantity:    quantity,
		OpenQuantity:     0,
		FilledQuantity:   quantity,
		CanceledQuantity: 0,
		Side:             string(action),
		Type:             typ,
	}

	// If this is not a market order, delay execution
	if typ == OrderTypeLimit || typ == OrderTypeStop {
		order.State = "Accepted"
		order.OpenQuantity = order.TotalQuantity
		order.FilledQuantity = 0
		if typ == OrderTypeLimit {
			order.LimitPrice = limitOrStopPrice
		} else if typ == OrderTypeStop {
			order.StopPrice = limitOrStopPrice
		}
		b.orders = append(b.orders, order)
		b.logger.LogInfo(
			"broker", "create_order,%d,%s,%s,%s,%d,S%.2f,L%.2f",
			order.Id, order.Side, order.Type, order.Symbol,
			order.TotalQuantity, order.StopPrice, order.LimitPrice,
		)
		return order, nil
	}

	order.AvgExecPrice, err = b.currentPrice(symId)
	if err != nil {
		return nil, errors.New("CreateOrder: currentPrice returned an error: " + err.Error())
	}
	order.AvgExecPrice = order.AvgExecPrice * 1.00025 // add slippage

	execution, err = b.CreateExecution(symbolDetails, order, order.AvgExecPrice)
	if err != nil {
		return nil, err
	}

	b.orders = append(b.orders, order)

	b.logger.LogInfo(
		"broker", "create_order,%d,%s,%s,%s,%d,%.2f,%.2f",
		order.Id, order.Side, order.Type, order.Symbol,
		order.TotalQuantity, order.AvgExecPrice, execution.TotalCost,
	)
	return order, nil
}

func (b *PaperBroker) CreateExecution(symbolDetails *SymbolDetails, order *BrokerOrder, price float64) (*BrokerExecution, error) {
	if order.State == "Executed" {
		return nil, errors.New("CreateExecution: Trying execute and already executed order. Order " + strconv.Itoa(order.Id))
	}

	action := OrderAction(order.Side)
	order.State = "Executed"
	order.OpenQuantity = 0
	order.FilledQuantity = order.TotalQuantity
	order.AvgExecPrice = price

	execution := &BrokerExecution{
		Id:         rand.Int(),
		OrderId:    order.Id,
		Timestamp:  b.tm.Now(),
		Symbol:     symbolDetails.Symbol,
		SymbolId:   symbolDetails.SymbolId,
		Quantity:   order.TotalQuantity,
		Side:       order.Side,
		Price:      order.AvgExecPrice,
		TotalCost:  order.AvgExecPrice * float64(order.TotalQuantity),
		Commission: fmin(float64(order.TotalQuantity)*0.01, 6.95),
	}
	// TODO remove if not trading ETFs
	if action == OrderActionBuy {
		execution.Commission = 0
	}
	// Add SEC fees when creating market order
	if order.Type == OrderTypeMarket {
		execution.SecFee = execution.TotalCost * 0.0000231
	}
	// Adjust cash
	if action == OrderActionBuy {
		b.cash -= execution.TotalCost
	} else if action == OrderActionSell {
		b.cash += execution.TotalCost
	} else {
		return nil, errors.New(fmt.Sprintf("Can't process unknown order action: %s", action))
	}
	b.cash -= execution.Commission + execution.SecFee

	// Update position
	var position *BrokerPosition
	for _, p := range b.positions {
		if p.SymbolId == symbolDetails.SymbolId {
			position = p
		}
	}
	if position == nil {
		position = &BrokerPosition{
			Symbol:   symbolDetails.Symbol,
			SymbolId: symbolDetails.SymbolId,
		}
		b.positions = append(b.positions, position)
	}

	position.AverageEntryPrice = order.AvgExecPrice
	if action == OrderActionBuy {
		position.OpenQuantity += order.TotalQuantity
	} else if action == OrderActionSell {
		if position.OpenQuantity-order.TotalQuantity < 0 {
			return nil, errors.New("CreateExecution: Trying to sell more shares than owned. Order " + strconv.Itoa(order.Id))
		}
		position.OpenQuantity -= order.TotalQuantity
	}

	b.executions = append(b.executions, execution)

	return execution, nil
}

func (b *PaperBroker) currentPrice(symId int) (float64, error) {
	var err error
	var candles []*SymbolCandle
	var daysToGoBack = 3
	var day = 0
	for len(candles) == 0 && day <= daysToGoBack {
		// Maybe try yesterday, sometimes there's no candle at 9:30 sharp
		startDiff := time.Duration(-1*(7+(day*24))) * time.Hour
		candles, err = b.tm.Datasource().Candles(symId, b.tm.Now().Add(startDiff), b.tm.Now(), CandleIntervalOneMinute)
		if err != nil {
			return 0, err
		}
		day++
	}

	if len(candles) == 0 {
		return 0, errors.New(fmt.Sprintf("currentPrice: 0 candles found for %d at %s", symId, b.tm.Now().Format("2006-01-02 15:04")))
	}
	return candles[len(candles)-1].Close, nil
}

func (b *PaperBroker) CheckLimitAndStopOrders() error {
	var execution *BrokerExecution

	for _, o := range b.orders {
		if o.State == "Accepted" {
			symbolDetails := findSymbolDetailsById(o.SymbolId)
			if symbolDetails == nil {
				return errors.New(fmt.Sprintf("Can't find symbol details for id: %d", o.SymbolId))
			}
			currentPrice, err := b.currentPrice(o.SymbolId)
			if err != nil {
				return err
			}
			if o.Type == OrderTypeStop &&
				OrderAction(o.Side) == OrderActionSell &&
				currentPrice < o.StopPrice {
				slippage := 0.01 + (o.StopPrice * 0.000025)
				if execution, err = b.CreateExecution(symbolDetails, o, o.StopPrice-slippage); err != nil {
					return err
				}
				b.logger.LogInfo(
					"broker", "execution,%d,%s,%s,%s,%d,%.2f,%.2f",
					o.Id, o.Side, o.Type, o.Symbol, o.TotalQuantity,
					o.AvgExecPrice, execution.TotalCost,
				)
			} else if o.Type == OrderTypeLimit &&
				OrderAction(o.Side) == OrderActionSell &&
				currentPrice > o.LimitPrice {
				slippage := 0.01 + (o.StopPrice * 0.000025)
				if execution, err = b.CreateExecution(symbolDetails, o, o.LimitPrice+slippage); err != nil {
					return err
				}
				b.logger.LogInfo(
					"broker", "execution,%d,%s,%s,%s,%d,%.2f,%.2f",
					o.Id, o.Side, o.Type, o.Symbol, o.TotalQuantity,
					o.AvgExecPrice, execution.TotalCost,
				)
			} else if OrderAction(o.Side) == OrderActionBuy {
				panic("Buy|Limit and Buy|Stop orders are not yet supported.")
			}
		}
	}
	return nil
}

func (b *PaperBroker) CancelOrder(orderId int) error {
	for _, o := range b.orders {
		if o.Id == orderId {
			if o.State != "Accepted" {
				return errors.New("CancelOrder: Can't cancel an order that already executed")
			}
			o.State = "Canceled"
			b.logger.LogInfo(
				"broker", "cancel,%d,%s,%s",
				o.Id, o.Side, o.Type,
			)
			return nil
		}
	}
	return errors.New("CancelOrder: Can't find order #" + strconv.Itoa(orderId))
}

func (b *PaperBroker) Balance() (*BrokerBalance, error) {
	var marketValue float64 = 0
	for _, p := range b.positions {
		marketValue += p.CurrentMarketValue
	}

	balance := &BrokerBalance{
		Currency:    "USD",
		Cash:        b.cash,
		MarketValue: marketValue,
	}
	return balance, nil
}

func (b *PaperBroker) Positions() ([]*BrokerPosition, error) {
	var err error
	for _, p := range b.positions {
		p.CurrentPrice, err = b.currentPrice(p.SymbolId)
		if err != nil {
			continue
		}
		p.CurrentMarketValue = float64(p.OpenQuantity) * p.CurrentPrice
		p.ClosedPnL = 0
		p.OpenPnL = p.CurrentMarketValue - (float64(p.OpenQuantity) * p.AverageEntryPrice)
	}
	return b.positions, nil
}

func (b *PaperBroker) Executions() ([]*BrokerExecution, error) {
	return b.executions, nil
}

func (b *PaperBroker) Orders() ([]*BrokerOrder, error) {
	return b.orders, nil
}

func (b *PaperBroker) LastBalance() *BrokerBalance {
	balance, _ := b.Balance()
	return balance
}

func (b *PaperBroker) LastPositions() []*BrokerPosition {
	return b.positions
}

func (b *PaperBroker) LastExecutions() []*BrokerExecution {
	return b.executions
}

func (b *PaperBroker) LastOrders() []*BrokerOrder {
	return b.orders
}
