package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type H map[string]interface{}

var (
	rootHandler          http.Handler
	indexFileContents    []byte
	serverPaperTM        *TradingManagerV1
	serverPaperTMRW      sync.Mutex
	serverStagingTM      *TradingManagerV1
	serverStagingTMRW    sync.Mutex
	serverProductionTM   *TradingManagerV1
	serverProductionTMRW sync.Mutex
)

func startServer() {
	indexFile, err := os.Open("static/index.html")
	if err != nil {
		log.Fatalln(err)
	}
	bytes, err := ioutil.ReadAll(indexFile)
	if err != nil {
		log.Fatalln(err)
	}
	indexFileContents = bytes

	serverPaperTM = NewTradingManagerV1(TradingManagerEnvironmentPaper, "long_ma", "")
	serverStagingTM = NewTradingManagerV1(TradingManagerEnvironmentStaging, "long_ma", "")
	serverProductionTM = NewTradingManagerV1(TradingManagerEnvironmentProduction, "long_ma", "")

	// Handle Ctrl-C and exit cleanly
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		go func() {
			<-sigs
			// Ctrl-C raised a second time exit NOW
			serverProductionTM.logger.LogError("trading_manager", "got EXIT/TERM signal twice, EXITING NOW")
			os.Exit(1)
		}()
		serverProductionTM.logger.LogError("trading_manager", "got EXIT/TERM signal, aborting")
		serverProductionTM.Stop()
		os.Exit(1)
	}()

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/data/1m-1d/", handleData1m1d)
	http.HandleFunc("/data/details", handleDataDetails)
	http.HandleFunc("/data/run/paper", handleDataRun)
	http.HandleFunc("/data/run/staging", handleDataRun)
	http.HandleFunc("/data/run/production", handleDataRun)
	http.HandleFunc("/data/account", handleDataAccount)
	http.HandleFunc("/data/logs", handleDataLogs)
	http.HandleFunc("/actions/run/paper", handleActionsRunPaper)
	http.HandleFunc("/actions/start/production", handleActionsStartProduction)
	http.HandleFunc("/actions/stop/production", handleActionsStopProduction)

	rootHandler = http.DefaultServeMux
	basicAuthCredentials := os.Getenv("TOREDA_BASIC_AUTH")
	if basicAuthCredentials == "" {
		log.Fatalln("Missing TOREDA_BASIC_AUTH environment variable")
	}
	credentialsParts := strings.SplitN(basicAuthCredentials, ":", 2)
	rootHandler = BasicAuth(credentialsParts[0], credentialsParts[1])(rootHandler)

	s := &http.Server{
		Addr:    ":10888",
		Handler: http.HandlerFunc(serverRootHandler),
	}
	log.Println("Starting server on port 10888")
	log.Fatal(s.ListenAndServe())
}

func serverRootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path[:min(8, len(r.URL.Path))] != "/static/" {
		log.Printf("%s %s\n", r.Method, r.URL.Path)
	}
	rootHandler.ServeHTTP(w, r)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		w.Header().Set("Content-Type", "text/html")
		w.Write(indexFileContents)
		return
	}
	handleNotFound(w, r)
}

func handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(404)
	w.Header().Set("Content-Type", "text/html")
	styles := `
	  <style>
		body {
		  margin: 0;
		  background: #fbfbfd;
		  padding-top: 300px;
		  font-family: monospace, serif;
		  font-weight: 300;
		  text-align:center;
		}
	  </style>
	`
	contents := `
	  <h1>404</h1>
	  <h3>Page Not Found</h3>
	`
	w.Write([]byte(styles + contents))
}

func handleData1m1d(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")

	// Symbol
	symbolName := parts[3]
	symbolDetails := findSymbolDetailsByName(symbolName)
	if symbolDetails == nil {
		renderError(w, "Can't find symbol named: "+symbolName)
		return
	}

	// Dates
	dateString := parts[4]
	start, err := time.ParseInLocation("2006-01-02 15:04", dateString+" 00:00", timeLocation)
	if err != nil {
		renderError(w, err.Error())
		return
	}
	end := start.Add(24 * time.Hour)

	candles, err := serverPaperTM.datasource.Candles(
		symbolDetails.SymbolId, start, end, CandleIntervalOneMinute,
	)
	if err != nil {
		renderError(w, err.Error())
		return
	}
	renderJson(w, candles)

}

func handleDataDetails(w http.ResponseWriter, r *http.Request) {
	renderJson(w, allSymbols)
}

