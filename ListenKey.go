package binance_modules

import (
	"context"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

var listenKeyLock = &sync.Mutex{}

type ListenKey struct {
	listenKey string
	client    *futures.Client
}

var listenKeyInstance *ListenKey

func GetListenKey(client *futures.Client) (*ListenKey, error) {
	if listenKeyInstance == nil {
		listenKeyLock.Lock()
		defer listenKeyLock.Unlock()
		if listenKeyInstance == nil {
			lk, err := client.NewStartUserStreamService().Do(context.Background())
			if err != nil {
				return listenKeyInstance, err
			}
			listenKeyInstance = &ListenKey{listenKey: lk}
			go listenKeyInstance.Renew()
		}
	}
	return listenKeyInstance, nil
}

func (lk *ListenKey) Renew() {
	// renew listenKey every 55 minutes

	lastUpdateTime := time.Now()
	timeout := lastUpdateTime.Add(55 * time.Minute)
	for {
		if time.Now().After(timeout) {
			err := lk.client.NewKeepaliveUserStreamService().ListenKey(lk.listenKey).Do(context.Background())
			if err == nil {
				lastUpdateTime = time.Now()
				timeout = lastUpdateTime.Add(55 * time.Minute)
			}
		}
	}
}
