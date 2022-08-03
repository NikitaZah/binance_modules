package binance_modules

import (
	"fmt"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

type priceLevel struct {
	price      string
	buyVolume  float64
	sellVolume float64
}

func newPriceLevel(trade *futures.WsAggTradeEvent) priceLevel {
	result := priceLevel{price: trade.Price}

	if trade.Maker {
		result.sellVolume, _ = strconv.ParseFloat(trade.Quantity, 64)
	} else {
		result.buyVolume, _ = strconv.ParseFloat(trade.Quantity, 64)
	}

	return result
}

func (pl *priceLevel) Quantity() float64 {
	return pl.buyVolume + pl.sellVolume
}

func (pl *priceLevel) Update(update *futures.WsAggTradeEvent) {
	quantity, _ := strconv.ParseFloat(update.Quantity, 64)
	if update.Maker {
		pl.sellVolume += quantity
	} else {
		pl.buyVolume += quantity
	}
}

type cluster struct {
	timeStart int // minute
	timeEnd   int // minute
	levels    map[string]priceLevel
}

func NewCluster(timeStart, timeEnd int) cluster {
	res := cluster{timeStart: timeStart, timeEnd: timeEnd, levels: make(map[string]priceLevel)}
	return res
}

func (c *cluster) Length() float64 {
	var (
		maxPrice float64
		minPrice float64
	)

	minPrice = -1
	maxPrice = -1

	for price := range c.levels {
		fPrice, _ := strconv.ParseFloat(price, 64)
		if minPrice == -1 {
			minPrice = fPrice
			maxPrice = fPrice
		}

		if fPrice > maxPrice {
			maxPrice = fPrice
		} else if fPrice < minPrice {
			minPrice = fPrice
		}
	}

	return ((maxPrice / minPrice) - 1) * 100
}

func (c *cluster) Update(update *futures.WsAggTradeEvent) bool {
	t := time.UnixMilli(update.Time).Minute()

	if (t < c.timeStart) || (t >= c.timeEnd) {
		return false
	}

	pl, ok := c.levels[update.Price]

	if ok {
		pl.Update(update)
		c.levels[update.Price] = pl
	} else {
		c.levels[update.Price] = newPriceLevel(update)
	}
	return true
}

type Clusters struct {
	clusters         []cluster
	timeFrameMinutes int
}

func NewClusters(timeFrame int) *Clusters {
	res := new(Clusters)
	res.timeFrameMinutes = timeFrame

	timeStart, timeEnd := res.currentTimeBounds()
	res.clusters = append(res.clusters, NewCluster(timeStart, timeEnd))

	return res
}

func (c *Clusters) Update(update *futures.WsAggTradeEvent) {
	updated := c.clusters[len(c.clusters)-1].Update(update)
	fmt.Printf("Update:\n %v\n", update)
	if !updated {
		c.add(update)
	}
}

func (c *Clusters) add(update *futures.WsAggTradeEvent) {
	var (
		timeStart int // minute
		timeEnd   int // minute
	)

	timeStart, timeEnd = c.currentTimeBounds()

	c.clusters = append(c.clusters, NewCluster(timeStart, timeEnd))
	c.Update(update)
}

func (c *Clusters) currentTimeBounds() (int, int) {
	var (
		timeStart int // minute
		timeEnd   int // minute
	)
	minute := time.Now().Minute()
	if minute%c.timeFrameMinutes == 0 {
		timeStart = minute
	} else {
		for ; minute%c.timeFrameMinutes != 0; minute-- {
		}
		timeStart = minute
	}
	timeEnd = timeStart + c.timeFrameMinutes
	return timeStart, timeEnd
}

func (c *Clusters) Print() {
	// sortFunc := func(pl1, pl2 priceLevel) bool {
	// 	price1, _ := strconv.ParseFloat(pl1.price, 64)
	// 	price2, _ := strconv.ParseFloat(pl2.price, 64)

	// 	return price1 < price2
	// }
	if len(c.clusters) > 0 {
		current := c.clusters[len(c.clusters)-1]

		fmt.Println(current.levels)
	}
}
