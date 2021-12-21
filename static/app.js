// Toreda

// {{{ Datatypes
/*
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
	Type             OrderType `json:"type"`
	LimitPrice       float64   `json:"limitPrice"`
	StopPrice        float64   `json:"stopPrice"`
	AvgExecPrice     float64   `json:"avgExecPrice"`
	State            string    `json:"state"`
}
*/
// }}}

// {{{ state
var state = {
  symbol: 'TQQQ',
  day: '2017-07-31',
  fastMA: 12,
  slowMA: 40,
  bollingerMA: 20,
  environmentsOptions: [
    ['paper', 'Paper'],
    ['staging', 'Staging'],
    ['production', 'Production'],
  ],
  strategyOptions: [
    ['rsi', 'RSI'],
    ['long_ma', 'MA Crossover'],
    ['bar_color', 'Bar Color Follow'],
    ['follow_nasdaq', 'Nasdaq Follow'],
    ['gapngo', 'Gap-N-Go'],
  ],
  accountsOptions: [
    ['26924694', 'Algo'],
    ['26914912', 'Margin'],
    ['51727447', 'RRSP'],
  ],
  accounts: {
    accountId: '26924694',
  },
  logs: {
    environment: 'paper',
  },
  runProduction: {
    strategy: 'long_ma',
  },
  runTest: {
    strategy: 'long_ma',
    cash: '1000',
    start: '2017-02-01',
    end: '2017-07-31',
  },
  api: {
    details: {status: 'request', data: null, error: null},
    dayData: {status: 'request', data: null, error: null},
    account: {status: 'request', data: null, error: null},
    logs: {status: 'request', data: null, error: null},
    runTest: {status: 'request', data: null, error: null},
    runProduction: {status: 'request', data: null, error: null},
  },
};
// }}}

// {{{ Alert, Input, Select
var Alert = {
  view: function(vnode) {
    var type = vnode.attrs.type;
    return m(`.bg-lightest-${type}.ba.b--light-${type}.br1.white.pa2`, [
      m(`.f5.${type}`, vnode.attrs.message),
    ]);
  },
};

var Input = {
  view: function(vnode) {
    var stateNode = vnode.attrs.stateNode;
    var currentValue = stateNode[0][stateNode[1]];

    return m('input' + (vnode.attrs.class || ''), {
      type: 'text',
      value: currentValue,
      onchange: function(e) { stateNode[0][stateNode[1]] = e.target.value; },
    });
  },
};

var Select = {
  view: function(vnode) {
    var stateNode = vnode.attrs.stateNode;
    var currentValue = stateNode[0][stateNode[1]];

    return m('select' + (vnode.attrs.class || ''), {
      onchange: function(e) { stateNode[0][stateNode[1]] = e.target.value; },
    }, vnode.attrs.options.map(function (o) {
      return m('option', {
        value: o[0],
        selected: currentValue == o[0]
      }, o[1]);
    }));
  },
};
// }}}

