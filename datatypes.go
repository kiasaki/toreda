package main

import "time"

type CandleInterval string

const (
	CandleIntervalOneMinute      CandleInterval = "OneMinute"
	CandleIntervalTwoMinute                     = "TwoMinutes"
	CandleIntervalFiveMinute                    = "FiveMinutes"
	CandleIntervalFifteenMinutes                = "FifteenMinutes"
	CandleIntervalHalfHour                      = "HalfHour"
	CandleIntervalOneHour                       = "OneHour"
	CandleIntervalOneDay                        = "OneDay"
	CandleIntervalOneWeek                       = "OneWeek"
)

type OrderAction string

const (
	OrderActionBuy  OrderAction = "Buy"
	OrderActionSell             = "Sell"
)

type OrderType string

const (
	OrderTypeMarket OrderType = "Market"
	OrderTypeLimit            = "Limit"
	OrderTypeStop             = "Stop"
)

type Revenue struct {
	Day    string
	Profit float64
}

type SymbolDetails struct {
	Symbol            string    `json:"symbol"`
	SymbolId          int       `json:"symbolId"`
	PrevDayClosePrice float64   `json:"prevDayClosePrice"`
	HighPrice52       float64   `json:"highPrice52"`
	LowPrice52        float64   `json:"lowPrice52"`
	AverageVol3Months int64     `json:"averageVol3Months"`
	AverageVol20Days  int64     `json:"averageVol20Days"`
	OutstandingShares int64     `json:"outstandingShares"`
	EPS               float64   `json:"eps"`
	PE                float64   `json:"pe"`
	Dividend          float64   `json:"dividend"`
	Yield             float64   `json:"yield"`
	ExDate            time.Time `json:"exDate"` // Dividend ex date
	MarketCap         int64     `json:"marketCap"`
	TradeUnit         int       `json:"tradeUnit"`
	ListingExchange   string    `json:"listingExchange"`
	Description       string    `json:"description"`
	SecurityType      string    `json:"securityType"`
	DividendDate      time.Time `json:"dividendDate"`
	IsTradable        bool      `json:"isTradable"`
	IsQuotable        bool      `json:"isQuotable"`
	IndustrySector    string    `json:"industrySector"`
	IndustryGroup     string    `json:"industryGroup"`
	IndustrySubGroup  string    `json:"industrySubGroup"`
}

type SymbolQuote struct {
	Symbol         string  `json:"symbol"`
	SymbolId       int     `json:"symbolId"`
	BidPrice       float64 `json:"bidPrice"`
	BidSize        int64   `json:"bidSize"`
	AskPrice       float64 `json:"askPrice"`
	AskSize        int64   `json:"askSize"`
	LastTradeTrHrs float64 `json:"lastTradeTrHrs"`
	LastTradePrice float64 `json:"lastTradePrice"`
	LastTradeSize  int64   `json:"lastTradeSize"`
	Volume         int64   `json:"volume"`
	OpenPrice      float64 `json:"openPrice"`
	HighPrice      float64 `json:"highPrice"`
	LowPrice       float64 `json:"lowPrice"`
	Delay          int     `json:"delay"`
	IsHalted       bool    `json:"isHalted"`
}

type SymbolCandle struct {
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
	Low    float64   `json:"low"`
	High   float64   `json:"high"`
	Open   float64   `json:"open"`
	Close  float64   `json:"close"`
	Volume int64     `json:"volume"`
}

func (c *SymbolCandle) Red() bool {
	return c.Close < c.Open
}

func (c *SymbolCandle) Green() bool {
	return c.Close >= c.Open
}

type BrokerBalance struct {
	Currency       string  `json:"currency"`
	Cash           float64 `json:"cash"`
	MarketValue    float64 `json:"marketValue"`
	StartOfDayCash float64 `json:"startOfDayCash"`
}

type BrokerPosition struct {
	Symbol             string  `json:"symbol"`
	SymbolId           int     `json:"symbolId"`
	OpenQuantity       int64   `json:"openQuantity"`
	CurrentMarketValue float64 `json:"currentMarketValue"`
	CurrentPrice       float64 `json:"currentPrice"`
	AverageEntryPrice  float64 `json:"averageEntryPrice"`
	ClosedPnL          float64 `json:"closedPnL"`
	OpenPnL            float64 `json:"openPnL"`
}

type BrokerExecution struct {
	Id                       int       `json:"id"`
	OrderId                  int       `json:"orderId"`
	Timestamp                time.Time `json:"timestamp"`
	Symbol                   string    `json:"symbol"`
	SymbolId                 int       `json:"symbolId"`
	Quantity                 int64     `json:"quantity"`
	Side                     string    `json:"side"`
	Price                    float64   `json:"price"`
	TotalCost                float64   `json:"totalCost"`
	OrderPlacementCommission float64   `json:"orderPlacementCommission"`
	Commission               float64   `json:"commission"`
	ExecutionFee             float64   `json:"executionFee"`
	SecFee                   float64   `json:"secFee"`
}

type BrokerOrder struct {
	Id               int       `json:"id"`
	Symbol           string    `json:"symbol"`
	SymbolId         int       `json:"symbolId"`
	CreationTime     time.Time `json:"creationTime"`
	UpdateTime       time.Time `json:"updateTime"`
	TotalQuantity    int64     `json:"totalQuantity"`
	OpenQuantity     int64     `json:"openQuantity"`
	FilledQuantity   int64     `json:"filledQuantity"`
	CanceledQuantity int64     `json:"canceledQuantity"`
	Side             string    `json:"side"`
	Type             OrderType `json:"orderType"`
	LimitPrice       float64   `json:"limitPrice"`
	StopPrice        float64   `json:"stopPrice"`
	AvgExecPrice     float64   `json:"avgExecPrice"`
	State            string    `json:"state"`
}

func (o *BrokerOrder) IsPending() bool {
	return o.State == "Queued" ||
		o.State == "Pending" ||
		o.State == "Accepted" ||
		o.State == "Partial" ||
		o.State == "Replaced" ||
		o.State == "ReplacePending" ||
		o.State == "CancelPending"
}

type Datasource interface {
	Details(symbolName string) (*SymbolDetails, error)
	Quote(id int) (*SymbolQuote, error)
	Candles(id int, start, end time.Time, interval CandleInterval) ([]*SymbolCandle, error)
}

type Broker interface {
	// Watch for X-RateLimit-Remaining and X-RateLimit-Reset, stopping at 50 calls left
	CreateOrder(symId int, action OrderAction, typ OrderType, limitOrStopPrice float64, qty int64) (*BrokerOrder, error)
	CancelOrder(orderId int) error
	Balance() (*BrokerBalance, error)
	Positions() ([]*BrokerPosition, error)
	Executions() ([]*BrokerExecution, error)
	Orders() ([]*BrokerOrder, error)

	LastBalance() *BrokerBalance
	LastPositions() []*BrokerPosition
	LastExecutions() []*BrokerExecution
	LastOrders() []*BrokerOrder
}
