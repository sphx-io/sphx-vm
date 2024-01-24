package orderbook

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// Order strores information about request
type Order struct {
	id        string
	accountId string
	side      Side
	quantity  decimal.Decimal
	price     decimal.Decimal
	timestamp time.Time
}

// MarketView represents order book in a glance
type MarketView struct {
	Asks map[string]decimal.Decimal `json:"asks"`
	Bids map[string]decimal.Decimal `json:"bids"`
}

// NewOrder creates new constant object Order
func NewOrder(orderId string, accountId string, side Side, quantity, price decimal.Decimal, timestamp time.Time) *Order {
	return &Order{
		id:        orderId,
		accountId: accountId,
		side:      side,
		quantity:  quantity,
		price:     price,
		timestamp: timestamp,
	}
}

// ID returns orderID field copy
func (o *Order) ID() string {
	return o.id
}

// ID returns accountId field copy
func (o *Order) AccountId() string {
	return o.accountId
}

// Side returns side of the order
func (o *Order) Side() Side {
	return o.side
}

// Quantity returns quantity field copy
func (o *Order) Quantity() decimal.Decimal {
	return o.quantity
}

// Price returns price field copy
func (o *Order) Price() decimal.Decimal {
	return o.price
}

// Time returns timestamp field copy
func (o *Order) Time() time.Time {
	return o.timestamp
}

// String implements Stringer interface
func (o *Order) String() string {
	return fmt.Sprintf("\n\"%s\":\n\taccountId: %s\n\tside: %s\n\tquantity: %s\n\tprice: %s\n\ttime: %s\n", o.ID(), o.AccountId(), o.Side(), o.Quantity(), o.Price(), o.Time())
}

// MarshalJSON implements json.Marshaler interface
func (o *Order) MarshalJSON() ([]byte, error) {
	return json.Marshal(
		&struct {
			ID        string          `json:"id"`
			AccountId string          `json:"accountId"`
			Timestamp time.Time       `json:"timestamp"`
			Quantity  decimal.Decimal `json:"quantity"`
			Price     decimal.Decimal `json:"price"`
			S         Side            `json:"side"`
		}{
			S:         o.Side(),
			ID:        o.ID(),
			AccountId: o.AccountId(),
			Timestamp: o.Time(),
			Quantity:  o.Quantity(),
			Price:     o.Price(),
		},
	)
}

// UnmarshalJSON implements json.Unmarshaler interface
func (o *Order) UnmarshalJSON(data []byte) error {
	obj := struct {
		ID        string          `json:"id"`
		AccountId string          `json:"accountId"`
		Timestamp time.Time       `json:"timestamp"`
		Quantity  decimal.Decimal `json:"quantity"`
		Price     decimal.Decimal `json:"price"`
		S         Side            `json:"side"`
	}{}

	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	o.id = obj.ID
	o.accountId = obj.AccountId
	o.timestamp = obj.Timestamp
	o.quantity = obj.Quantity
	o.price = obj.Price
	o.side = obj.S
	return nil
}

// GetOrderSide gets the orderside along with its orders in one side of the market
func (ob *OrderBook) GetOrderSide(side Side) *OrderSide {
	switch side {
	case Buy:
		return ob.bids
	default:
		return ob.asks
	}
}

// MarketOverview gives an overview of the market including the quantities and prices of each side in the market
// asks:   qty   price       bids:  qty   price
//
//	0.2   14                 0.9   13
//	0.1   14.5               5     14
//	0.8   16                 2     16
func (ob *OrderBook) MarketOverview() *MarketView {

	return &MarketView{
		Asks: compileOrders(ob.asks),
		Bids: compileOrders(ob.bids),
	}
}

// compileOrders compiles orders in the following format
func compileOrders(orders *OrderSide) map[string]decimal.Decimal {
	// show queue
	queue := make(map[string]decimal.Decimal)

	if orders != nil {
		level := orders.MaxPriceQueue()
		for level != nil {
			if q, exists := queue[level.Price().String()]; exists {
				queue[level.Price().String()] = q.Add(level.Volume())
			} else {
				queue[level.Price().String()] = level.Volume()
			}

			level = orders.LessThan(level.Price())
		}

	}

	return queue
}