// {{{ DayChart
var DayChart = {
  oninit: function() {
    this.panning = false;
    this.offsetX = 0;
  },
  onMouseDown: function(e) {
    this.panningStartX = e.pageX;
    this.previousOffsetX = this.offsetX;
    this.panning = true;
  },
  onMouseMove: function(e) {
    if (!this.panning) return;
    this.offsetX = this.previousOffsetX + (e.pageX - this.panningStartX);
  },
  onMouseUp: function(e) {
    if (!this.panning) return;
    this.panning = false;
    this.offsetX = this.previousOffsetX + (e.pageX - this.panningStartX);
  },
  view: function(vnode) {
    var width = 1560;
    var height = 700;
    var topMargin = 15;
    var leftMargin = 30;
    var barWidth = 4;
    var totalHeight = height + (topMargin * 2);
    var candles = vnode.attrs.candles;
    var dayCandle = candleForArray(candles);

    var backgroundEls = [];
    var outerEls = [];
    var yLabelsEls = [];
    var candleEls = [];
    var drawingEls = [];

    if (!this.backgroundEls) {
      var dayCandleRange = dayCandle.high - dayCandle.low;
      var priceTick = (Math.round(((dayCandleRange * 100) / 20) / 5) * 5) / 100;
      var startPrice = (Math.floor((dayCandle.high * 100) / (priceTick * 100)) * (priceTick * 100)) / 100;

      // Top and Bottom bars
      backgroundEls.push(m('line.stroke-light-gray', {
        x1: 0, y1: 0, x2: width, y2: 0,
      }));
      backgroundEls.push(m('line.stroke-light-gray', {
        x1: 0, y1: height, x2: width, y2: height,
      }));

      // Left and Right bars
      backgroundEls.push(m('line.stroke-light-gray', {
        x1: 0, y1: 0, x2: 0, y2: height,
      }));
      backgroundEls.push(m('line.stroke-light-gray', {
        x1: width, y1: 0, x2: width, y2: height,
      }));

      // Horizontal Price Lines
      for (var p = startPrice; p > dayCandle.low; p -= priceTick) {
        var y = ((dayCandle.high - p) / dayCandleRange) * height;
        backgroundEls.push(m('line.stroke-light-gray', {
          x1: 0, y1: y, x2: width, y2: y,
        }));
        outerEls.push(m('text', {
          'style': 'user-select: none;',
          'font-size': 10, x: 5, y: y + topMargin + 4
        }, p.toFixed(2)));
      }

      // Vertical Time Lines
      for (var i = 0; i < (6.5 * 60); i += 30) {
        backgroundEls.push(m('line.stroke-light-gray', {
          x1: i * barWidth, y1: 0, x2: i * barWidth, y2: height,
        }));
        backgroundEls.push(m('line.stroke-light-gray', {
          x1: (i + 15) * barWidth, y1: 0, x2: (i + 15) * barWidth, y2: height,
        }));
        yLabelsEls.push(m('text', {
          'style': 'user-select: none;',
          'font-size': 10, x: (i * barWidth) + leftMargin - 12, y: 12
        }, `${Math.floor((i + (9.5 * 60)) / 60)}:${padLeft((i + 30) % 60, 2, '0')}`));
      }

      var candleCloses = candles.map(c => c.close);
      var slowMA = wma(candleCloses, state.slowMA);
      var fastMA = wma(candleCloses, state.fastMA);
      var bollingerBandsValues = bollingerBands(candleCloses, state.bollingerMA);
      var slowMAPoints = [];
      var fastMAPoints = [];
      var bollingerBandsPoints = [];

      for (var i in candles) {
        var candle = candles[i];
        var d = new Date(candle.start);
        var offset = (d.getHours() * 60) + d.getMinutes() - (9.5 * 60);
        var color = candle.close >= candle.open ? 'green' : 'red';
        var coords = boxCoordsForCandle(candle, dayCandle, height);

        if (offset >= 0 && offset <= (6.5 * 60)) {
          candleEls.push(m(`line.stroke-${color}`, {
            x1: (offset * barWidth) + (barWidth * 0.75), y1: coords.y1,
            x2: (offset * barWidth) + (barWidth * 0.75), y2: coords.y2,
          }));
          candleEls.push(m(`rect.fill-${color}`, {
            x: (offset * barWidth) + 1, y: coords.y,
            width: (barWidth - 1), height: coords.height,
          }));
          slowMAPoints.push({
            x: offset * barWidth,
            y: ((dayCandle.high - slowMA[i]) / dayCandleRange) * height,
          });
          fastMAPoints.push({
            x: offset * barWidth,
            y: ((dayCandle.high - fastMA[i]) / dayCandleRange) * height,
          });
          bollingerBandsPoints.push({
            x: offset * barWidth,
            yTop: ((dayCandle.high - bollingerBandsValues[i].top) / dayCandleRange) * height,
            yValue: ((dayCandle.high - bollingerBandsValues[i].value) / dayCandleRange) * height,
            yBottom: ((dayCandle.high - bollingerBandsValues[i].bottom) / dayCandleRange) * height,
          });
        }
      }

      // Draw SMMAs
      drawingEls.push(m('polyline.stroke-purple', {
        'shape-rendering': 'geometricPrecision', fill: 'none',
        points: slowMAPoints.reduce(function (points, p) {
          return points + `${p.x},${p.y} `;
        }, ''),
      }));
      drawingEls.push(m('polyline.stroke-blue', {
        'shape-rendering': 'geometricPrecision', fill: 'none',
        points: fastMAPoints.reduce(function (points, p) {
          return points + `${p.x},${p.y} `;
        }, ''),
      }));
      drawingEls.push(m('polyline.stroke-gold', {
        'shape-rendering': 'geometricPrecision', fill: 'none',
        points: bollingerBandsPoints.reduce(function (points, p) {
          return points + `${p.x},${p.yTop} `;
        }, ''),
      }));
      drawingEls.push(m('polyline.stroke-gold', {
        'shape-rendering': 'geometricPrecision', fill: 'none',
        points: bollingerBandsPoints.reduce(function (points, p) {
          return points + `${p.x},${p.yBottom} `;
        }, ''),
      }));

      this.candleEls = candleEls;
      this.outerEls = outerEls;
      this.yLabelsEls = yLabelsEls;
      this.backgroundEls =  backgroundEls;
      this.drawingEls = drawingEls;
    } else {
      candleEls = this.candleEls;
      outerEls = this.outerEls;
      yLabelsEls = this.yLabelsEls;
      backgroundEls = this.backgroundEls;
      drawingEls = this.drawingEls;
    }

    return m('svg', {
      onmouseup: this.onMouseUp.bind(this),
      onmousedown: this.onMouseDown.bind(this),
      onmousemove: this.onMouseMove.bind(this),
      width: 1022, height: totalHeight, viewbox: `0 0 ${width} ${totalHeight}`,
      xmlns: 'http://www.w3.org/2000/svg', 'shape-rendering': 'crispEdges',
    }, [
      m('g', {}, outerEls),
      m('g', {
        transform: `translate(${this.offsetX}, 0)`
      }, yLabelsEls),
      m('g', {
        transform: `translate(${leftMargin+this.offsetX}, ${topMargin})`
      }, backgroundEls.concat(candleEls).concat(drawingEls)),
    ]);
  },
};
// }}}

// {{{ PageChart
var PageChart = {
  oninit: function() {
    this.fetchDayData();
  },
  fetchDayData: function() {
    state.api.dayData.status = 'request';
    return m.request({
      method: 'GET',
      url: `/data/1m-1d/${state.symbol}/${state.day}`,
    }).then(function(result) {
      state.api.dayData.status = 'success';
      state.api.dayData.data = result;
    }).catch(function(err) {
      state.api.dayData.status = 'failure';
      state.api.dayData.error = err;
    });
  },
  keypress: function(k, e) {
    if (e.key === 'Enter') {
      return this.blur(k, e);
    }
    state[k] = e.target.value;
  },
  blur: function(k, e) {
    var changed = e.target.value != state[k];
    state[k] = e.target.value;
    if (changed) {
      return this.fetchDayData();
    }
  },
  view: function() {
    var chartEl = null;

    if (state.api.dayData.status === 'success') {
      var candles = state.api.dayData.data;
      chartEl = m(DayChart, {candles: candles});
    }

    return m('div', [
      m('div.bg-light-gray.flex.pa2.items-center.bb.b--silver', [
        m('input.hk-input.mr2', {
          type: 'text',
          onblur: this.blur.bind(this, 'symbol'),
          onkeypress: this.keypress.bind(this, 'symbol'),
          placeholder: 'Symbol',
          value: state.symbol,
          size: 6,
        }),
        m('input.hk-input.mr2', {
          type: 'text',
          onblur: this.blur.bind(this, 'day'),
          onkeypress: this.keypress.bind(this, 'day'),
          placeholder: 'Day',
          value: state.day,
          size: 12,
        }),
        (state.api.dayData.status === 'failure') ? (
          m(Alert, {type: 'red', message: 'Error fetching day data'})
        ) : (state.api.dayData.status === 'request') ? (
          m(Alert, {type: 'blue', message: 'Loading...'})
        ) : null,
      ]),
      chartEl,
    ]);
  },
};
// }}}

