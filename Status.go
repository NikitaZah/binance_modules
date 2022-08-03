package binance_modules

import (
	"context"
	"strconv"

	"github.com/adshao/go-binance/v2/futures"
)

type Status struct {
	symbol   string
	Balance  *futures.Balance
	Position *futures.PositionRisk
	Orders   []*futures.Order
}

func NewStatus(client *futures.Client, symbol string) (*Status, error) {
	status := new(Status)
	status.symbol = symbol

	underlyingAsset := symbol[len(symbol)-4:]

	balances, err := client.NewGetBalanceService().Do(context.Background())
	if err != nil {
		return status, err
	}
	for _, balance := range balances {
		if balance.Asset == underlyingAsset {
			status.Balance = balance
			break
		}
	}

	positions, err := client.NewGetPositionRiskService().Symbol(symbol).Do(context.Background())
	if err != nil {
		return status, err
	}
	status.Position = positions[0]

	orders, err := client.NewListOpenOrdersService().Symbol(symbol).Do(context.Background())
	if err != nil {
		return status, err
	}
	status.Orders = orders

	return status, nil
}

func (s *Status) AccountUpdate(update *futures.WsAccountUpdate) {
	for _, balance := range update.Balances {
		if balance.Asset == s.Balance.Asset {
			s.Balance = BalanceAdapter(&balance)
			break
		}
	}
	for _, position := range update.Positions {
		if position.Symbol == s.symbol {
			s.Position = PositionAdapter(&position)
		}
	}
}

func (s *Status) OrderUpdate(update *futures.WsOrderTradeUpdate) {
	for i := range s.Orders {
		if s.Orders[i].OrderID == update.ID {
			s.Orders[i] = OrderAdapter(update)
			return
		}
	}
	s.Orders = append(s.Orders, OrderAdapter(update))
}

func (s *Status) CreateOrderUpdate(update *futures.CreateOrderResponse) {
	s.Orders = append(s.Orders, CreateOrderAdapter(update))
}

func (s *Status) PositionStatus() string {
	var (
		status      string
		positionAmt float64
	)
	positionAmt, _ = strconv.ParseFloat(s.Position.PositionAmt, 64)
	if positionAmt != 0.0 {
		for i := range s.Orders {
			if (s.Orders[i].Status == futures.OrderStatusTypeNew) || (s.Orders[i].Status == futures.OrderStatusTypePartiallyFilled) {
				if s.Orders[i].ClosePosition == true {
					status = "CLOSING"
					return status
				} else {
					if (s.Orders[i].Side == futures.SideTypeBuy) && (positionAmt < 0) {
						status = "CLOSING"
						return status
					}
					if (s.Orders[i].Side == futures.SideTypeSell) && (positionAmt > 0) {
						status = "CLOSING"
						return status
					}
				}
			}
		}
		status = "OPENED"
		return status
	} else {
		for i := range s.Orders {
			if (s.Orders[i].Status == futures.OrderStatusTypeNew) && (s.Orders[i].ReduceOnly == false) && (s.Orders[i].ClosePosition == false) {
				status = "OPENING"
				return status
			}
		}
		status = "CLOSED"
		return status
	}
}
