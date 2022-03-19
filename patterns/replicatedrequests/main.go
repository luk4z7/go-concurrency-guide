package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

func main() {

	doWork := func(done <-chan interface{}, id int, wg *sync.WaitGroup, result chan<- int) {
		started := time.Now()
		defer wg.Done()

		// Simulate random load
		simulatedLoadTime := time.Duration(1+rand.Intn(5)) * time.Second
		select {
		case <-done:
		case <-time.After(simulatedLoadTime):
		}

		select {
		case <-done:
		case result <- id:
		}

		took := time.Since(started)
		// Display how long handlers would have taken
		if took < simulatedLoadTime {
			took = simulatedLoadTime

		}

		fmt.Printf("%v took %v\n", id, took)
	}

	done := make(chan interface{})
	result := make(chan int)

	var wg sync.WaitGroup
	wg.Add(10)

	// Here we start 10 handlers to handle our requests.
	for i := 0; i < 10; i++ {
		go doWork(done, i, &wg, result)
	}

	// This line grabs the first returned value from the group of handlers.
	firstReturned := <-result

	// Here we cancel all the remaining handlers.
	// This ensures they don’t continue to do unnecessary work.
	close(done)
	wg.Wait()

	fmt.Printf("Received an answer from #%v\n", firstReturned)
}