// {{{ RunStatistics
var RunStatistics = {
  oninit: function(vnode) {
      this.tradesSortBy = 'openedAt';
      this.tradesSortOrder = 'desc';
  },
  sort: function(by) {
    if (this.tradesSortBy === by) {
      this.tradesSortOrder = this.tradesSortOrder === 'asc' ? 'desc' : 'asc';
    } else {
      this.tradesSortBy = by;
      this.tradesSortOrder = 'desc';
    }
  },
  view: function(vnode) {
    var cash = vnode.attrs.cash;
    var data = vnode.attrs.data;

    var statsHeadClass = '.dtc.pv2.f5.bb.b--light-gray.pl2.dark-gray.b';
    var statsCellClass = '.dtc.pv2.f5.bb.b--light-gray.pr2.tr';
    var headClass = '.dtc.pv2.f5.bb.b--light-gray.dark-gray.b';
    var cellClass = '.dtc.pv2.f5.bb.b--light-gray';

    var trades = computeTrades(data.executions);

    var topStatsBoxesEl = m('.flex.flex-row', [
      m('.bg-lightest-silver.shadow-outer-1.br2.flex-auto.w-50.tc.mr2', [
        m('.mv2.b', 'Time'),
        m('.b2.f3', formatDateTime(data.time)),
      ]),
      m('.bg-lightest-silver.shadow-outer-1.br2.flex-auto.w-50.tc.mr2', [
        m('.mv2.b', 'Cash'),
        m('.mb2.f3', data.balance.cash.toFixed(2)),
      ]),
      m('.bg-lightest-silver.shadow-outer-1.br2.flex-auto.w-50.tc', [
        m('.mv2.b', 'Market Value'),
        m('.mb2.f3', data.balance.marketValue.toFixed(2)),
      ]),
    ]);

    var positionsTableEl = m('.dt.w-100', [
      m('.dt-row', [
        m(headClass, 'Sym.'),
        m(headClass+'.tr', 'Open Qty.'),
        m(headClass+'.tr', 'Avg. Entry Price'),
        m(headClass+'.tr', 'Current Price'),
        m(headClass+'.tr', 'Current Value'),
        m(headClass+'.tr', 'Trade Count'),
        m(headClass+'.tr', 'Profit'),
      ]),
      data.positions.length === 0 ? m('.dt-row', m('.dtc.f5.pv2', 'No positions.')) : null,
      data.positions.map(function (p) {
        return m('.dt-row', [
          m(cellClass, p.symbol),
          m(cellClass+'.tr', p.openQuantity),
          m(cellClass+'.tr', p.averageEntryPrice.toFixed(2)),
          m(cellClass+'.tr', p.currentPrice.toFixed(2)),
          m(cellClass+'.tr', p.currentMarketValue.toFixed(2)),
          m(cellClass+'.tr', filter(propEq('symbol', p.symbol), trades).length),
          m(cellClass+'.tr', sum(map(prop('profit'), filter(propEq('symbol', p.symbol), trades))).toFixed(2)),
        ]);
      }),
    ]);

    var openOrders = data.orders.filter(function (o) {
      return o.state !== 'Executed' && o.state !== 'Canceled';
    });
    var ordersTableEl = m('.dt.w-100', [
      m('.dt-row', [
        m(headClass, 'Sym.'),
        m(headClass, 'Created'),
        m(headClass, 'Side'),
        m(headClass, 'Type'),
        m(headClass, 'State'),
        m(headClass+'.tr', 'Total Qty.'),
        m(headClass+'.tr', 'Limit/Stop Price'),
      ]),
      openOrders.length === 0 ? m('.dt-row', m('.dtc.f5.pv2', 'No open orders.')) : null,
      openOrders.map(function (o) {
        return m('.dt-row', [
          m(cellClass, o.symbol),
          m(cellClass, formatDateTime(o.creationTime)),
          m(cellClass, o.side),
          m(cellClass, o.orderType),
          m(cellClass, o.state),
          m(cellClass+'.tr', o.totalQuantity),
          m(cellClass+'.tr', (o.stopPrice || o.limitPrice).toFixed(2)),
        ]);
      }),
    ]);

    var sortedTrades = sortBy(this.tradesSortBy, trades);
    if (this.tradesSortOrder === 'desc') {
      sortedTrades = reverse(sortedTrades);
    }
    var tradesTableEl = m('.dt.w-100', [
      m('.dt-row', [
        m(headClass, {onclick: this.sort.bind(this, 'symbol')}, 'Sym.'),
        m(headClass, {onclick: this.sort.bind(this, 'openedAt')}, 'Opened'),
        m(headClass, {onclick: this.sort.bind(this, 'closeAt')}, 'Closed'),
        m(headClass+'.tr', {onclick: this.sort.bind(this, 'quantity')}, 'Total Qty.'),
        m(headClass+'.tr', {onclick: this.sort.bind(this, 'profit')}, 'P/L $'),
        m(headClass+'.tr', {onclick: this.sort.bind(this, 'profitPercent')}, 'P/L %'),
        m(headClass+'.tr', {onclick: this.sort.bind(this, 'fees')}, 'Fees'),
        m(headClass+'.tr', {onclick: this.sort.bind(this, 'executionsCount')}, 'Exec. #'),
      ]),
      sortedTrades.length === 0 ? m('.dt-row', m('.dtc.f5.pv2', 'No trades.')) : null,
      sortedTrades.map(function (t) {
        var color = t.profit > 0 ? '.dark-green' : '.red';
        return m('.dt-row', [
          m(cellClass, t.symbol),
          m(cellClass, formatDateTime(t.openedAt)),
          m(cellClass, formatDateTime(t.closedAt)),
          m(cellClass+'.tr', t.quantity),
          m(cellClass+'.tr'+color, t.profit.toFixed(2)),
          m(cellClass+'.tr'+color, formatPercent(t.profitPercent)),
          m(cellClass+'.tr', t.fees.toFixed(2)),
          m(cellClass+'.tr', t.executions.length),
        ]);
      }),
    ]);

    var tradesStats = computeTradesStats(cash, data, trades);
    var stats = [
      ['Duration', tradesStats.duration],
      ['Starting $', tradesStats.startingCash.toFixed(2)],
      ['Value $', tradesStats.cash.toFixed(2)],
      ['Total Return $', tradesStats.totalReturn.toFixed(2), true],
      ['Total Fees $', tradesStats.totalFees.toFixed(2)],
      ['Total Return %', formatPercent(tradesStats.totalReturnPercent), true],
      ['CAGR', formatPercent(tradesStats.cagr)],
      ['Sharpe Ratio', tradesStats.sharpe.toFixed(2)],
      ['Sortino Ratio', tradesStats.sortino.toFixed(2)],
      ['Avg Drawdown %', formatPercent(tradesStats.avgDrawdownPercent)],
      ['Max Drawdown %', formatPercent(tradesStats.maxDrawdownPercent)],
      ['Trade Win %', formatPercent(tradesStats.tradeWinPercent)],
      ['Average Trade %', formatPercent(tradesStats.avgTradePercent), true],
      ['Average Win %', formatPercent(tradesStats.avgWinPercent), true],
      ['Average Loss %', formatPercent(tradesStats.avgLossPercent), true],
      ['Best Trade %', formatPercent(tradesStats.bestTradePercent), true],
      ['Worst Trade %', formatPercent(tradesStats.worstTradePercent), true],
      ['Worst Trade Date', 'TBD'],
      ['Avg. Time In Trade', (tradesStats.avgTimeInTrade/(1000*60)).toFixed(1)+'m'],
      ['Trade Count', tradesStats.tradeCount],
      ['Winning Months %', '-'],
      ['Average Month %', '-'],
      ['Average Winning Month %', '-'],
      ['Average Loosing Month %', '-'],
      ['Best Month %', '-'],
      ['Worst Month %', '-'],
    ];

    return m('.flex', [
      // Contents
      m('.flex-auto.w-75.pa2.br.b--silver', [
        topStatsBoxesEl,
        m('img.mt3', {
          src: [
            'http://chartd.co/a.svg?hl=1&w=1022&h=250',
            '&d0=', encodeChartDataset(tradesStats.profitAmounts),
            '&ymin=', Math.floor(min(tradesStats.profitAmounts)),
            '&ymax=', Math.ceil(max(tradesStats.profitAmounts)),
          ].join('')
        }),
        m('h3.fw3.mt3.mb0', 'Positions'),
        positionsTableEl,
        m('h3.fw3.mt3.mb0', 'Open Orders'),
        ordersTableEl,
        m('h3.fw3.mt3.mb0', 'Trades'),
        tradesTableEl,
      ]),
      // Sidebar
      m('.flex-auto.w-25', [
        m('h3.fw3.pl2.mv2', 'Statistics'),
        m('.dt.w-100.bt.b--light-gray', stats.map(function(s) {
          return m('.dt-row', [
            m(statsHeadClass, s[0]),
            m(statsCellClass + (s[2] ? (parseFloat(s[1]) >= 0 ? '.dark-green' : '.red') : ''), s[1]),
          ]);
        })),
      ]),
    ]);
  }
};
// }}}

