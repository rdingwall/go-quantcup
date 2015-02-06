package main

import (
	"fmt"
)

type Price uint16 // 0-65536 eg the price 123.45 = 12345
type OrderID uint64
type Size uint64
type Side int

type Order struct {
	symbol string
	trader string
	side   Side
	price  Price
	size   Size
}

// Execution Report (send one per opposite-sided order completely filled).
type Execution Order

const (
	Bid Side = iota
	Ask
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
