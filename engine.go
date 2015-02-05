/*****************************************************************************
 *                QuantCup 1:   Price-Time Matching Engine
 *
 * Submitted by: voyager (golang port by rdingwall@gmail.com)
 *
 * Design Overview:
 *   In this implementation, the limit order book is represented using
 *   a flat linear array (pricePoints), indexed by the numeric price value.
 *   Each entry in this array corresponds to a specific price point and holds
 *   an instance of struct pricePoint. This data structure maintains a list
 *   of outstanding buy/sell orders at the respective price. Each outstanding
 *   limit order is represented by an instance of struct orderBookEntry.
 *
 *   askMin and bidMax are global variables that maintain starting points,
 *   at which the matching algorithm initiates its search.
 *   askMin holds the lowest price that contains at least one outstanding
 *   sell order. Analogously, bidMax represents the maximum price point that
 *   contains at least one outstanding buy order.
 *
 *   When a Buy order arrives, we search the book for outstanding Sell orders
 *   that cross with the incoming order. We start the search at askMin and
 *   proceed upwards, incrementing askMin until:
 *     a) The incoming Buy order is filled.
 *     b) We reach a price point that no longer crosses with the incoming
 *        limit price (askMin > BuyOrder.price)
 *     In case b), we create a new orderBookEntry to record the
 *     remainder of the incoming Buy order and add it to the global order
 *     book by appending it to the list at pricePoints[BuyOrder.price].
 *
 *  Incoming Sell orders are handled analogously, except that we start at
 *  bidMax and proceed downwards.
 *
 *  Although this matching algorithm runs in linear time and may, in
 *  degenerate cases, require scanning a large number of array slots,
 *  it appears to work reasonably well in practice, at least on the
 *  simulated data feed (score_feed.h). The vast majority of incoming
 *  limit orders can be handled by examining no more than two distinct
 *  price points and no order requires examining more than five price points.
 *
 *  To avoid incurring the costs of dynamic heap-based memory allocation,
 *  this implementation maintains the full set of orderBookEntry instances
 *  in a statically-allocated contiguous memory arena (arenaBookEntries).
 *  Allocating a new entry is simply a matter of bumping up the orderID
 *  counter (curOrderID) and returning a pointer to arenaBookEntries[curOrderID].
 *
 *  To cancel an order, we simply set its size to zero. Notably, we avoid
 *  unhooking its orderBookEntry from the list of active orders in order to
 *  avoid incurring the costs of pointer manipulation and conditional branches.
 *  This allows us to handle order cancellation requests very efficiently; the
 *  current implementation requires only one memory store instruction on
 *  x86_64. During order matching, when we walk the list of outstanding orders,
 *  we simply skip these zero-sized entries.
 *
 *  The current implementation uses a custom version of strcpy() to copy the string
 *  fields ("symbol" and "trader") between data structures. This custom version
 *  has been optimized for the case STRINGLEN=5 and implements loop unrolling
 *  to eliminate the use of induction variables and conditional branching.
 *
 *  The memory layout of struct orderBookEntry has been optimized for
 *  efficient cache access.
 *****************************************************************************/

package main

type (
	// struct orderBookEntry: describes a single outstanding limit order (Buy or Sell).
	orderBookEntry struct {
		size   Size
		next   *orderBookEntry
		trader string
	}

	// struct pricePoint: describes a single price point in the limit order book.
	pricePoint struct {
		listHead *orderBookEntry
		listTail *orderBookEntry
	}
)

const maxNumOrders uint = 1010000

// Global state
var (
	// An array of pricePoint structures representing the entire limit order book
	pricePoints [uint(maxPrice) + 1]pricePoint

	curOrderID OrderID // Monotonically-increasing orderID
	askMin     uint    // Minimum Ask price
	bidMax     uint    // Maximum Bid price

	// Statically-allocated memory arena for order book entries. This data structure allows us to avoid the overhead of heap-based memory allocation.
	arenaBookEntries [maxNumOrders]orderBookEntry

	executionCallback func(exec Execution)
)