// {{{ PageRunPaper
var PageRunPaper = {
  oninit: function(vnode) {
    this.refresh();
  },
  refresh: function() {
    simpleRequest(state.api.runTest, '/data/run/paper');
  },
  refreshLoop: function() {
    this.intervalHandle = setInterval(function() {
      this.refresh();
    }.bind(this), 1000);
  },
  run: function() {
    this.refreshLoop();
    m.request({
      method: 'POST',
      url: '/actions/run/paper',
      data: state.runTest,
    }).catch(function (err) {
      console.error(err);
    }).then(function() {
      clearInterval(this.intervalHandle);
      this.intervalHandle = null;
      this.refresh();
    }.bind(this));
  },
  onremove: function(vnode) {
    if (this.intervalHandle) {
      clearInterval(this.intervalHandle);
    }
  },
  view: function() {
    var data = state.api.runTest.data;
    var cash = parseFloat(state.runTest.cash);

    if (!data) {
      return m('.tc.pa4', 'Loading...');
    }

    var status = state.api.runTest.status;
    if (status == 'success') {
      status = data.state;
    }
    var statusClass = backgroundForStatus(status);


    return m('div', [
      // Sub-header
      m('div.bg-light-gray.flex.pa2.items-center.bb.b--silver.relative', [
        m(Select, {
          class: '.hk-input.mr2',
          stateNode: [state.runTest, 'strategy'],
          options: state.strategyOptions,
        }),
        m(Input, {
          class: '.hk-input.w5.mr2',
          stateNode: [state.runTest, 'cash'],
        }),
        m(Input, {
          class: '.hk-input.w5.mr2',
          stateNode: [state.runTest, 'start'],
        }),
        m(Input, {
          class: '.hk-input.w5.mr2',
          stateNode: [state.runTest, 'end'],
        }),
        m('button.hk-button--primary.mr2', {
          onclick: this.run.bind(this),
        }, 'Run'),
        m('button.hk-button--secondary.mr2', {
          onclick: this.refresh.bind(this),
        }, 'Refresh'),
        m('.hk-badge.absolute.right-1'+statusClass, status),
      ]),
      m(RunStatistics, {data: data, cash: cash}),
    ]);
  },
};
// }}}

