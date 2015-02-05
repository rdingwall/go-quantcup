package main

import (
	"fmt"
	"time"

	"github.com/grd/stat"
)

const (
	batchSize   int = 10
	replayCount     = 200
)

func main() {

	var e Engine

	// batch latency measurements.
	latencies := make([]time.Duration, replayCount*(len(ordersFeed)/batchSize))

	for j := 0; j < replayCount; j++ {
		e.Reset()
		for i := batchSize; i < len(ordersFeed); i += batchSize {
			begin := time.Now()
			feed(&e, i-batchSize, i)
			end := time.Now()
			latencies[i/batchSize-1+(j*(len(ordersFeed)/batchSize))] = end.Sub(begin)
		}
	}

	data := DurationSlice(latencies)

	var mean float64 = stat.Mean(data)
	var stdDev = stat.SdMean(data, mean)
	var score = 0.5 * (mean + stdDev)

	fmt.Printf("mean(latency) = %1.2f, sd(latency) = %1.2f\n", mean, stdDev)
	fmt.Printf("You scored %1.2f. Try to minimize this.\n", score)
}

func feed(e *Engine, begin, end int) {
	for i := begin; i < end; i++ {
		var order Order = ordersFeed[i]
		if order.price == 0 {
			orderID := OrderID(order.size)
			e.Cancel(orderID)
		} else {
			e.Limit(order)
		}
	}
}

type DurationSlice []time.Duration

func (f DurationSlice) Get(i int) float64 { return float64(f[i]) }
func (f DurationSlice) Len() int          { return len(f) }
