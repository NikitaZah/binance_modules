package binance_modules

import (
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
)

type Strategy interface {
	Initialize()
}

type BaseStrategyInterface interface {
	SetClient(client *futures.Client)
	SetSymbol(symbol *futures.Symbol)
	Activate()
}

type BaseStrategy struct {
	Client *futures.Client
	Symbol *futures.Symbol
	On     bool
}

func (BS *BaseStrategy) SetClient(client *futures.Client) {
	BS.Client = client
}

func (BS *BaseStrategy) SetSymbol(symbol *futures.Symbol) {
	BS.Symbol = symbol
}

func (BS *BaseStrategy) Activate() {
	BS.On = true
}

type AccountStrategyInterface interface {
	InitAccount() error
	accountUpdateHandler(event *futures.WsUserDataEvent)
	accountErrorHandler(err error)
	OnAccountUpdate()
}

type AccountStrategy struct {
	AccountStrategyInterface
	BaseStrategy
	Account      *Status
	accountDoneC chan struct{}
	accountStopC chan struct{}
}

func (AS *AccountStrategy) InitAccount() error {
	var err error
	AS.Account, err = NewStatus(AS.Client, AS.Symbol.Symbol)
	if err != nil {
		return err
	}

	listenKey, err := GetListenKey(AS.Client)
	if err != nil {
		return err
	}
	doneC, stopC, err := futures.WsUserDataServe(listenKey.listenKey, AS.accountUpdateHandler, AS.accountErrorHandler)
	if err != nil {
		return err
	}
	AS.accountDoneC = doneC
	AS.accountStopC = stopC
	return nil
}

func (AS *AccountStrategy) accountUpdateHandler(event *futures.WsUserDataEvent) {
	if event.Event == futures.UserDataEventTypeAccountUpdate {
		AS.Account.AccountUpdate(&event.AccountUpdate)
	}
	if event.Event == futures.UserDataEventTypeOrderTradeUpdate {
		AS.Account.OrderUpdate(&event.OrderTradeUpdate)
	}
	if event.Event == futures.UserDataEventTypeListenKeyExpired {
		AS.InitAccount()
	}
	if AS.On {
		AS.OnAccountUpdate()
	}
}

func (AS *AccountStrategy) accountErrorHandler(err error) {
	<-AS.accountDoneC
	AS.InitAccount()
}

type OrderBookStrategyInterface interface {
	InitOrderBook() error
	depthUpdateHandler(event *futures.WsDepthEvent)
	depthErrorHandler(err error)
	OnDepthUpdate()
}

type OrderBookStrategy struct {
	BaseStrategy
	OrderBookStrategyInterface
	Orderbook      *OrderBook
	orderbookStopC chan struct{}
	orderbookDoneC chan struct{}
	conn           chan *futures.WsDepthEvent
}

func (OS *OrderBookStrategy) InitOrderBook() error {
	OS.conn = make(chan *futures.WsDepthEvent, 10)
	doneC, stopC, err := futures.WsDiffDepthServe(OS.Symbol.Symbol, OS.depthUpdateHandler, OS.depthErrorHandler)
	if err != nil {
		return err
	}
	OS.Orderbook, err = NewOrderBook(OS.Client, OS.Symbol.Symbol, 1000)
	if err != nil {
		<-OS.orderbookStopC
		<-OS.orderbookDoneC
		for len(OS.conn) > 0 {
			<-OS.conn
		}
		OS.Orderbook = nil
		return err
	}
	OS.orderbookDoneC = doneC
	OS.orderbookStopC = stopC
	return nil
}

func (OS *OrderBookStrategy) depthUpdateHandler(event *futures.WsDepthEvent) {
	var (
		update  *futures.WsDepthEvent
		updated int
	)

	OS.conn <- event
	if OS.Orderbook != nil {
		for update = range OS.conn {
			updated = OS.Orderbook.Update(update)
		}
		if updated < -1 {
			OS.Orderbook = nil
			<-OS.orderbookStopC
			<-OS.orderbookDoneC
			OS.InitOrderBook()
		} else {
			if updated == 0 {
				if OS.On {
					OS.OnDepthUpdate()
				}
			}
		}
	}
}

func (OS *OrderBookStrategy) depthErrorHandler(err error) {
	for len(OS.conn) > 0 {
		<-OS.conn
	}
	OS.Orderbook = nil
	<-OS.orderbookStopC
	<-OS.orderbookDoneC
	OS.InitOrderBook()
}

type ClusterStrategyInterface interface {
	InitClusters() error
	tradeUpdateHandler(event *futures.WsAggTradeEvent)
	tradeErrorHandler(err error)
	OnCLusterUpdate()
}

type ClusterStrategy struct {
	BaseStrategy
	ClusterStrategyInterface
	Clusters     *Clusters
	clusterStopC chan struct{}
	clusterDoneC chan struct{}
	TimeFrame    int
}

func (CS *ClusterStrategy) InitClusters() error {
	CS.Clusters = NewClusters(CS.TimeFrame)
	doneC, stopC, err := futures.WsAggTradeServe(CS.Symbol.Symbol, CS.tradeUpdateHandler, CS.tradeErrorHandler)
	if err != nil {
		return err
	}
	CS.clusterDoneC = doneC
	CS.clusterStopC = stopC
	return nil
}

func (CS *ClusterStrategy) tradeUpdateHandler(event *futures.WsAggTradeEvent) {
	CS.Clusters.Update(event)
	if CS.On {
		CS.OnCLusterUpdate()
	}
}

func (CS *ClusterStrategy) tradeErrorHandler(err error) {
	<-CS.clusterStopC
	<-CS.clusterDoneC
	CS.InitClusters()
}

type CandlesStrategyInterface interface {
	InitCandles() error
	candleUpdateHandler(event *futures.WsKlineEvent)
	candleErrorHandler(err error)
	OnCandleUpdate()
}

type CandlesStrategy struct {
	BaseStrategy
	CandlesStrategyInterface
	Candles        *Candles
	InitCandlesNum int
	candlesStopC   chan struct{}
	candlesDoneC   chan struct{}
	TimeFrame      int
}

func (CS *CandlesStrategy) InitCandles() error {
	candles, err := NewCandles(CS.Client, CS.Symbol.Symbol, strconv.Itoa(CS.TimeFrame)+"m", CS.InitCandlesNum)
	if err != nil {
		return err
	}
	CS.Candles = candles

	doneC, stopC, err := futures.WsKlineServe(CS.Symbol.Symbol, strconv.Itoa(CS.TimeFrame)+"m", CS.candleUpdateHandler, CS.candleErrorHandler)
	if err != nil {
		return err
	}
	CS.candlesDoneC = doneC
	CS.candlesStopC = stopC
	return nil
}

func (CS *CandlesStrategy) candleUpdateHandler(event *futures.WsKlineEvent) {
	CS.Candles.Update(&event.Kline)
	if CS.On {
		CS.OnCandleUpdate()
	}
}

func (CS *CandlesStrategy) candleErrorHandler(err error) {
	<-CS.candlesStopC
	<-CS.candlesDoneC
	CS.Candles = nil
	CS.InitCandles()
}