// {{{ PageRunStaging
var PageRunStaging = {
  view: function() {
    return m('div', [
    ]);
  },
};
// }}}

// {{{ PageRunProduction
var PageRunProduction = {
  oninit: function(vnode) {
    this.refreshLoop();
  },
  onremove: function(vnode) {
    if (this.intervalHandle) {
      clearInterval(this.intervalHandle);
    }
  },
  refreshLoop: function() {
    this.intervalHandle = setInterval(function() {
      this.refresh();
    }.bind(this), 5000);
  },
  refresh: function() {
    simpleRequest(state.api.runProduction, '/data/run/production');
  },
  start: function() {
    m.request({
      method: 'POST',
      url: '/actions/start/production',
      data: state.runProduction,
    }).catch(function (err) {
      console.error(err);
      alert(err);
    });
  },
  stop: function() {
    m.request({
      method: 'POST',
      url: '/actions/stop/production',
      data: state.runProduction,
    }).catch(function (err) {
      console.error(err);
      alert(err);
    });
  },
  view: function() {
    var data = state.api.runProduction.data;

    if (!data) {
      return m('.tc.pa4', 'Loading...');
    }

    var status = state.api.runProduction.status;
    if (status == 'success') {
      status = data.state;
    }
    var statusClass = backgroundForStatus(status);

    var cash = data.balance.startOfDayCash;

    return m('div', [
      // Sub-header
      m('div.bg-light-gray.flex.pa2.items-center.bb.b--silver.relative', [
        m(Select, {
          class: '.hk-input.mr2',
          stateNode: [state.runProduction, 'strategy'],
          options: state.strategyOptions,
        }),
        m('button.hk-button--primary.mr2', {
          onclick: this.start.bind(this),
        }, 'Start'),
        m('button.hk-button--danger.mr2', {
          onclick: this.stop.bind(this),
        }, 'Stop'),
        m('button.hk-button--secondary.mr2', {
          onclick: this.refresh.bind(this),
        }, 'Refresh'),
        m('.hk-badge.absolute.right-1'+statusClass, status),
      ]),
      m(RunStatistics, {data: data, cash: cash}),
    ]);
  },
};
// }}}

// {{{ PageScan
var PageScan = {
  oninit: function() {
  },
  view: function() {
    var cellClass = '.dtc.pv2.f5.bb.b--light-gray';
    var headerCellClass = '.dtc.pv2.b.bb.b--light-gray.dark-gray.f5';
    var scanResults = state.api.details.data;

    var sortOrder = localStorage.scannerSortOrder || 'desc';
    var sortBy = localStorage.scannerSortBy || 'marketCap';
    scanResults.sort(function(a, b) {
      var s = a[sortBy] < b[sortBy] ? -1 : 1;
      return sortOrder === 'asc' ? s : 0-s;
    });
    scanResults = scanResults.slice(0, 100);

    return m('div', [
      m('div.bg-light-gray.flex.pa2.items-center.bb.b--silver', [
        m(Select, {
          class: '.hk-input.mr2',
          stateNode: [localStorage, 'scannerSortBy'],
          options: [
            ['symbol', 'Symbol'],
            ['description', 'Description'],
            ['prevDayClosePrice', 'Price'],
            ['lowPrice52', '52W Low'],
            ['highPrice52', '52W High'],
            ['averageVol20Days', '20D Average Volume'],
            ['marketCap', 'Market Cap'],
            ['outstandingShares', 'Outstanding Shares'],
            ['pe', 'Profit/Earning Ratio'],
            ['eps', 'Earning Per Share'],
            ['industrySector', 'Sector'],
            ['industryGroup', 'Group'],
          ],
        }),
        m(Select, {
          class: '.hk-input.mr2',
          stateNode: [localStorage, 'scannerSortOrder'],
          options: [['asc', 'Asc.'], ['desc', 'Desc.']],
        }),
      ]),
      m('.dt.w-100', [
        m('.dt-row', [
          m(headerCellClass+'.pl2', 'Sym.'),
          m(headerCellClass, 'Desc.'),
          m(headerCellClass+'.tr', 'Price'),
          m(headerCellClass+'.tr', '52W Low'),
          m(headerCellClass+'.tr', '52W High'),
          m(headerCellClass+'.tr', '20D Avg. Vol.'),
          m(headerCellClass+'.tr', 'Mrkt. Cap'),
          m(headerCellClass+'.tr', 'Out. Shares'),
          m(headerCellClass+'.tr', 'P/E'),
          m(headerCellClass+'.tr', 'EPS'),
          m(headerCellClass+'.pl2', 'Sector'),
          m(headerCellClass, 'Group'),
        ]),
        scanResults.map(function (s) {
          return m('.dt-row', {id: 'sym_'+s.symbolId}, [
            m(cellClass+'.pl2', s.symbol),
            m(cellClass, s.description.slice(0,12)),
            m(cellClass+'.tr', s.prevDayClosePrice.toFixed(2)),
            m(cellClass+'.tr', s.lowPrice52.toFixed(2)),
            m(cellClass+'.tr', s.highPrice52.toFixed(2)),
            m(cellClass+'.tr', formatLargeNum(s.averageVol20Days)),
            m(cellClass+'.tr', formatLargeNum(s.marketCap)),
            m(cellClass+'.tr', formatLargeNum(s.outstandingShares)),
            m(cellClass+'.tr', s.pe.toFixed(2)),
            m(cellClass+'.tr', s.eps.toFixed(2)),
            m(cellClass+'.pl2', s.industrySector.slice(0,12)),
            m(cellClass, s.industryGroup.slice(0,12)),
          ]);
        }),
      ])
    ]);
  },
};
// }}}