func Init() {
	for i := range pricePoints {
		pricePoints[i].listHead = nil
		pricePoints[i].listTail = nil
	}

	for i := range arenaBookEntries {
		arenaBookEntries[i].size = 0
		arenaBookEntries[i].next = nil
		arenaBookEntries[i].trader = ""
	}

	curOrderID = 0
	askMin = uint(maxPrice) + 1
	bidMax = uint(minPrice) - 1
}

func destroy() {
}

// Process an incoming limit order
func limit(order Order) OrderID {
	var bookEntry *orderBookEntry
	var entry *orderBookEntry
	var ppEntry *pricePoint

	var price Price = order.price
	var orderSize Size = order.size

	if order.side == Bid { // Buy order
		// Look for outstanding sell orders that cross with the incoming order
		if uint(price) >= askMin {
			ppEntry = &pricePoints[askMin]

			for {
				bookEntry = ppEntry.listHead

				for bookEntry != nil {
					if bookEntry.size < orderSize {
						executeTrade(order.symbol, order.trader, bookEntry.trader, price, bookEntry.size)

						orderSize -= bookEntry.size
						bookEntry = bookEntry.next
					} else {
						executeTrade(order.symbol, order.trader, bookEntry.trader, price, orderSize)

						if bookEntry.size > orderSize {
							bookEntry.size -= orderSize
						} else {
							bookEntry = bookEntry.next
						}

						ppEntry.listHead = bookEntry
						curOrderID++
						return curOrderID
					}
				}

				// We have exhausted all orders at the askMin price point. Move on to the next price level.
				ppEntry.listHead = nil
				askMin++
				ppEntry = &pricePoints[askMin]

				if uint(price) < askMin {
					break
				}
			}
		}

		curOrderID++
		entry = &arenaBookEntries[curOrderID]
		entry.size = orderSize
		entry.trader = order.trader
		ppInsertOrder(&pricePoints[price], entry)

		if bidMax < uint(price) {
			bidMax = uint(price)
		}

		return curOrderID
	} else { // Sell order
		// Look for outstanding Buy orders that cross with the incoming order
		if uint(price) <= bidMax {
			ppEntry = &pricePoints[bidMax]

			for {
				bookEntry = ppEntry.listHead

				for bookEntry != nil {
					if bookEntry.size < orderSize {
						executeTrade(order.symbol, bookEntry.trader, order.trader, price, bookEntry.size)

						orderSize -= bookEntry.size
						bookEntry = bookEntry.next
					} else {
						executeTrade(order.symbol, bookEntry.trader, order.trader, price, orderSize)

						if bookEntry.size > orderSize {
							bookEntry.size -= orderSize
						} else {
							bookEntry = bookEntry.next
						}

						ppEntry.listHead = bookEntry
						curOrderID++
						return curOrderID
					}
				}

				// We have exhausted all orders at the bidMax price point. Move on to the next price level.
				ppEntry.listHead = nil
				bidMax--
				ppEntry = &pricePoints[bidMax]

				if uint(price) > bidMax {
					break
				}
			}
		}

		curOrderID++
		entry = &arenaBookEntries[curOrderID]
		entry.size = orderSize
		entry.trader = order.trader
		ppInsertOrder(&pricePoints[price], entry)

		if askMin > uint(price) {
			askMin = uint(price)
		}

		return curOrderID
	}
}

func cancel(orderID OrderID) {
	arenaBookEntries[orderID].size = 0
}

// Report trade execution
func executeTrade(symbol string, buyTrader string, sellTrader string, tradePrice Price, tradeSize Size) {
	if tradeSize == 0 { // Skip orders that have been cancelled
		return
	}

	var exec Execution = Execution{symbol: symbol, price: tradePrice, size: tradeSize}

	exec.side = Bid
	exec.trader = buyTrader
	executionCallback(exec) // Report the buy-side trade

	exec.side = Ask
	exec.trader = sellTrader
	executionCallback(exec) // Report the sell-side trade
}

// Insert a new order book entry at the tail of the price point list
func ppInsertOrder(ppEntry *pricePoint, entry *orderBookEntry) {
	if ppEntry.listHead != nil {
		ppEntry.listTail.next = entry
	} else {
		ppEntry.listHead = entry
	}
	ppEntry.listTail = entry
}
