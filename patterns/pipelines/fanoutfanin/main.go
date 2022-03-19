package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

var toString = func(done <-chan interface{}, valueStream <-chan interface{}) <-chan string {
	stringStream := make(chan string)

	go func() {
		defer close(stringStream)

		for v := range valueStream {
			select {
			case <-done:
				return
			case stringStream <- v.(string):
			}
		}
	}()

	return stringStream
}

var toInt = func(done <-chan interface{}, valueStream <-chan interface{}) <-chan int {
	intStream := make(chan int)

	go func() {
		defer close(intStream)

		for v := range valueStream {
			select {
			case <-done:
				return
			case intStream <- v.(int):
			}
		}
	}()

	return intStream
}

var repeat = func(done <-chan interface{}, values ...interface{}) <-chan interface{} {
	valueStream := make(chan interface{})

	go func() {
		defer close(valueStream)

		for {
			for _, v := range values {
				select {
				case <-done:
					return
				case valueStream <- v:
				}
			}
		}
	}()

	return valueStream
}

// This function will repeat the values you pass to it infinitely until you tell it to stop.
// Let’s take a look at another generic pipeline stage that is helpful when used in combination with repeat, take:
// With the take stage, the concern is limiting our pipeline.
var take = func(done <-chan interface{}, valueStream <-chan interface{}, num int) <-chan interface{} {
	takeStream := make(chan interface{})

	go func() {
		defer close(takeStream)

		for i := 0; i < num; i++ {
			select {
			case <-done:
				return
			case takeStream <- <-valueStream:
			}
		}
	}()

	return takeStream
}

// In the repeat and repeatFn generators, the concern is generating a stream of data by looping over a list or operator.
var repeatFn = func(done <-chan interface{}, fn func() interface{}) <-chan interface{} {
	valueStream := make(chan interface{})

	go func() {
		defer close(valueStream)

		for {
			select {
			case <-done:
				return
			case valueStream <- fn():
			}
		}
	}()

	return valueStream
}

// 1º
// Here we take in our standard done channel to allow our goroutines to be torn down,
// and then a variadic slice of interface{} channels to fan-in.
var fanIn = func(done <-chan interface{}, channels ...<-chan interface{}) <-chan interface{} {

	// 2º
	// On this line we create a sync.WaitGroup so that we can wait until all channels have been drained.
	var wg sync.WaitGroup
	multiplexedStream := make(chan interface{})

	// 3º
	// Here we create a function, multiplex, which, when passed a channel,
	// will read from the channel, and pass the value read onto the multiplexedStream channel.
	multiplex := func(c <-chan interface{}) {
		defer wg.Done()
		for i := range c {
			select {
			case <-done:
				return
			case multiplexedStream <- i:
			}
		}
	}

	// 4º
	// This line increments the sync.WaitGroup by the number of channels we’re multiplexing.
	// Select from all the channels
	wg.Add(len(channels))
	for _, c := range channels {
		go multiplex(c)
	}

	// 5º
	// Here we create a goroutine to wait for all the channels we’re multiplexing
	// to be drained so that we can close the multiplexedStream channel.
	// Wait for all the reads to complete
	go func() {
		wg.Wait()
		close(multiplexedStream)
	}()

	return multiplexedStream
}

var primeFinder = func(done <-chan interface{}, valueStream <-chan int) <-chan interface{} {
	intStream := make(chan interface{})

	go func() {
		defer close(intStream)

		for {
			select {
			case <-done:
				return
			case intStream <- <-valueStream:
			}
		}
	}()

	return intStream
}

// Without Fan-in and Fan-out
func v1() {
	fmt.Println("V1")
	rand := func() interface{} { return rand.Intn(50000000) }

	done := make(chan interface{})
	defer close(done)

	start := time.Now()

	randIntStream := toInt(done, repeatFn(done, rand))
	fmt.Println("Primes:")
	for prime := range take(done, primeFinder(done, randIntStream), 10) {
		fmt.Printf("\t%d\n", prime)
	}

	fmt.Printf("Search took: %v\n", time.Since(start))
}

// Fan In
func v2() {
	fmt.Println("V2")
	done := make(chan interface{})
	defer close(done)

	start := time.Now()

	rand := func() interface{} { return rand.Intn(50000000) }

	randIntStream := toInt(done, repeatFn(done, rand))

	// now that we have four goroutines, we also have four channels,
	// but our range over primes is only expecting one channel.
	// This brings us to the fan-in portion of the pattern.
	numFinders := runtime.NumCPU()
	fmt.Printf("Spinning up %d prime finders.\n", numFinders)

	finders := make([]<-chan interface{}, numFinders)
	fmt.Println("Primes:")
	for i := 0; i < numFinders; i++ {
		finders[i] = primeFinder(done, randIntStream)
	}

	// As we discussed earlier, fanning in means multiplexing or joining
	// together multiple streams of data into a single stream. The algorithm to do so is relatively simple:
	// That's fanIn functions make, receive the many channels of "finders..." and return only one stream
	for prime := range take(done, fanIn(done, finders...), 10) {
		fmt.Printf("\t%d\n", prime)
	}

	fmt.Printf("Search took: %v\n", time.Since(start))

}

func main() {
	v1()
	v2()
}