// {{{ PageResearch
var PageResearch = {
  view: function() {
    return m('div', [
      m('h2', 'Research'),
    ]);
  },
};
// }}}

// {{{ PageAccounts
var PageAccounts = {
  oninit: function() {
    if (!state.api.account.data) {
      this.fetch();
    }
  },
  fetch: function() {
    simpleRequest(
      state.api.account,
      '/data/account?accountId=' + state.accounts.accountId
    );
  },
  renderContents: function() {
    return m('.pa3', [
      m('pre.f6', JSON.stringify(state.api.account.data, null, 2)),
    ]);
  },
  view: function() {
    var status = state.api.account.status;

    var contentsEl = m('.tc.pa3', 'Loading...');
    if (state.api.account.data) {
      contentsEl = this.renderContents();
    }

    return m('div', [
      m('div.bg-light-gray.flex.pa2.items-center.bb.b--silver.relative', [
        m(Select, {
          class: '.hk-input.mr2',
          stateNode: [state.accounts, 'accountId'],
          options: state.accountsOptions,
        }),
        m('button.hk-button--primary.mr2', {
          onclick: this.fetch.bind(this),
        }, 'Fetch'),
        m('.hk-badge.absolute.right-1'+backgroundForStatus(status), status),
      ]),
      contentsEl,
    ]);
  },
};
// }}}

// {{{ PageLogs
var PageLogs = {
  fetch: function() {
    m.request('/data/logs?environment=' + state.logs.environment, {
      deserialize: identity,
    }).catch(function(err) {
      state.api.logs.status = 'failure';
      state.api.logs.error = err;
    }).then(function(response) {
      state.api.logs.status = 'success';
      state.api.logs.data = response;
    });
  },
  view: function() {
    var status = state.api.logs.status;
    return m('div', [
      m('div.bg-light-gray.flex.pa2.items-center.bb.b--silver.relative', [
        m(Select, {
          class: '.hk-input.mr2',
          stateNode: [state.logs, 'environment'],
          options: state.environmentsOptions,
        }),
        m('button.hk-button--primary.mr2', {
          onclick: this.fetch.bind(this),
        }, 'Show'),
        m('.hk-badge.absolute.right-1'+backgroundForStatus(status), status),
      ]),
      m('pre.pa3.f5', state.api.logs.data || 'Nothing to show.'),
    ]);
  },
};
// }}}

// {{{ App
var App = {
  oninit: function() {
    if (!state.api.details.data) {
      return simpleRequest(state.api.details, '/data/details');
    }
  },
  changePage: function(page) {
    localStorage.page = page;
  },
  view: function() {
    var page = localStorage.page || 'scan';
    var pageEl;

    var pages = [
      {key: 'run-paper', name: 'Run Paper', component: PageRunPaper},
      {key: 'run-staging', name: 'Run Staging', component: PageRunStaging},
      {key: 'run-production', name: 'Run Poduction', component: PageRunProduction},
      {key: 'chart', name: 'Chart', component: PageChart},
      {key: 'scan', name: 'Scan', component: PageScan},
      {key: 'research', name: 'Research', component: PageResearch},
      {key: 'accounts', name: 'Accounts', component: PageAccounts},
      {key: 'logs', name: 'Logs', component: PageLogs},
    ];

    if (state.api.details.status === 'request') {
      pageEl = m('.tc.pa3', 'Loading...');
    } else if (state.api.details.status === 'request') {
      pageEl = m(Alert, {message: state.api.details.error});
    } else {
      for (var i in pages) {
        if (pages[i].key === page) {
          pageEl = m(pages[i].component);
        }
      }
    }

    return m('div.h-100.w-100.mw8.bl.br.bb.b--silver.bg-white', [
      m('h1.ma0.ph1.pv2.fw3.f2.ml2', 'Toreda'),
      m('div.hk-tabs.bb.b--silver', pages.map(function (p) {
        return m('a', {
          class: 'hk-tabs__tab' + (page === p.key ? '--active' : ''),
          onclick: this.changePage.bind(this, p.key),
        }, p.name);
      }.bind(this))),
      pageEl,
    ]);
  },
};
// }}}

m.mount(document.getElementById('appRoot'), App);

