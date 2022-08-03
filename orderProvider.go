package binance_modules

import (
	"context"
	"fmt"
	"strings"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/shopspring/decimal"
)

type OrderProvider struct {
	client futures.Client
}

func NewOrderProvider(client *futures.Client) OrderProvider {
	return OrderProvider{client: *client}
}

func (op *OrderProvider) SetLeverage(symbol futures.Symbol, leverage int) (*futures.SymbolLeverage, error) {
	response, err := op.client.NewChangeLeverageService().Symbol(symbol.Symbol).Leverage(leverage).Do(context.Background())

	return response, err
}

func (op *OrderProvider) MarketOrder(symbol futures.Symbol, side string, quantity float64) (*futures.CreateOrderResponse, error) {
	var (
		fmt_symbol   string
		fmt_quantity string
		fmt_side     futures.SideType
		fmt_type     futures.OrderType
	)
	fmt_symbol = symbol.Symbol
	fmt_quantity = op.quantityToString(quantity, symbol.LotSizeFilter())
	fmt_type = futures.OrderTypeMarket

	if side == "BUY" {
		fmt_side = futures.SideTypeBuy
	} else {
		fmt_side = futures.SideTypeSell
	}

	order, err := op.client.NewCreateOrderService().
		Symbol(fmt_symbol).
		Side(fmt_side).Type(fmt_type).
		Quantity(fmt_quantity).
		Do(context.Background())

	return order, err
}

func (op *OrderProvider) LimitOrder(symbol futures.Symbol, side string, quantity, price float64) (*futures.CreateOrderResponse, error) {
	var (
		fmt_symbol   string
		fmt_quantity string
		fmt_price    string
		fmt_side     futures.SideType
		fmt_type     futures.OrderType
		fmt_time     futures.TimeInForceType
	)
	fmt_symbol = symbol.Symbol
	fmt_quantity = op.quantityToString(quantity, symbol.LotSizeFilter())
	fmt_price = op.priceToString(price, symbol.PriceFilter())
	fmt_type = futures.OrderTypeLimit
	fmt_type = futures.OrderType(futures.TimeInForceTypeGTC)

	if side == "BUY" {
		fmt_side = futures.SideTypeBuy
	} else {
		fmt_side = futures.SideTypeSell
	}

	order, err := op.client.NewCreateOrderService().
		Symbol(fmt_symbol).
		Side(fmt_side).
		Type(fmt_type).
		Quantity(fmt_quantity).
		Price(fmt_price).
		TimeInForce(fmt_time).
		Do(context.Background())

	return order, err
}

func (op *OrderProvider) priceToString(price float64, filter *futures.PriceFilter) string {
	precision := strings.Index(filter.TickSize, "1")
	extra_zeroes := len(filter.TickSize) - precision - 1

	if precision > 0 {
		precision -= 1
	}
	stringPrice := decimal.NewFromFloat(price).Round(int32(precision)).StringFixed(int32(precision) + int32(extra_zeroes))

	return stringPrice
}

func (op *OrderProvider) quantityToString(quantity float64, filter *futures.LotSizeFilter) string {
	precision := strings.Index(filter.StepSize, "1")

	if precision > 0 {
		precision -= 1
	}

	stringQuantity := decimal.NewFromFloat(quantity).Round(int32(precision)).String()

	return stringQuantity
}

func (op *OrderProvider) Testing(symbols []futures.Symbol, price float64) {

	fmt.Printf("initial price: %v\n", price)
	for _, symbol := range symbols {
		strQ := op.priceToString(price, symbol.PriceFilter())
		fmt.Printf("string price: %v\ntick size: %v\n", strQ, symbol.PriceFilter().TickSize)
	}
}
