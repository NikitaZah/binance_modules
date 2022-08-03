package binance_modules

import (
	"context"
	"sync"

	"github.com/adshao/go-binance/v2/futures"
)

var exchangeInfoLock = &sync.Mutex{}

type ExchangeInfo struct {
	futures.ExchangeInfo
}

var exchangeInfoInstance *ExchangeInfo

func GetExchangeInfo(client *futures.Client) (*ExchangeInfo, error) {
	if exchangeInfoInstance == nil {
		exchangeInfoLock.Lock()
		defer exchangeInfoLock.Unlock()
		if exchangeInfoInstance == nil {
			exInfo, err := client.NewExchangeInfoService().Do(context.Background())

			if err != nil {
				return exchangeInfoInstance, err
			}
			exchangeInfoInstance = &ExchangeInfo{*exInfo}
		}
	}
	return exchangeInfoInstance, nil
}

func (exInfo *ExchangeInfo) Symbol(token string) *futures.Symbol {
	for _, symbol := range exInfo.Symbols {
		if symbol.Symbol == token {
			return &symbol
		}
	}
	return nil
}
