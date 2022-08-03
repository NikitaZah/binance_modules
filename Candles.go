package binance_modules

import (
	"context"

	"github.com/adshao/go-binance/v2/futures"
)

type Candles struct {
	candles []*futures.Kline
}

func NewCandles(client *futures.Client, symbol, timeframe string, limit int) (*Candles, error) {
	candles := new(Candles)
	klines, err := client.NewKlinesService().Symbol(symbol).Interval(timeframe).Limit(limit).Do(context.Background())

	if err != nil {
		return candles, err
	}
	candles.candles = klines
	return candles, nil
}

func (c *Candles) Update(update *futures.WsKline) {
	if len(c.candles) > 0 {
		if c.candles[len(c.candles)-1].OpenTime == update.StartTime {
			c.candles[len(c.candles)-1] = KlineAdapter(update)
		}
		if c.candles[len(c.candles)-1].OpenTime < update.StartTime {
			c.candles = append(c.candles, KlineAdapter(update))
		}
	} else {
		c.candles = append(c.candles, KlineAdapter(update))
	}
}
