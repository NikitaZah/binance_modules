package binance_modules

import (
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/sirupsen/logrus"
)

type Strategy interface {
	Initialize()
}

type BaseStrategyInterface interface {
	SetClient(client *futures.Client)
	SetSymbol(symbol *futures.Symbol)
	SetLogger() error
	Activate()
}

type AccountStrategyInterface interface {
	InitAccount() error
	accountUpdateHandler(event *futures.WsUserDataEvent)
	accountErrorHandler(err error)
	OnAccountUpdate()
}

type OrderBookStrategyInterface interface {
	InitOrderBook() error
	depthUpdateHandler(event *futures.WsDepthEvent)
	depthErrorHandler(err error)
	OnDepthUpdate()
}

type ClusterStrategyInterface interface {
	InitClusters() error
	tradeUpdateHandler(event *futures.WsAggTradeEvent)
	tradeErrorHandler(err error)
	OnCLusterUpdate()
}

type CandlesStrategyInterface interface {
	InitCandles() error
	candleUpdateHandler(event *futures.WsKlineEvent)
	candleErrorHandler(err error)
	OnCandleUpdate()
}

type BaseStrategy struct {
	Client *futures.Client
	Symbol *futures.Symbol
	On     bool
	log    *Logger
}

type AccountStrategy struct {
	AccountStrategyInterface
	Account      *Status
	accountDoneC chan struct{}
	accountStopC chan struct{}
}

type OrderBookStrategy struct {
	OrderBookStrategyInterface
	Orderbook      *OrderBook
	orderbookStopC chan struct{}
	orderbookDoneC chan struct{}
	conn           chan *futures.WsDepthEvent
}

type ClusterStrategy struct {
	ClusterStrategyInterface
	Clusters     *Clusters
	clusterStopC chan struct{}
	clusterDoneC chan struct{}
	TimeFrame    int
}

type CandlesStrategy struct {
	CandlesStrategyInterface
	Candles        *Candles
	InitCandlesNum int
	candlesStopC   chan struct{}
	candlesDoneC   chan struct{}
	TimeFrame      int
}

type AbstractStrategy struct {
	BaseStrategy
	AccountStrategy
	OrderBookStrategy
	ClusterStrategy
	CandlesStrategy
}

func (AS *AbstractStrategy) SetClient(client *futures.Client) {
	AS.Client = client
}

func (AS *AbstractStrategy) SetSymbol(symbol *futures.Symbol) {
	AS.Symbol = symbol
}

func (AS *AbstractStrategy) SetLogger() error {
	lg, err := GetLogger()
	if err != nil {
		return err
	}
	AS.log = lg
	return nil
}

func (AS *AbstractStrategy) Activate() {
	AS.On = true
}

func (AS *AbstractStrategy) InitAccount() error {
	var err error
	AS.Account, err = NewStatus(AS.Client, AS.Symbol.Symbol)
	if err != nil {
		AS.log.WithFields(logrus.Fields{
			"symbol": AS.Symbol.Symbol,
			"err":    err.Error(),
		}).Error("Failed to initialize account")
		return err
	}

	listenKey, err := GetListenKey(AS.Client)
	if err != nil {
		AS.log.WithFields(logrus.Fields{
			"symbol": AS.Symbol.Symbol,
			"err":    err.Error(),
		}).Error("Failed to get Listen Key")
		return err
	}
	doneC, stopC, err := futures.WsUserDataServe(listenKey.listenKey, AS.accountUpdateHandler, AS.accountErrorHandler)
	if err != nil {
		AS.log.WithFields(logrus.Fields{
			"symbol": AS.Symbol.Symbol,
			"err":    err.Error(),
		}).Error("Failed to initialize User Data Stream")
		return err
	}
	AS.accountDoneC = doneC
	AS.accountStopC = stopC
	AS.log.WithFields(logrus.Fields{
		"symbol": AS.Symbol.Symbol,
	}).Info("Successfully initialized account")
	return nil
}

