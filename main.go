package main

import (
	"fmt"
	"math"
	"time"
)

const (
	msgBatchSize int = 10
	replays          = 200
)

func main() {
	executionCallback = func(exec Execution) {}

	samples := replays * (len(rawFeed) / msgBatchSize)
	late := make([]time.Duration, samples) // batch latency measurements

	for j := 0; j < replays; j++ {
		Init()

		for i := msgBatchSize; i < len(rawFeed); i += msgBatchSize {
			begin := time.Now()
			feed(i-msgBatchSize, i)
			end := time.Now()

			late[i/msgBatchSize-1+(j*(len(rawFeed)/msgBatchSize))] = end.Sub(begin)
		}

		destroy()
	}

	var lateTotal int64 = 0

	for i := 0; i < samples; i++ {
		lateTotal += int64(late[i])
	}

	var lateMean float32 = float32(lateTotal) / float32(samples)
	var lateCentered float32 = 0
	var lateSqTotal float64 = 0
	for i := 0; i < samples; i++ {
		lateCentered = float32(late[i]) - lateMean
		lateSqTotal += float64(lateCentered * lateCentered / float32(samples))
	}
	var lateSd float32 = float32(math.Sqrt(lateSqTotal))
	fmt.Printf("mean(latency) = %v, sd(latency) = %v\n", lateMean, lateSd)

	var score float32 = 0.5 * (lateMean + lateSd)
	fmt.Printf("You scored %v. Try to minimize this.\n", score)
}

func feed(begin int, end int) {
	for i := begin; i < end; i++ {
		var order Order = rawFeed[i]
		if rawFeed[i].price == 0 {
			orderID := OrderID(order.size)
			cancel(orderID)
		} else {
			limit(order)
		}
	}
}
