package binance_modules

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/shopspring/decimal"
	"golang.org/x/exp/slices"
)

type OrderBook struct {
	futures.DepthResponse

	newAsks []futures.Ask
	newBids []futures.Bid

	firstUpdateProcessed bool
}

func NewOrderBook(client *futures.Client, symbol string, limit int) (*OrderBook, error) {
	orderbook := new(OrderBook)

	depth, err := client.NewDepthService().Symbol(symbol).Limit(limit).Do(context.Background())

	if err != nil {
		return orderbook, err
	}

	orderbook.DepthResponse = *depth

	return orderbook, nil
}

func (orderbook *OrderBook) Update(update *futures.WsDepthEvent) int {
	if update.LastUpdateID < orderbook.LastUpdateID {
		return -1
	}
	if !(orderbook.firstUpdateProcessed) && (update.FirstUpdateID > orderbook.LastUpdateID) {
		return -2
	}
	if (orderbook.firstUpdateProcessed) && (orderbook.LastUpdateID != update.PrevLastUpdateID) {
		return -3
	}

	orderbook.firstUpdateProcessed = true
	orderbook.LastUpdateID = update.LastUpdateID

	orderbook.updateAsks(update)
	orderbook.updateBids(update)

	orderbook.removeZeroPriceLevels()

	return 0
}

func (orderbook *OrderBook) updateAsks(update *futures.WsDepthEvent) {
	var sortFunc = func(ask1, ask2 futures.Ask) bool {
		price1, _ := strconv.ParseFloat(ask1.Price, 64)
		price2, _ := strconv.ParseFloat(ask2.Price, 64)
		return price1 < price2
	}
	orderbook.newAsks = nil

	for _, newAsk := range update.Asks {
		index := slices.IndexFunc(orderbook.Asks, func(ask futures.Ask) bool { return ask.Price == newAsk.Price })
		if index == -1 {
			orderbook.newAsks = append(orderbook.newAsks, newAsk)
			orderbook.Asks = append(orderbook.Asks, newAsk)
		} else {
			newQuantity, _ := decimal.NewFromString(newAsk.Quantity)
			oldQuantity, _ := decimal.NewFromString(orderbook.Asks[index].Quantity)
			diff := newQuantity.Add(oldQuantity.Neg()).String()

			orderbook.newAsks = append(orderbook.newAsks, futures.Ask{Price: newAsk.Price, Quantity: diff})
			orderbook.Asks[index] = newAsk
		}
	}

	slices.SortFunc(orderbook.Asks, sortFunc)
	slices.SortFunc(orderbook.newAsks, sortFunc)
}

func (orderbook *OrderBook) updateBids(update *futures.WsDepthEvent) {
	var sortFunc = func(bid1, bid2 futures.Bid) bool {
		price1, _ := strconv.ParseFloat(bid1.Price, 64)
		price2, _ := strconv.ParseFloat(bid2.Price, 64)
		return price1 > price2
	}
	orderbook.newBids = nil

	for _, newBid := range update.Bids {
		index := slices.IndexFunc(orderbook.Bids, func(bid futures.Bid) bool { return bid.Price == newBid.Price })
		if index == -1 {
			orderbook.newBids = append(orderbook.newBids, newBid)
			orderbook.Bids = append(orderbook.Bids, newBid)
		} else {
			newQuantity, _ := decimal.NewFromString(newBid.Quantity)
			oldQuantity, _ := decimal.NewFromString(orderbook.Bids[index].Quantity)
			diff := newQuantity.Add(oldQuantity.Neg()).String()

			orderbook.newBids = append(orderbook.newBids, futures.Bid{Price: newBid.Price, Quantity: diff})
			orderbook.Bids[index] = newBid
		}
	}

	slices.SortFunc(orderbook.Bids, sortFunc)
	slices.SortFunc(orderbook.Bids, sortFunc)
}

func (orderbook *OrderBook) removeZeroPriceLevels() {
	var (
		nonZeroAsks []futures.Ask
		nonZeroBids []futures.Bid

		isZero = func(s string) bool {
			if len(s)-strings.Count(s, "0")-strings.Count(s, ".") > 0 {
				return false
			}
			return true
		}
	)

	for _, ask := range orderbook.Asks {

		if !isZero(ask.Quantity) {
			nonZeroAsks = append(nonZeroAsks, ask)
		}
	}

	for _, bid := range orderbook.Bids {
		if !isZero(bid.Quantity) {
			nonZeroBids = append(nonZeroBids, bid)
		}
	}

	orderbook.Asks = nonZeroAsks
	orderbook.Bids = nonZeroBids
}

func (orderbook *OrderBook) BestAsk() futures.Ask {
	return orderbook.Asks[0]
}

func (orderbook *OrderBook) BestBid() futures.Bid {
	return orderbook.Bids[0]
}

func (orderbook *OrderBook) Print(n int) {
	var i int

	fmt.Print("ASKS:\n")
	for i = n - 1; i >= 0; i-- {
		fmt.Printf("%v: %v", orderbook.Asks[i].Price, orderbook.Asks[i].Quantity)
	}
	fmt.Print("BIDS:\n")
	for i = 0; i < n; i++ {
		fmt.Printf("%v: %v", orderbook.Bids[i].Price, orderbook.Bids[i].Quantity)
	}

}
