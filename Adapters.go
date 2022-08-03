package binance_modules

import (
	"github.com/adshao/go-binance/v2/futures"
)

func KlineAdapter(kline *futures.WsKline) *futures.Kline {
	restKline := futures.Kline{
		OpenTime:                 kline.StartTime,
		CloseTime:                kline.EndTime,
		Open:                     kline.Open,
		High:                     kline.High,
		Low:                      kline.Low,
		Close:                    kline.Close,
		Volume:                   kline.Volume,
		QuoteAssetVolume:         kline.QuoteVolume,
		TradeNum:                 kline.TradeNum,
		TakerBuyBaseAssetVolume:  kline.ActiveBuyVolume,
		TakerBuyQuoteAssetVolume: kline.ActiveBuyQuoteVolume,
	}

	return &restKline
}

func OrderAdapter(order *futures.WsOrderTradeUpdate) *futures.Order {
	restOrder := futures.Order{
		Symbol:           order.Symbol,
		OrderID:          order.ID,
		ClientOrderID:    order.ClientOrderID,
		Price:            order.OriginalPrice,
		ReduceOnly:       order.IsReduceOnly,
		OrigQuantity:     order.OriginalQty,
		ExecutedQuantity: order.AccumulatedFilledQty,
		Status:           order.Status,
		TimeInForce:      order.TimeInForce,
		Type:             order.Type,
		Side:             order.Side,
		StopPrice:        order.StopPrice,
		ActivatePrice:    order.ActivationPrice,
		PriceRate:        order.CallbackRate,
		AvgPrice:         order.AveragePrice,
		OrigType:         string(order.OriginalType),
		PositionSide:     order.PositionSide,
		ClosePosition:    order.IsClosingPosition,
	}

	return &restOrder
}

func PositionAdapter(position *futures.WsPosition) *futures.PositionRisk {
	restPosition := futures.PositionRisk{
		EntryPrice:       position.EntryPrice,
		MarginType:       string(position.MarginType),
		MarkPrice:        position.MarkPrice,
		PositionAmt:      position.Amount,
		Symbol:           position.Symbol,
		UnRealizedProfit: position.UnrealizedPnL,
		PositionSide:     string(position.Side),
		IsolatedWallet:   position.IsolatedWallet,
	}
	return &restPosition
}

func BalanceAdapter(balance *futures.WsBalance) *futures.Balance {
	restBalance := futures.Balance{
		Asset:              balance.Asset,
		Balance:            balance.Balance,
		CrossWalletBalance: balance.CrossWalletBalance,
	}
	return &restBalance
}

func CreateOrderAdapter(order *futures.CreateOrderResponse) *futures.Order {
	restOrder := futures.Order{
		Symbol:           order.Symbol,
		OrderID:          order.OrderID,
		ClientOrderID:    order.ClientOrderID,
		Price:            order.Price,
		OrigQuantity:     order.OrigQuantity,
		ExecutedQuantity: order.ExecutedQuantity,
		CumQuote:         order.CumQuote,
		Status:           order.Status,
		TimeInForce:      order.TimeInForce,
		Type:             order.Type,
		Side:             order.Side,
		StopPrice:        order.StopPrice,
		UpdateTime:       order.UpdateTime,
		WorkingType:      order.WorkingType,
		ActivatePrice:    order.ActivatePrice,
		PriceRate:        order.PriceRate,
		AvgPrice:         order.AvgPrice,
		OrigType:         string(order.Type),
		PositionSide:     order.PositionSide,
		PriceProtect:     order.PriceProtect,
		ClosePosition:    order.ClosePosition,
	}
	return &restOrder
}
