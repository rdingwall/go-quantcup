package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	maxExecs uint = 100
)

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

	orderID OrderID

	execsOut    [maxExecs]Execution
	execsOutLen uint
)

func TestAsk(t *testing.T) {
	test(t, []Order{oa101x100}, []Execution{})
}

func TestBid(t *testing.T) {
	test(t, []Order{ob101x100}, []Execution{})
}

func TestExecution(t *testing.T) {
	test(t, []Order{oa101x100, ob101x100}, []Execution{xa101x100, xb101x100})
}

func TestReordering1(t *testing.T) {
	test(t, []Order{oa101x100, ob101x100}, []Execution{xb101x100, xa101x100})
}

func TestReordering2(t *testing.T) {
	test(t, []Order{ob101x100, oa101x100}, []Execution{xa101x100, xb101x100})
}

func TestReordering3(t *testing.T) {
	test(t, []Order{ob101x100, oa101x100}, []Execution{xb101x100, xa101x100})
}

func TestPartialFill1(t *testing.T) {
	test(t, []Order{oa101x100, ob101x50}, []Execution{xa101x50, xb101x50})
}

func TestPartialFill2(t *testing.T) {
	test(t, []Order{oa101x50, ob101x100}, []Execution{xa101x50, xb101x50})
}

func TestIncrementalOverFill1(t *testing.T) {
	test(t, []Order{oa101x100, ob101x25, ob101x25, ob101x25, ob101x25, ob101x25}, []Execution{xa101x25, xb101x25, xa101x25, xb101x25, xa101x25, xb101x25, xa101x25, xb101x25})
}

func TestIncrementalOverFill2(t *testing.T) {
	test(t, []Order{ob101x100, oa101x25, oa101x25, oa101x25, oa101x25, oa101x25}, []Execution{xa101x25, xb101x25, xa101x25, xb101x25, xa101x25, xb101x25, xa101x25, xb101x25})
}

func TestQueuePosition(t *testing.T) {
	test(t, []Order{ob101x25x, ob101x25, oa101x25}, []Execution{xa101x25, xb101x25x})
}

func TestCancel(t *testing.T) {
	test_cancel(t, []Order{ob101x25}, []OrderID{1}, []Order{oa101x25}, []Execution{})
}

func TestCancelFromFrontOfQueue(t *testing.T) {
	test_cancel(t, []Order{ob101x25x, ob101x25}, []OrderID{1}, []Order{oa101x25}, []Execution{xa101x25, xb101x25})
}

func TestCancelFrontBackOutOfOrderThenPartialExecution(t *testing.T) {
	test_cancel(t, []Order{ob101x100, ob101x25x, ob101x25x, ob101x50}, []OrderID{1, 4, 3}, []Order{oa101x50}, []Execution{xb101x25x, xa101x25})
}

func test(t *testing.T, orders []Order, execs []Execution) {
	setGlobals()
	Init()

	executionCallback = func(exec Execution) {
		t.Logf("received execution: %v", &exec)
		assert.False(t, execsOutLen > maxExecs, "too many executions, test array overflow")
		execsOut[execsOutLen] = exec
		execsOutLen++
	}

	feedOrders(t, orders)
	assertExecCount(t, len(execs))
	assertExecs(t, execs)

	destroy()
}

func test_cancel(t *testing.T, orders1 []Order, cancels []OrderID, orders2 []Order, execs []Execution) {
	setGlobals()
	Init()

	executionCallback = func(exec Execution) {
		t.Logf("received execution: %v", &exec)
		assert.False(t, execsOutLen > maxExecs, "too many executions, test array overflow")
		execsOut[execsOutLen] = exec
		execsOutLen++
	}

	feedOrders(t, orders1)
	feedCancels(t, cancels)
	feedOrders(t, orders2)
	assertExecCount(t, len(execs))
	assertExecs(t, execs)

	destroy()
}

func feedOrders(t *testing.T, orders []Order) {
	for i := range orders {
		var order Order = orders[i]
		id := limit(order)
		t.Logf("submitted order #%v: %v", id, &order)
		orderID++
		assert.Equal(t, id, orderID, "orderid returned was %v, should have been %v.", id, i+1)
	}
}

func feedCancels(t *testing.T, cancels []OrderID) {
	for _, orderID := range cancels {
		cancel(orderID)
		t.Logf("cancelled #%v", orderID)
	}
}

func setGlobals() {
	orderID = 0
	execsOutLen = 0
}

func assertExecCount(t *testing.T, numExecsExpected int) {
	assert.Equal(t, numExecsExpected, execsOutLen, "execution called %v times, should have been %v.", execsOutLen, numExecsExpected)
}

func assertExecs(t *testing.T, execs []Execution) {
	for i := 0; i < len(execs); i += 2 {

		var expected1 *Execution = &execs[i]
		var expected2 *Execution = &execs[i+1]

		var actual1 *Execution = &execsOut[i]
		var actual2 *Execution = &execsOut[i+1]

		var match1 bool = execEq(expected1, actual1) && execEq(expected2, actual2)
		var match2 bool = execEq(expected1, actual2) && execEq(expected2, actual1)

		assert.True(t, match1 || match2, `executions #%v and #%v,
	     %v,
	     %v
	     should have been
	     %v,
	     %v.
	     Stopped there.`,
			i, i+1,
			actual1,
			actual2,
			expected1,
			expected2)
	}
}

func execEq(e1 *Execution, e2 *Execution) bool {
	return e1.symbol == e2.symbol &&
		e1.trader == e2.trader &&
		e1.side == e2.side &&
		e1.price == e2.price &&
		e1.size == e2.size
}