func (AS *AbstractStrategy) accountUpdateHandler(event *futures.WsUserDataEvent) {
	AS.log.WithFields(logrus.Fields{
		"symbol": AS.Symbol.Symbol,
		"Event":  event.Event,
	}).Info("User Data Event recieved")

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

func (AS *AbstractStrategy) accountErrorHandler(err error) {
	AS.log.WithFields(logrus.Fields{
		"symbol": AS.Symbol.Symbol,
		"err":    err.Error(),
	}).Error("User Data Stream error. Restarting...")

	<-AS.accountDoneC
	err = AS.InitAccount()
	if err != nil {
		AS.log.WithFields(logrus.Fields{
			"symbol": AS.Symbol.Symbol,
			"err":    err.Error(),
		}).Fatal("Failed to restart Account Updates")
	}
}

func (AS *AbstractStrategy) InitOrderBook() error {
	AS.conn = make(chan *futures.WsDepthEvent, 10)
	doneC, stopC, err := futures.WsDiffDepthServeWithRate(AS.Symbol.Symbol, 100*time.Millisecond, AS.depthUpdateHandler, AS.depthErrorHandler)
	if err != nil {
		AS.log.WithFields(logrus.Fields{
			"symbol": AS.Symbol.Symbol,
			"err":    err.Error(),
		}).Error("Failed to initialize DiffDepth stream")
		return err
	}
	AS.Orderbook, err = NewOrderBook(AS.Client, AS.Symbol.Symbol, 1000)
	if err != nil {
		AS.log.WithFields(logrus.Fields{
			"symbol": AS.Symbol.Symbol,
			"err":    err.Error(),
		}).Error("Failed to get orderbook snapshot")

		<-AS.orderbookStopC
		<-AS.orderbookDoneC
		for len(AS.conn) > 0 {
			<-AS.conn
		}
		AS.Orderbook = nil
		return err
	}
	AS.orderbookDoneC = doneC
	AS.orderbookStopC = stopC
	AS.log.WithFields(logrus.Fields{
		"symbol": AS.Symbol.Symbol,
	}).Info("Successfully initialized orderbook")
	return nil
}

func (AS *AbstractStrategy) depthUpdateHandler(event *futures.WsDepthEvent) {
	var (
		update  *futures.WsDepthEvent
		updated int
	)

	AS.conn <- event
	if AS.Orderbook != nil {
		for update = range AS.conn {
			updated = AS.Orderbook.Update(update)
		}
		if updated < 0 {
			AS.log.WithFields(logrus.Fields{
				"symbol": AS.Symbol.Symbol,
				"err":    updated,
			}).Error("Failed to update orderbook")

			if updated < -1 {
				AS.Orderbook = nil
				<-AS.orderbookStopC
				<-AS.orderbookDoneC
				AS.log.WithFields(logrus.Fields{
					"symbol": AS.Symbol.Symbol,
				}).Error("Restarting orderbook...")

				err := AS.InitOrderBook()
				if err != nil {
					AS.log.WithFields(logrus.Fields{
						"symbol": AS.Symbol.Symbol,
						"err":    err,
					}).Fatal("Failed to restart orderbook")
				}
			}
		} else {
			if updated == 0 {
				if AS.On {
					AS.log.WithFields(logrus.Fields{
						"symbol": AS.Symbol.Symbol,
					}).Info("Orderbook was updated successfully")
					AS.OnDepthUpdate()
				}
			}
		}
	}
}

func (AS *AbstractStrategy) depthErrorHandler(err error) {
	AS.log.WithFields(logrus.Fields{
		"symbol": AS.Symbol.Symbol,
	}).Error("DiffDepth stream error. Restarting...")

	for len(AS.conn) > 0 {
		<-AS.conn
	}
	AS.Orderbook = nil
	<-AS.orderbookStopC
	<-AS.orderbookDoneC
	err = AS.InitOrderBook()
	AS.log.WithFields(logrus.Fields{
		"symbol": AS.Symbol.Symbol,
	}).Fatal("Failed to restart DiffDepth stream")
}

func (AS *AbstractStrategy) InitClusters() error {
	AS.Clusters = NewClusters(AS.ClusterStrategy.TimeFrame)
	doneC, stopC, err := futures.WsAggTradeServe(AS.Symbol.Symbol, AS.tradeUpdateHandler, AS.tradeErrorHandler)
	if err != nil {
		AS.log.WithFields(logrus.Fields{
			"symbol": AS.Symbol.Symbol,
			"err":    err.Error(),
		}).Error("Failed to initialize AggTrade stream")
		return err
	}
	AS.clusterDoneC = doneC
	AS.clusterStopC = stopC
	AS.log.WithFields(logrus.Fields{
		"symbol": AS.Symbol.Symbol,
	}).Info("Successfully initialized clusters")
	return nil
}

func (AS *AbstractStrategy) tradeUpdateHandler(event *futures.WsAggTradeEvent) {
	AS.Clusters.Update(event)
	AS.log.WithFields(logrus.Fields{
		"symbol": AS.Symbol.Symbol,
	}).Info("Clusters were updated successfully")

	if AS.On {
		AS.OnCLusterUpdate()
	}
}

func (AS *AbstractStrategy) tradeErrorHandler(err error) {
	AS.log.WithFields(logrus.Fields{
		"symbol": AS.Symbol.Symbol,
		"err":    err,
	}).Error("AggTrade stream error. Restarting...")
	<-AS.clusterStopC
	<-AS.clusterDoneC
	err = AS.InitClusters()
	if err != nil {
		AS.log.WithFields(logrus.Fields{
			"symbol": AS.Symbol.Symbol,
			"err":    err,
		}).Fatal("Failed to restart clusters")
	}
}

func (AS *AbstractStrategy) InitCandles() error {
	candles, err := NewCandles(AS.Client, AS.Symbol.Symbol, strconv.Itoa(AS.CandlesStrategy.TimeFrame)+"m", AS.InitCandlesNum)
	if err != nil {
		AS.log.WithFields(logrus.Fields{
			"symbol": AS.Symbol.Symbol,
			"err":    err.Error(),
		}).Error("Failed to get candles")
		return err
	}
	AS.Candles = candles

	doneC, stopC, err := futures.WsKlineServe(AS.Symbol.Symbol, strconv.Itoa(AS.CandlesStrategy.TimeFrame)+"m", AS.candleUpdateHandler, AS.candleErrorHandler)
	if err != nil {
		AS.log.WithFields(logrus.Fields{
			"symbol": AS.Symbol.Symbol,
			"err":    err.Error(),
		}).Error("Failed to initialize Kline stream")
		return err
	}
	AS.candlesDoneC = doneC
	AS.candlesStopC = stopC
	AS.log.WithFields(logrus.Fields{
		"symbol": AS.Symbol.Symbol,
	}).Info("Successfully initialized candles")

	return nil
}

func (AS *AbstractStrategy) candleUpdateHandler(event *futures.WsKlineEvent) {
	AS.Candles.Update(&event.Kline)
	AS.log.WithFields(logrus.Fields{
		"symbol": AS.Symbol.Symbol,
	}).Info("Candles were updated successfullys")
	if AS.On {
		AS.OnCandleUpdate()
	}
}

func (AS *AbstractStrategy) candleErrorHandler(err error) {
	AS.log.WithFields(logrus.Fields{
		"symbol": AS.Symbol.Symbol,
		"err":    err,
	}).Error("Kline stream error. Restarting...")
	<-AS.candlesStopC
	<-AS.candlesDoneC
	AS.Candles = nil
	err = AS.InitCandles()
	if err != nil {
		AS.log.WithFields(logrus.Fields{
			"symbol": AS.Symbol.Symbol,
			"err":    err,
		}).Fatal("Failed to restart candles")
	}
}
