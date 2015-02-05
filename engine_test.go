package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type Test struct {
	Orders   []Order
	Cancels  []OrderID
	Orders2  []Order
	Expected []Execution
}

const maxExecutionCount int = 100

var (
	oa101x100 = Order{"JPM", "MAX", Ask, 101, 100}
	ob101x100 = Order{"JPM", "MAX", Bid, 101, 100}
	oa101x50  = Order{"JPM", "MAX", Ask, 101, 50}
	ob101x50  = Order{"JPM", "MAX", Bid, 101, 50}
	oa101x25  = Order{"JPM", "MAX", Ask, 101, 25}
	ob101x25  = Order{"JPM", "MAX", Bid, 101, 25}
	ob101x25x = Order{"JPM", "XAM", Bid, 101, 25}

	xa101x100 = Execution{"JPM", "MAX", Ask, 101, 100}
	xb101x100 = Execution{"JPM", "MAX", Bid, 101, 100}
	xa101x50  = Execution{"JPM", "MAX", Ask, 101, 50}
	xb101x50  = Execution{"JPM", "MAX", Bid, 101, 50}
	xa101x25  = Execution{"JPM", "MAX", Ask, 101, 25}
	xb101x25  = Execution{"JPM", "MAX", Bid, 101, 25}
	xb101x25x = Execution{"JPM", "XAM", Bid, 101, 25}
)

func TestAsk(t *testing.T) {
	runTest(t, &Test{Orders: []Order{oa101x100}})
}

func TestBid(t *testing.T) {
	runTest(t, &Test{Orders: []Order{ob101x100}})
}

func TestExecution(t *testing.T) {
	runTest(t, &Test{Orders: []Order{oa101x100, ob101x100}, Expected: []Execution{xa101x100, xb101x100}})
}

func TestReordering1(t *testing.T) {
	runTest(t, &Test{Orders: []Order{oa101x100, ob101x100}, Expected: []Execution{xb101x100, xa101x100}})
}

func TestReordering2(t *testing.T) {
	runTest(t, &Test{Orders: []Order{ob101x100, oa101x100}, Expected: []Execution{xa101x100, xb101x100}})
}

func TestReordering3(t *testing.T) {
	runTest(t, &Test{Orders: []Order{ob101x100, oa101x100}, Expected: []Execution{xb101x100, xa101x100}})
}

func TestPartialFill1(t *testing.T) {
	runTest(t, &Test{Orders: []Order{oa101x100, ob101x50}, Expected: []Execution{xa101x50, xb101x50}})
}

func TestPartialFill2(t *testing.T) {
	runTest(t, &Test{Orders: []Order{oa101x50, ob101x100}, Expected: []Execution{xa101x50, xb101x50}})
}

func TestIncrementalOverFill1(t *testing.T) {
	runTest(t, &Test{Orders: []Order{oa101x100, ob101x25, ob101x25, ob101x25, ob101x25, ob101x25}, Expected: []Execution{xa101x25, xb101x25, xa101x25, xb101x25, xa101x25, xb101x25, xa101x25, xb101x25}})
}

func TestIncrementalOverFill2(t *testing.T) {
	runTest(t, &Test{Orders: []Order{ob101x100, oa101x25, oa101x25, oa101x25, oa101x25, oa101x25}, Expected: []Execution{xa101x25, xb101x25, xa101x25, xb101x25, xa101x25, xb101x25, xa101x25, xb101x25}})
}

func TestQueuePosition(t *testing.T) {
	runTest(t, &Test{Orders: []Order{ob101x25x, ob101x25, oa101x25}, Expected: []Execution{xa101x25, xb101x25x}})
}

func TestCancel(t *testing.T) {
	runTest(t, &Test{Orders: []Order{ob101x25}, Cancels: []OrderID{1}, Orders2: []Order{oa101x25}})
}

func TestCancelFromFrontOfQueue(t *testing.T) {
	runTest(t, &Test{Orders: []Order{ob101x25x, ob101x25}, Cancels: []OrderID{1}, Orders2: []Order{oa101x25}, Expected: []Execution{xa101x25, xb101x25}})
}

func TestCancelFrontBackOutOfOrderThenPartialExecution(t *testing.T) {
	runTest(t, &Test{Orders: []Order{ob101x100, ob101x25x, ob101x25x, ob101x50}, Cancels: []OrderID{1, 4, 3}, Orders2: []Order{oa101x50}, Expected: []Execution{xb101x25x, xa101x25}})
}

func runTest(t *testing.T, test *Test) {
	var executions []Execution
	var e Engine
	e.Reset()

	e.Execute = func(e Execution) {
		t.Logf("<- received execution: %v", &e)
		executions = append(executions, e)
		assert.False(t, len(executions) > maxExecutionCount, "too many executions, test array overflow")
	}

	curOrderID := feedOrders(t, &e, 0, &test.Orders)
	feedCancels(t, &e, &test.Cancels)
	feedOrders(t, &e, curOrderID, &test.Orders2)

	assert.Equal(t, len(test.Expected), len(executions), "incorrect number of executions")

	// Assert executions.
	for i := 0; i < len(test.Expected); i += 2 {
		expected1 := &test.Expected[i]
		expected2 := &test.Expected[i+1]
		actual1 := &executions[i]
		actual2 := &executions[i+1]

		match1 := compare(expected1, actual1) && compare(expected2, actual2)
		match2 := compare(expected1, actual2) && compare(expected2, actual1)

		assert.True(t, match1 || match2, `executions #%v and #%v,
	     %v,
	     %v
	     should have been
	     %v,
	     %v.
	     Stopped there.`, i, i+1, actual1, actual2, expected1, expected2)
	}
}

func feedOrders(t *testing.T, e *Engine, curOrderID OrderID, orders *[]Order) OrderID {
	if orders != nil {
		for i, order := range *orders {
			id := e.Limit(order)
			t.Logf("-> submitted order #%v: %v", id, &order)
			curOrderID++
			assert.Equal(t, id, curOrderID, "orderid returned was %v, should have been %v.", id, i+1)
		}
	}

	return curOrderID
}

func feedCancels(t *testing.T, e *Engine, cancels *[]OrderID) {
	if cancels != nil {
		for _, orderID := range *cancels {
			e.Cancel(orderID)
			t.Logf("-> cancelled #%v", orderID)
		}
	}
}

func compare(a, b *Execution) bool {
	return a.symbol == b.symbol &&
		a.trader == b.trader &&
		a.side == b.side &&
		a.price == b.price &&
		a.size == b.size
}
