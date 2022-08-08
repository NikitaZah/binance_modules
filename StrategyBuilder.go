package binance_modules

import (
	"github.com/adshao/go-binance/v2/futures"
)

type StrategyBuilder struct {
	client *futures.Client
	symbol *futures.Symbol
}

func NewStrategyBuilder(client *futures.Client, symbol string) (*StrategyBuilder, error) {
	builder := new(StrategyBuilder)
	builder.client = client

	exInfo, err := GetExchangeInfo(client)
	if err != nil {
		return builder, err
	}

	builder.symbol = exInfo.Symbol(symbol)
	return builder, nil
}

func (SB *StrategyBuilder) RegisterStrategy(strategy BaseStrategyInterface) {
	strategy.SetClient(SB.client)
	strategy.SetSymbol(SB.symbol)
	strategy.SetLogger()
}

func (SB *StrategyBuilder) LaunchAccount(strategy AccountStrategyInterface) error {
	var err error
	err = strategy.InitAccount()
	return err
}

func (SB *StrategyBuilder) LaunchOrderBook(strategy OrderBookStrategyInterface) error {
	var err error
	err = strategy.InitOrderBook()
	return err
}

func (SB *StrategyBuilder) LaunchClusters(strategy ClusterStrategyInterface) error {
	var err error
	err = strategy.InitClusters()
	return err
}

func (SB *StrategyBuilder) LaunchCandles(strategy CandlesStrategyInterface) error {
	var err error
	err = strategy.InitCandles()
	return err
}

func (SB *StrategyBuilder) Run(strategy BaseStrategyInterface) {
	strategy.Activate()
}