func handleDataRun(w http.ResponseWriter, r *http.Request) {
	var tradingManager *TradingManagerV1
	switch r.URL.Path[len("/data/run/"):] {
	case "paper":
		tradingManager = serverPaperTM
	case "staging":
		tradingManager = serverStagingTM
	case "production":
		tradingManager = serverProductionTM
	default:
		panic("unreachable")
	}
	renderJson(w, H{
		"time":       tradingManager.Now(),
		"state":      tradingManager.State(),
		"strategy":   tradingManager.strategyName,
		"balance":    tradingManager.broker.LastBalance(),
		"positions":  tradingManager.broker.LastPositions(),
		"orders":     tradingManager.broker.LastOrders(),
		"executions": tradingManager.broker.LastExecutions(),
	})
}

func handleDataAccount(w http.ResponseWriter, r *http.Request) {
	accountId := r.URL.Query().Get("accountId")
	broker := NewQTBroker(nil, NewConsoleLogger(), accountId)
	balance, err := broker.Balance()
	if err != nil {
		renderError(w, err.Error())
		return
	}
	positions, err := broker.Positions()
	if err != nil {
		renderError(w, err.Error())
		return
	}
	orders, err := broker.Orders()
	if err != nil {
		renderError(w, err.Error())
		return
	}
	executions, err := broker.Executions()
	if err != nil {
		renderError(w, err.Error())
		return
	}
	renderJson(w, H{
		"balance":    balance,
		"positions":  positions,
		"orders":     orders,
		"executions": executions,
	})
}

func handleDataLogs(w http.ResponseWriter, r *http.Request) {
	var err error
	var file *os.File
	environment := TradingManagerEnvironment(r.URL.Query().Get("environment"))

	if environment == TradingManagerEnvironmentPaper {
		file, err = os.Open("data/run/paper/log.txt")
	} else if environment == TradingManagerEnvironmentStaging {
		file, err = os.Open("data/run/staging/log.txt")
	} else if environment == TradingManagerEnvironmentProduction {
		date := time.Now().In(timeLocation).Format("2006-01-02")
		file, err = os.Open("data/run/production/" + date + "/log.txt")
	} else {
		renderError(w, "Unknown environment: "+string(environment))
	}

	if err != nil {
		renderError(w, err.Error())
		return
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		renderError(w, err.Error())
		return
	}
	w.Write(bytes)
}

func handleActionsRunPaper(w http.ResponseWriter, r *http.Request) {
	serverPaperTMRW.Lock()
	defer serverPaperTMRW.Unlock()

	var values = map[string]string{}
	err := json.NewDecoder(r.Body).Decode(&values)
	if err != nil {
		renderError(w, err.Error())
		return
	}

	cash, err := strconv.ParseFloat(values["cash"], 64)
	if err != nil {
		renderError(w, err.Error())
		return
	}
	start, err := time.ParseInLocation("2006-01-02 15:04", values["start"]+" 09:00", timeLocation)
	if err != nil {
		renderError(w, err.Error())
		return
	}
	end, err := time.ParseInLocation("2006-01-02 15:04", values["end"]+" 16:30", timeLocation)
	if err != nil {
		renderError(w, err.Error())
		return
	}

	serverPaperTM = NewTradingManagerV1(TradingManagerEnvironmentPaper, values["strategy"], "")
	serverPaperTM.broker.(*PaperBroker).cash = cash
	serverPaperTM.start = start
	serverPaperTM.end = end

	if err := serverPaperTM.Start(); err != nil {
		renderError(w, err.Error())
		return
	}

	serverPaperTM.WaitForState(TradingManagerStateDone, TradingManagerStateFailed)

	renderJson(w, H{})
}

func handleActionsStartProduction(w http.ResponseWriter, r *http.Request) {
	serverProductionTMRW.Lock()
	defer serverProductionTMRW.Unlock()

	var values = map[string]string{}
	err := json.NewDecoder(r.Body).Decode(&values)
	if err != nil {
		renderError(w, err.Error())
		return
	}

	// Reset if failed
	if serverProductionTM.State() == TradingManagerStateFailed ||
		serverProductionTM.State() == TradingManagerStateStopped {
		serverProductionTM = NewTradingManagerV1(TradingManagerEnvironmentProduction, values["strategy"], "")
	}

	if err := serverProductionTM.Start(); err != nil {
		renderError(w, err.Error())
		return
	}

	renderJson(w, H{})
}

func handleActionsStopProduction(w http.ResponseWriter, r *http.Request) {
	serverProductionTMRW.Lock()
	defer serverProductionTMRW.Unlock()

	serverProductionTM.Stop()

	renderJson(w, H{})
}

func renderJson(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		renderError(w, err.Error())
	}
}

func renderError(w http.ResponseWriter, e string) {
	w.WriteHeader(500)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{\"error\": \"" + e + "\"}"))
}
