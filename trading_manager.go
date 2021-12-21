package main

import (
	"errors"
	"os"
	"time"
)

type TradingManagerEnvironment string

const (
	TradingManagerEnvironmentPaper      TradingManagerEnvironment = "paper"
	TradingManagerEnvironmentStaging                              = "staging"
	TradingManagerEnvironmentProduction                           = "production"
)

type TradingManagerState string

const (
	TradingManagerStateStarting TradingManagerState = "starting"
	TradingManagerStateRunning                      = "running"
	TradingManagerStateDone                         = "done"
	TradingManagerStateFailing                      = "failing"
	TradingManagerStateFailed                       = "failed"
	TradingManagerStateStopping                     = "stopping"
	TradingManagerStateStopped                      = "stopped"
)

type TradingManager interface {
	Now() time.Time
	State() TradingManagerState
	Environment() TradingManagerEnvironment
	Broker() Broker
	Datasource() Datasource
	WaitForState(states ...TradingManagerState)
	Start() error
	Stop()
}

type TradingManagerV1 struct {
	environment  TradingManagerEnvironment
	state        TradingManagerState
	now          time.Time
	start        time.Time
	end          time.Time
	strategyName string
	logger       Logger
	strategy     Strategy
	broker       Broker
	datasource   Datasource
	orderManager *OrderManager
}

func NewTradingManagerV1(
	environment TradingManagerEnvironment, strategyName, strategyConfig string,
) *TradingManagerV1 {
	tm := &TradingManagerV1{
		environment: environment,
		state:       TradingManagerStateStarting,
		now:         time.Now().In(timeLocation),
	}
	truncateLogFiles := environment != TradingManagerEnvironmentProduction
	tm.logger = NewFileLogger(tm, "data/run/", string(environment), truncateLogFiles)

	if environment == TradingManagerEnvironmentPaper {
		tm.datasource = NewPaperDatasource(tm, tm.logger)
		tm.broker = NewPaperBroker(tm, tm.logger)
	} else if environment == TradingManagerEnvironmentStaging {
		tm.datasource = NewQTDatasource(tm.logger)
		tm.broker = NewPaperBroker(tm, tm.logger)
	} else if environment == TradingManagerEnvironmentProduction {
		tm.datasource = NewQTDatasource(tm.logger)
		tm.broker = NewQTBroker(tm, tm.logger, os.Getenv("QT_ACCOUNT_ID"))
	}

	tm.orderManager = NewOrderManager(tm.broker, tm.logger)
	tm.loadStrategy(strategyName, strategyConfig)

	return tm
}

func (tm *TradingManagerV1) Now() time.Time {
	return tm.now
}

func (tm *TradingManagerV1) State() TradingManagerState {
	return tm.state
}

func (tm *TradingManagerV1) Environment() TradingManagerEnvironment {
	return tm.environment
}

func (tm *TradingManagerV1) Broker() Broker {
	return tm.broker
}

func (tm *TradingManagerV1) Datasource() Datasource {
	return tm.datasource
}

func (tm *TradingManagerV1) loadStrategy(strategyName, strategyConfig string) {
	switch strategyName {
	case "rsi":
		tm.strategy = NewRsiStrategy()
	case "long_ma":
		tm.strategy = NewLongMAStrategy()
	case "bar_color":
		tm.strategy = NewBarColorFollowStrategy()
	case "follow_nasdaq":
		tm.strategy = NewFollowNasdaqStrategy()
	case "gapngo":
		tm.strategy = NewGapNGoStrategy()
	default:
		panic("Unknown strategy: " + strategyName)
	}
	tm.strategyName = strategyName
}

