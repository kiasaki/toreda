package main

import (
	"errors"
	"fmt"
	"strconv"
)

type QTBroker struct {
	tm             TradingManager
	logger         Logger
	api            *QTApi
	accountId      string
	lastBalance    *BrokerBalance
	lastPositions  []*BrokerPosition
	lastExecutions []*BrokerExecution
	lastOrders     []*BrokerOrder
}

func NewQTBroker(tradingManager TradingManager, logger Logger, accountId string) *QTBroker {
	return &QTBroker{
		tm:             tradingManager,
		logger:         logger,
		api:            NewQTApi(),
		accountId:      accountId,
		lastBalance:    &BrokerBalance{},
		lastPositions:  []*BrokerPosition{},
		lastExecutions: []*BrokerExecution{},
		lastOrders:     []*BrokerOrder{},
	}
}

func (b *QTBroker) CreateOrder(
	symId int, action OrderAction, typ OrderType, limitOrStopPrice float64, quantity int64,
) (*BrokerOrder, error) {
	b.logger.LogInfo("broker", "create_order,%d,%d,%s,%s,%.2f", symId, quantity, action, typ, limitOrStopPrice)
	path := "v1/accounts/" + b.accountId + "/orders"
	data := map[string]interface{}{
		"symbolId":       symId,
		"quantity":       quantity,
		"action":         string(action),
		"orderType":      string(typ),
		"timeInForce":    "GoodTillCanceled",
		"primaryRoute":   "AUTO",
		"secondaryRoute": "AUTO",
	}
	if typ == OrderTypeStop {
		data["stopPrice"] = froundn(limitOrStopPrice, 4)
	}
	if typ == OrderTypeLimit {
		data["limitPrice"] = froundn(limitOrStopPrice, 4)
	}
	response := QTApiOrdersResponse{}
	err := b.api.Request(APIRequest{Method: "POST", Path: path, Data: data}, &response)
	if err != nil {
		return nil, err
	}

	if len(response.Orders) <= 0 {
		return nil, nil
	}

	order := response.Orders[0]
	if order.State == "Rejected" || order.State == "Failed" {
		return nil, errors.New("order rejected")
	}

	return order, nil
}

func (b *QTBroker) CancelOrder(orderId int) error {
	b.logger.LogInfo("broker", "cancel_order,%d", orderId)
	response := struct{ orderId int }{}
	path := "v1/accounts/" + b.accountId + "/orders/" + strconv.Itoa(orderId)
	return b.api.Request(APIRequest{Method: "DELETE", Path: path}, &response)
}

type QTApiBalancesResponse struct {
	PerCurrencyBalances    []*BrokerBalance `json:"perCurrencyBalances"`
	CombinedBalances       []*BrokerBalance `json:"combinedBalances"`
	SodPerCurrencyBalances []*BrokerBalance `json:"sodPerCurrencyBalances"`
	SodCombinedBalances    []*BrokerBalance `json:"sodCombinedBalances"`
}

func (b *QTBroker) Balance() (*BrokerBalance, error) {
	b.logger.LogDebug("broker", "fetching_balance,%s", b.accountId)
	response := QTApiBalancesResponse{}
	path := "v1/accounts/" + b.accountId + "/balances"
	err := b.api.Request(APIRequest{Path: path}, &response)
	if err != nil {
		return nil, err
	}

	var balance *BrokerBalance
	for _, b := range response.CombinedBalances {
		if b.Currency == "USD" {
			balance = b
		}
	}
	for _, b := range response.SodCombinedBalances {
		if b.Currency == "USD" && balance != nil {
			balance.StartOfDayCash = b.Cash
		}
	}
	if balance == nil {
		return nil, errors.New("qt_broker: can't find USD combined balance")
	}

	if b.tm != nil {
		env, date := b.tm.Environment(), b.tm.Now().Format(dateFormat)
		filePath := fmt.Sprintf("data/run/%s/%s/balance.json", env, date)
		err = writeJsonFile(filePath, balance)
	}
	b.lastBalance = balance
	return balance, err
}

type QTApiPositionsResponse struct {
	Positions []*BrokerPosition `json:"positions"`
}

func (b *QTBroker) Positions() ([]*BrokerPosition, error) {
	b.logger.LogDebug("broker", "fetching_positions,%s", b.accountId)
	response := QTApiPositionsResponse{}
	path := "v1/accounts/" + b.accountId + "/positions"
	err := b.api.Request(APIRequest{Path: path}, &response)
	if err != nil {
		return nil, err
	}
	if b.tm != nil {
		env, date := b.tm.Environment(), b.tm.Now().Format(dateFormat)
		filePath := fmt.Sprintf("data/run/%s/%s/positions.json", env, date)
		err = writeJsonFile(filePath, response.Positions)
	}
	b.lastPositions = response.Positions
	return response.Positions, err
}

type QTApiExecutionsResponse struct {
	Executions []*BrokerExecution `json:"executions"`
}

func (b *QTBroker) Executions() ([]*BrokerExecution, error) {
	b.logger.LogDebug("broker", "fetching_executions,%s", b.accountId)
	response := QTApiExecutionsResponse{}
	path := "v1/accounts/" + b.accountId + "/executions"
	err := b.api.Request(APIRequest{Path: path}, &response)
	if err != nil {
		return nil, err
	}
	if b.tm != nil {
		env, date := b.tm.Environment(), b.tm.Now().Format(dateFormat)
		filePath := fmt.Sprintf("data/run/%s/%s/executions.json", env, date)
		err = writeJsonFile(filePath, response.Executions)
	}
	b.lastExecutions = response.Executions
	return response.Executions, err
}

type QTApiOrdersResponse struct {
	Orders []*BrokerOrder `json:"orders"`
}

func (b *QTBroker) Orders() ([]*BrokerOrder, error) {
	b.logger.LogDebug("broker", "fetching_orders,%s", b.accountId)
	response := QTApiOrdersResponse{}
	path := "v1/accounts/" + b.accountId + "/orders?stateFilter=All"
	err := b.api.Request(APIRequest{Path: path}, &response)
	if err != nil {
		return nil, err
	}
	if b.tm != nil {
		env, date := b.tm.Environment(), b.tm.Now().Format(dateFormat)
		filePath := fmt.Sprintf("data/run/%s/%s/orders.json", env, date)
		err = writeJsonFile(filePath, response.Orders)
	}
	b.lastOrders = response.Orders
	return response.Orders, err
}

func (b *QTBroker) LastBalance() *BrokerBalance {
	return b.lastBalance
}

func (b *QTBroker) LastPositions() []*BrokerPosition {
	return b.lastPositions
}

func (b *QTBroker) LastExecutions() []*BrokerExecution {
	return b.lastExecutions
}

func (b *QTBroker) LastOrders() []*BrokerOrder {
	return b.lastOrders
}