// {{{ helpers
function computeTrades(executions) {
  var trades = [];
  var openTrades = {};
  for (var i in executions) {
    var execution = executions[i];
    if (execution.symbol in openTrades) {
      // Append execution to current open trade
      var t = openTrades[execution.symbol];
      t.executions.push(execution);
      if (execution.side === "Buy") {
        t.quantity += execution.quantity;
        t.highestQuanity = Math.max(t.highestQuanity, t.quantity);
      } else {
        t.quantity -= execution.quantity;
      }

      // Close trade if qty is back down to 0
      if (t.quantity === 0) {
        t.closedAt = new Date(execution.timestamp);
        t.quantity = t.highestQuanity;
        t.value = t.highestQuanity * avg(map(prop('price'), t.executions));
        t.profitBeforeFees = t.executions.reduce(function (t, e) {
          return t + (e.quantity * e.price * (e.side === "Buy" ? -1 : 1));
        }, 0);
        t.fees = sum(map(prop('commission'), t.executions)) +
          sum(map(prop('secFee'), t.executions)) +
          sum(map(prop('orderPlacementCommission'), t.executions)) +
          sum(map(prop('executionFee'), t.executions));
        t.profit = t.profitBeforeFees - t.fees;
        t.profitPercent = t.profit / t.value;
        t.executionsCount = t.executions.length;
        trades.push(t);
        delete openTrades[execution.symbol];
      }
    } else {
      // Start a new trade
      openTrades[execution.symbol] = {
        symbol: execution.symbol,
        openedAt: new Date(execution.timestamp),
        quantity: execution.quantity,
        highestQuanity: execution.quantity,
        executions: [execution],
      };
    }
  }
  return trades;
}

function computeTradesStats(startingCash, data, trades) {
  var stats = {};

  var firstTradeDate = min(map(prop('openedAt'), trades));
  var lastTradeDate = max(map(prop('closedAt'), trades));
  if (trades.length === 0) {
    firstTradeDate = new Date();
    lastTradeDate = new Date();
  }
  var delta = lastTradeDate.getTime() - firstTradeDate.getTime();

  var profit = 0;
  var profitHigh = 0 - Number.MAX_SAFE_INTEGER;
  var profitAmounts = [];
  var drawdowns = [];
  var wins = [];
  var winsPercents = [];
  var losses = [];
  var lossesPercents = [];
  var timesInTrade = [];
  for (var i in trades) {
    var trade = trades[i];
    profit += trade.profit;
    profitHigh = Math.max(profitHigh, profit);
    profitAmounts.push(profit);
    timesInTrade.push(trade.closedAt.getTime() - trade.openedAt.getTime());
    drawdowns.push(1 - (profit / profitHigh));
    if (trade.profit >= 0) {
      wins.push(trade.profit);
      winsPercents.push(trade.profitPercent);
    } else {
      losses.push(trade.profit);
      lossesPercents.push(trade.profitPercent);
    }
  }

  stats.profitAmounts = profitAmounts;
  stats.duration = formatDateDuration(firstTradeDate, lastTradeDate);
  stats.startingCash = startingCash;
  stats.cash = data.balance.cash + data.balance.marketValue;
  stats.totalReturn = stats.cash - stats.startingCash;
  stats.totalFees = sum(map(prop('fees'), trades));
  stats.totalReturnPercent = (stats.cash / startingCash) - 1;
  stats.cagr = (stats.totalReturnPercent / (delta / (3600000 * 24))) * 365;
  stats.sharpe = 0; // (mean return % - risk free %) / return std dev
  stats.sortino = 0; // (mean return % - risk free %) / downside returns std dev
  stats.avgDrawdownPercent = avg(drawdowns);
  stats.maxDrawdownPercent = max(drawdowns);
  stats.tradeWinPercent = wins.length / trades.length;
  stats.avgTradePercent = avg(winsPercents.concat(lossesPercents));
  stats.avgWinPercent = avg(winsPercents);
  stats.avgLossPercent = avg(lossesPercents);
  stats.bestTradePercent = max(winsPercents.concat(lossesPercents));
  stats.worstTradePercent = min(winsPercents.concat(lossesPercents));
  stats.avgTimeInTrade = avg(timesInTrade);
  stats.tradeCount = trades.length;

  return stats;
}

function simpleRequest(stateNode, url) {
  stateNode.status = 'request';
  return m.request({
    method: 'GET', url: url,
  }).then(function(result) {
    stateNode.status = 'success';
    stateNode.data = result;
  }).catch(function(err) {
    stateNode.status = 'failure';
    stateNode.error = err;
  });
}

function backgroundForStatus(status) {
  var statusClass = '';
  switch (status) {
    case 'failure':
    case 'failing':
    case 'failed':
      statusClass = '.bg-red';
      break;
    case 'stopping':
    case 'stopped':
      statusClass = '.bg-orange';
      break;
    case 'running':
      statusClass = '.bg-green';
      break;
    case 'starting':
    case 'done':
      statusClass = '.bg-blue';
      break;
  }
  return statusClass;
}
// }}}

// {{{ format utils
function formatPercent(value) {
  return (value * 100).toFixed(2) + '%';
}

function formatLargeNum(num) {
  if (num < 1000) {
    return num.toString();
  } else if (num < 1000000) {
    return (num/1000).toFixed(1)+'K';
  } else if (num < 1000000000) {
    return (num/1000000).toFixed(1)+'M';
  } else {
    return (num/1000000000).toFixed(1)+'B';
  }
}

function formatDateTime(dateString) {
  var d = new Date(dateString);
  var date = [
    d.getFullYear(),
    padLeft(d.getMonth()+1, 2, '0'),
    padLeft(d.getDate(), 2, '0')
  ].join('-');
  var time = [
    padLeft(d.getHours(), 2, '0'),
    padLeft(d.getMinutes(), 2, '0')
  ].join(':');
  return date + ' ' + time;
}

