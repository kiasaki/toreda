package main

import "time"

type OrderManager struct {
	broker  Broker
	logger  Logger
	targets map[int]int64
}

func NewOrderManager(broker Broker, logger Logger) *OrderManager {
	return &OrderManager{
		broker:  broker,
		logger:  logger,
		targets: map[int]int64{},
	}
}

func (om *OrderManager) Ensure(symId int, qty int64) error {
	// Save target so we know what we want to achieve when orders get cancelled
	om.targets[symId] = qty

	// TODO looking at positions is not enough to determine the current state of
	// things, we need to also looks at pending orders
	positions := om.broker.LastPositions()

	var positionQty int64
	for _, p := range positions {
		if p.SymbolId == symId {
			positionQty = p.OpenQuantity
		}
	}

	if positionQty > qty {
		_, err := om.broker.CreateOrder(symId, OrderActionSell, OrderTypeMarket, 0, positionQty-qty)
		return err
	}
	if positionQty < qty {
		_, err := om.broker.CreateOrder(symId, OrderActionBuy, OrderTypeMarket, 0, qty-positionQty)
		return err
	}
	return nil
}

func (om *OrderManager) CancelPendingOrders() error {
	orders := om.broker.LastOrders()
	for _, o := range orders {
		if o.IsPending() {
			if err := om.broker.CancelOrder(o.Id); err != nil {
				return err
			}
		}
	}
	return nil
}

func (om *OrderManager) CancelAllStops(symId int) error {
	orders := om.broker.LastOrders()
	for _, o := range orders {
		if o.SymbolId == symId && o.Type == OrderTypeStop && o.State == "Accepted" {
			if err := om.broker.CancelOrder(o.Id); err != nil {
				return err
			}
		}
	}
	return nil
}

func (om *OrderManager) CancelAllLimits(symId int) error {
	orders := om.broker.LastOrders()
	for _, o := range orders {
		if o.SymbolId == symId && o.Type == OrderTypeLimit && o.State == "Accepted" {
			if err := om.broker.CancelOrder(o.Id); err != nil {
				return err
			}
		}
	}
	return nil
}

func (om *OrderManager) CurrentPositionFor(symId int) *BrokerPosition {
	positions := om.broker.LastPositions()

	var position *BrokerPosition
	for _, p := range positions {
		if p.SymbolId == symId {
			position = p
		}
	}

	if position == nil {
		// Return an empty position
		return &BrokerPosition{
			SymbolId: symId,
		}
	}

	return position
}

func (om *OrderManager) CurrentPositionForIncludingPending(symbolId int) *BrokerPosition {
	position := om.CurrentPositionFor(symbolId)
	for _, order := range om.broker.LastOrders() {
		if order.SymbolId == symbolId &&
			order.Side == string(OrderActionBuy) &&
			(order.Type == OrderTypeMarket || order.Type == OrderTypeLimit) {
			position.OpenQuantity += order.OpenQuantity
		}
	}
	return position
}

func (om *OrderManager) Buy(symId int, quantity int64, stop float64) error {
	_, err := om.broker.CreateOrder(symId, OrderActionBuy, OrderTypeMarket, 0, quantity)
	if err != nil {
		return err
	}
	// TODO remove
	// Give QT some time to process the fact we now have a position?
	time.Sleep(15 * time.Millisecond)
	_, err = om.broker.CreateOrder(symId, OrderActionSell, OrderTypeStop, stop, quantity)
	return err
}

// Sells a whole position all while cancelling any stop or limit on them
func (om *OrderManager) SellAll(symId int) error {
	position := om.CurrentPositionFor(symId)
	if position.OpenQuantity <= 0 {
		// Nothing to do
		return nil
	}

	if err := om.CancelAllStops(symId); err != nil {
		return err
	}
	if err := om.CancelAllLimits(symId); err != nil {
		return err
	}
	_, err := om.broker.CreateOrder(symId, OrderActionSell, OrderTypeMarket, 0, position.OpenQuantity)
	return err
}
