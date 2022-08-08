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

type BaseStrategy struct {
	Client *futures.Client
	Symbol *futures.Symbol
	On     bool
	log    *Logger
}

func (BS *BaseStrategy) SetClient(client *futures.Client) {
	BS.Client = client
}

func (BS *BaseStrategy) SetSymbol(symbol *futures.Symbol) {
	BS.Symbol = symbol
}

func (BS *BaseStrategy) SetLogger() error {
	lg, err := GetLogger()
	if err != nil {
		return err
	}
	BS.log = lg
	return nil
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

func (AS *AccountStrategy) accountUpdateHandler(event *futures.WsUserDataEvent) {
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

func (AS *AccountStrategy) accountErrorHandler(err error) {
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
	doneC, stopC, err := futures.WsDiffDepthServeWithRate(OS.Symbol.Symbol, 100*time.Millisecond, OS.depthUpdateHandler, OS.depthErrorHandler)
	if err != nil {
		OS.log.WithFields(logrus.Fields{
			"symbol": OS.Symbol.Symbol,
			"err":    err.Error(),
		}).Error("Failed to initialize DiffDepth stream")
		return err
	}
	OS.Orderbook, err = NewOrderBook(OS.Client, OS.Symbol.Symbol, 1000)
	if err != nil {
		OS.log.WithFields(logrus.Fields{
			"symbol": OS.Symbol.Symbol,
			"err":    err.Error(),
		}).Error("Failed to get orderbook snapshot")

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
	OS.log.WithFields(logrus.Fields{
		"symbol": OS.Symbol.Symbol,
	}).Info("Successfully initialized orderbook")
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
		if updated < 0 {
			OS.log.WithFields(logrus.Fields{
				"symbol": OS.Symbol.Symbol,
				"err":    updated,
			}).Error("Failed to update orderbook")

			if updated < -1 {
				OS.Orderbook = nil
				<-OS.orderbookStopC
				<-OS.orderbookDoneC
				OS.log.WithFields(logrus.Fields{
					"symbol": OS.Symbol.Symbol,
				}).Error("Restarting orderbook...")

				err := OS.InitOrderBook()
				if err != nil {
					OS.log.WithFields(logrus.Fields{
						"symbol": OS.Symbol.Symbol,
						"err":    err,
					}).Fatal("Failed to restart orderbook")
				}
			}
		} else {
			if updated == 0 {
				if OS.On {
					OS.log.WithFields(logrus.Fields{
						"symbol": OS.Symbol.Symbol,
					}).Info("Orderbook was updated successfully")
					OS.OnDepthUpdate()
				}
			}
		}
	}
}

func (OS *OrderBookStrategy) depthErrorHandler(err error) {
	OS.log.WithFields(logrus.Fields{
		"symbol": OS.Symbol.Symbol,
	}).Error("DiffDepth stream error. Restarting...")

	for len(OS.conn) > 0 {
		<-OS.conn
	}
	OS.Orderbook = nil
	<-OS.orderbookStopC
	<-OS.orderbookDoneC
	err = OS.InitOrderBook()
	OS.log.WithFields(logrus.Fields{
		"symbol": OS.Symbol.Symbol,
	}).Fatal("Failed to restart DiffDepth stream")
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
		CS.log.WithFields(logrus.Fields{
			"symbol": CS.Symbol.Symbol,
			"err":    err.Error(),
		}).Error("Failed to initialize AggTrade stream")
		return err
	}
	CS.clusterDoneC = doneC
	CS.clusterStopC = stopC
	CS.log.WithFields(logrus.Fields{
		"symbol": CS.Symbol.Symbol,
	}).Info("Successfully initialized clusters")
	return nil
}

func (CS *ClusterStrategy) tradeUpdateHandler(event *futures.WsAggTradeEvent) {
	CS.Clusters.Update(event)
	CS.log.WithFields(logrus.Fields{
		"symbol": CS.Symbol.Symbol,
	}).Info("Clusters were updated successfully")

	if CS.On {
		CS.OnCLusterUpdate()
	}
}

func (CS *ClusterStrategy) tradeErrorHandler(err error) {
	CS.log.WithFields(logrus.Fields{
		"symbol": CS.Symbol.Symbol,
		"err":    err,
	}).Error("AggTrade stream error. Restarting...")
	<-CS.clusterStopC
	<-CS.clusterDoneC
	err = CS.InitClusters()
	if err != nil {
		CS.log.WithFields(logrus.Fields{
			"symbol": CS.Symbol.Symbol,
			"err":    err,
		}).Fatal("Failed to restart clusters")
	}
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
		CS.log.WithFields(logrus.Fields{
			"symbol": CS.Symbol.Symbol,
			"err":    err.Error(),
		}).Error("Failed to get candles")
		return err
	}
	CS.Candles = candles

	doneC, stopC, err := futures.WsKlineServe(CS.Symbol.Symbol, strconv.Itoa(CS.TimeFrame)+"m", CS.candleUpdateHandler, CS.candleErrorHandler)
	if err != nil {
		CS.log.WithFields(logrus.Fields{
			"symbol": CS.Symbol.Symbol,
			"err":    err.Error(),
		}).Error("Failed to initialize Kline stream")
		return err
	}
	CS.candlesDoneC = doneC
	CS.candlesStopC = stopC
	CS.log.WithFields(logrus.Fields{
		"symbol": CS.Symbol.Symbol,
	}).Info("Successfully initialized candles")

	return nil
}

func (CS *CandlesStrategy) candleUpdateHandler(event *futures.WsKlineEvent) {
	CS.Candles.Update(&event.Kline)
	CS.log.WithFields(logrus.Fields{
		"symbol": CS.Symbol.Symbol,
	}).Info("Candles were updated successfullys")
	if CS.On {
		CS.OnCandleUpdate()
	}
}

func (CS *CandlesStrategy) candleErrorHandler(err error) {
	CS.log.WithFields(logrus.Fields{
		"symbol": CS.Symbol.Symbol,
		"err":    err,
	}).Error("Kline stream error. Restarting...")
	<-CS.candlesStopC
	<-CS.candlesDoneC
	CS.Candles = nil
	err = CS.InitCandles()
	if err != nil {
		CS.log.WithFields(logrus.Fields{
			"symbol": CS.Symbol.Symbol,
			"err":    err,
		}).Fatal("Failed to restart candles")
	}
}