function formatDateDuration(dateA, dateB) {
  var delta = Math.abs(dateA.getTime() - dateB.getTime());
  var dayMillis = 1000 * 60 * 60 * 24;
  var monthMillis = dayMillis * 30;
  var yearMillis = dayMillis * 365;
  var years = Math.floor(delta / yearMillis);
  var months = Math.floor((delta - (years * yearMillis)) / monthMillis);
  var days = Math.floor((delta - (years * yearMillis) - (months * monthMillis)) / dayMillis);
  return years + 'y ' + months + 'm ' + days + 'd';
}
// }}}

// {{{ chart utils
function candleForArray(candles) {
  if (candles.length === 0) {
    return {start: null, end: null, open: 0, high: 0, low: 0, close: 0};
  }
  candles = candles.filter(function(c) {
    var date = new Date(c.start);
    var time = (date.getHours() * 60) + date.getMinutes();
    return time >= (9.5 * 60) && time < (16 * 60);
  });
  var start = candles[0].start;
  var end = candles[candles.length-1].end;
  var open = candles[0].open;
  var close = candles[candles.length-1].close;
  var high = 0;
  var low = Number.MAX_SAFE_INTEGER;

  candles.forEach(function(candle) {
    high = Math.max(high, candle.high);
    low = Math.min(low, candle.low);
  });

  return {start: start, end: end, open: open, high: high, low: low, close: close};
}

function boxCoordsForCandle(candle, dayCandle, chartHeight) {
  var dayCandleRange = dayCandle.high - dayCandle.low;
  var candleOCHigh = Math.max(candle.open, candle.close);
  var candleOCLow = Math.min(candle.open, candle.close);
  var y1 = ((dayCandle.high - candle.high) / dayCandleRange) * chartHeight;
  var y2 = ((dayCandle.high - candle.low) / dayCandleRange) * chartHeight;
  var y = ((dayCandle.high - candleOCHigh) / dayCandleRange) * chartHeight;
  var height = Math.max(1, ((candleOCHigh - candleOCLow) / dayCandleRange) * chartHeight);
  return {y1: y1, y2: y2, y: y, height: height};
}

function wmaValue(values) {
  var denominator = (values.length * (values.length + 1)) / 2;
  var total = 0;
  for (var i in values) {
    total += values[i] * ((parseInt(i)+1) / denominator);
  }
  return total;
}

function wma(closePrices, n) {
  var periodValues = [];
  var values = [];
  for (var i in closePrices) {
    var close = closePrices[i];
    periodValues.push(close);
    periodValues = periodValues.slice(Math.max(0, periodValues.length - n));
    values.push(wmaValue(periodValues));
  }
  return values;
}

function bollingerBandsValue(values) {
  var closeAvg = avg(values);
  var deviationsTotal = 0;
  for (var i in values) {
    deviationsTotal += (values[i] - closeAvg) ** 2;
  }
  var stdDeviation = deviationsTotal / values.length;
  return {
    top: closeAvg + (2 * stdDeviation),
    value: closeAvg,
    bottom: closeAvg + (-2 * stdDeviation),
  };
}

function bollingerBands(closePrices, n) {
  var periodValues = [];
  var values = [];
  for (var i in closePrices) {
    var close = closePrices[i];
    periodValues.push(close);
    periodValues = periodValues.slice(Math.max(0, periodValues.length - n));
    values.push(bollingerBandsValue(periodValues));
  }
  return values;
}
// }}}

// {{{ utils
var b62 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

function encodeChartDataset(data, minV, maxV) {
  minV = Math.floor(minV || min(data));
  maxV = Math.ceil(maxV || max(data));
  var r = max([maxV, minV, 0]);
  var bs = [];
  var enclen = b62.length - 1;
  for (var i in data) {
    var y = data[i];
    var index = Math.floor(enclen * (y - minV) / r)
    if (index >= 0) {
      bs.push(b62[index]);
    } else if (index < b62.length) {
      bs.push(b62[b62.length-1]);
    } else {
      bs.push(b62[0]);
    }
  }
  return bs.join('');
}

function padLeft(num, len, ch) {
  var numText = num.toString();
  for (var i = numText.length; i < len; i++) {
    numText = ch + numText;
  }
  return numText;
}

function sum(values) {
  return values.reduce(function (acc, v) {
    return acc + v;
  }, 0);
}

function avg(values) {
  return sum(values) / values.length;
}

function debounce(milliseconds, fn) {
  var timeoutHandle;

  return function() {
    var self = this;
    var args = [].slice.call(arguments);
    // Cancel previous call
    if (timeoutHandle) {
      clearTimeout(timeoutHandle);
    }
    // Schedule calling fn in n milliseconds
    timeoutHandle = setTimeout(() => fn.apply(self, args), milliseconds);
  };
}

function identity(a) {
  return a;
}

function map(fn, v) {
  return v.map(fn);
}

function filter(fn, v) {
  return v.filter(fn);
}

function sortBy(key, v) {
  var vCopy = v.map(identity);
  vCopy.sort(function(a, b) {
    if (a[key] < b[key]) return -1;
    if (a[key] > b[key]) return 1;
    return 0;
  });
  return vCopy;
}

function reverse(v) {
  return v.reverse();
}

function prop(p) {
  return function(o) {
    return o[p];
  };
}

function propEq(prop, eqValue) {
  return function(obj) {
    return obj[prop] === eqValue;
  };
}

function max(values) {
  var high = 0;
  values.forEach(function (v) { if (v > high) { high = v; } });
  return high;
}

function min(values) {
  var low = Number.MAX_SAFE_INTEGER;
  values.forEach(function (v) { if (v < low) { low = v; } });
  return low;
}
// }}}