func (tm *TradingManagerV1) WaitForState(states ...TradingManagerState) {
	for {
		for _, s := range states {
			if tm.state == s {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func (tm *TradingManagerV1) tick() {
	if _, err := tm.broker.Positions(); err != nil {
		tm.logger.LogError("trading_manager", "broker positions: %s", err.Error())
		tm.state = TradingManagerStateFailing
		return
	}
	if _, err := tm.broker.Executions(); err != nil {
		tm.logger.LogError("trading_manager", "broker executions: %s", err.Error())
		tm.state = TradingManagerStateFailing
		return
	}
	if _, err := tm.broker.Orders(); err != nil {
		tm.logger.LogError("trading_manager", "broker orders: %s", err.Error())
		tm.state = TradingManagerStateFailing
		return
	}
	if _, err := tm.broker.Balance(); err != nil {
		tm.logger.LogError("trading_manager", "broker balance: %s", err.Error())
		tm.state = TradingManagerStateFailing
		return
	}
	if err := tm.strategy.Run(tm.now, tm.datasource, tm.broker, tm.orderManager); err != nil {
		tm.logger.LogError("trading_manager", "strategy: %s", err.Error())
		tm.state = TradingManagerStateFailing
		return
	}

	// Handle Limit and Stop orders done with the PaperBroker
	if tm.environment != TradingManagerEnvironmentProduction {
		if err := tm.broker.(*PaperBroker).CheckLimitAndStopOrders(); err != nil {
			tm.logger.LogError("trading_manager", "broker limit/stops: %s", err.Error())
			tm.state = TradingManagerStateFailing
			return
		}
	}

	// Handle Circuit Breakers
	if tm.environment == TradingManagerEnvironmentProduction {
		balance := tm.broker.LastBalance()
		if balance != nil && balance.Cash != 0 {
			if 1-((balance.Cash+balance.MarketValue)/balance.StartOfDayCash) > 0.5 {
				// Wowza, we're down 10%. Let's stop right now.
				tm.logger.LogError("trading_manager", "circuit_breaker: we're down >5% ABORTING")
				tm.state = TradingManagerStateFailing
			}
		}
	}
}

func (tm *TradingManagerV1) loopCore() bool {
	switch tm.state {
	case TradingManagerStateStarting:
		tm.state = TradingManagerStateRunning
		tm.logger.LogInfo("trading_manager", "done starting")
	case TradingManagerStateRunning:
		// We don't play weekends
		if tm.now.Weekday() == time.Sunday && tm.now.Weekday() == time.Saturday {
			return false
		}
		// We don't play after hours
		h, m, _ := tm.now.Clock()
		hm := (h * 60) + m
		if hm < 9*60 || hm > 16.5*60 {
			return false
		}
		tm.tick()
	case TradingManagerStateDone:
		return true
	case TradingManagerStateStopping:
		tm.logger.LogInfo("trading_manager", "start stopping")
		tm.abort()
		tm.logger.LogInfo("trading_manager", "done stopping")
		tm.state = TradingManagerStateStopped
	case TradingManagerStateStopped:
		return true
	case TradingManagerStateFailing:
		tm.logger.LogInfo("trading_manager", "start failing")
		tm.abort()
		tm.logger.LogInfo("trading_manager", "done failing")
		tm.state = TradingManagerStateFailed
	case TradingManagerStateFailed:
		return true
	default:
		panic("unknown trading manager state")
	}
	return false
}

func (tm *TradingManagerV1) loopPaper() {
	tm.now = tm.start.Truncate(time.Minute).Add(-1 * time.Second)
	for tm.now.Before(tm.end) {
		if stop := tm.loopCore(); stop {
			break
		}
		tm.now = tm.now.Add(1 * time.Minute)
	}
	tm.state = TradingManagerStateDone
}

func (tm *TradingManagerV1) loopStaging() {
	tm.now = tm.start.Truncate(time.Minute).Add(-1 * time.Second)
	for tm.now.Before(tm.end) {
		if stop := tm.loopCore(); stop {
			break
		}
		tm.now = tm.now.Add(1 * time.Minute)
		time.Sleep(300 * time.Millisecond) // Run 4h30m in ~2m15s
	}
	tm.state = TradingManagerStateDone
}

func (tm *TradingManagerV1) loopProduction() {
	var paniced = false
	defer func() {
		if r := recover(); r != nil {
			if paniced {
				tm.logger.LogError("trading_manager", "paniced while handling panic, EXITING NOW")
				os.Exit(1)
			}
			paniced = true
			tm.logger.LogError("trading_manager", "paniced ABORTING")
			if tm.state != TradingManagerStateFailed {
				tm.state = TradingManagerStateFailing
			}
			tm.WaitForState(TradingManagerStateFailed)
			os.Exit(1)
		}
	}()
	for {
		tm.now = time.Now().In(timeLocation)
		if stop := tm.loopCore(); stop {
			break
		}
		time.Sleep(10 * time.Second)
	}
}

func (tm *TradingManagerV1) loop() {
	switch tm.environment {
	case TradingManagerEnvironmentPaper:
		tm.loopPaper()
	case TradingManagerEnvironmentStaging:
		tm.loopStaging()
	case TradingManagerEnvironmentProduction:
		tm.loopProduction()
	default:
		panic("unknown trading manager environment")
	}
}

func (tm *TradingManagerV1) Start() error {
	if tm.State() != TradingManagerStateStarting {
		return errors.New("can't start when state is: " + string(tm.State()))
	}
	go tm.loop()
	return nil
}

func (tm *TradingManagerV1) abort() {
	// When on prod, cancel orders & sell everything
	if tm.environment == TradingManagerEnvironmentProduction {
		encounteredError := false

		// Cancel any outstanding orders
		orders, err := tm.broker.Orders()
		if err != nil {
			tm.logger.LogError("trading_manager", "aborting: error fetching orders: %s", err.Error())
			// Just pretend we didn't find any order and try moving on to positions
			encounteredError = true
			orders = []*BrokerOrder{}
		}

		for _, o := range orders {
			if o.IsPending() {
				err = tm.broker.CancelOrder(o.Id)
				if err != nil {
					tm.logger.LogError("trading_manager", "aborting: error canceling order: %s", err.Error())
					encounteredError = true
				}
			}
		}

		// Now let's liquidate all our open positions
		positions, err := tm.broker.Positions()
		if err != nil {
			tm.logger.LogError("trading_manager", "aborting: error fetching positions: %s", err.Error())
			// Just pretend we didn't find any positions and move on
			positions = []*BrokerPosition{}
			encounteredError = true
		}

		for _, p := range positions {
			// TODO handle liquidating short positions with buy-to-cover's
			if p.OpenQuantity > 0 {
				_, err = tm.broker.CreateOrder(p.SymbolId, OrderActionSell, OrderTypeMarket, 0, p.OpenQuantity)
				if err != nil {
					tm.logger.LogError("trading_manager", "aborting: error creating sell order: %s", err.Error())
					encounteredError = true
				}
			}
		}

		if encounteredError {
			// Ok let's die now so that server uptime alert's trigger
			// As we encountered an error we probably missed a cancel/sell order
			os.Exit(1)
		}
	}
}

func (tm *TradingManagerV1) Stop() {
	if tm.state == TradingManagerStateStarting ||
		tm.state == TradingManagerStateDone ||
		tm.state == TradingManagerStateStopped ||
		tm.state == TradingManagerStateFailed {
		return
	}

	tm.state = TradingManagerStateStopping
	tm.WaitForState(TradingManagerStateFailed, TradingManagerStateStopped)
}
