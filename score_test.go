package main

import (
	"testing"
)

const (
	msgBatchSize int = 10
)

func BenchmarkScore(b *testing.B) {
	executionCallback = func(exec Execution) {}
	Init()
	b.ResetTimer()
	//var samples int = replays * (len(rawFeed) / msgBatchSize)
	for i := 0; i < b.N; i++ {
		feed(i*(msgBatchSize-1), i*msgBatchSize)
	}
	destroy()
}

func feed(begin int, end int) {
	for i := begin; i < end; i++ {
		if rawFeed[i].price == 0 {
			cancel(OrderID(rawFeed[i].size))
		} else {
			limit(rawFeed[i])
		}
	}
}
