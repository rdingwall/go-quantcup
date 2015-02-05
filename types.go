package main

import (
	"fmt"
)

type (
	OrderID uint64

	// Price (0-65536 interpreted as divided by 100).
	// eg the range is 000.00-655.36
	// eg the price 123.45 = 12345
	// eg the price 23.45 = 2345
	// eg the price 23.4 = 2340
	Price uint16

	// Order Size.
	Size uint64

	// Side (Ask=1, Bid=0).
	Side int

	// Limit Order.
	Order struct {
		symbol string
		trader string
		side   Side
		price  Price
		size   Size
	}

	// Execution Report (send one per opposite-sided order completely filled).
	Execution Order
)

const (
	Bid Side = 0
	Ask      = 1
)

func (o *Execution) String() string {
	return fmt.Sprintf("{symbol: %v, trader: %v, side: %v, price: %v, size: %v}", o.symbol, o.trader, o.side, o.price, o.size)
}

func (o *Order) String() string {
	return fmt.Sprintf("{symbol: %v, trader: %v, side: %v, price: %v, size: %v}", o.symbol, o.trader, o.side, o.price, o.size)
}

func (s Side) String() string {
	switch s {
	case Bid:
		return "Bid"
	default:
		return "Ask"
	}
}
