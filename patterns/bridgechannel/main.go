package main

import (
	"fmt"
)

var orDone = func(done, c <-chan interface{}) <-chan interface{} {
	valStream := make(chan interface{})
	go func() {
		defer close(valStream)
		for {
			select {
			case <-done:
				fmt.Println("finish")
				return
			case v, ok := <-c:
				if ok == false {
					return
				}
				select {
				case valStream <- v:
				case <-done:
				}
			}
		}
	}()

	return valStream
}

func main() {

	bridge := func(done <-chan interface{}, chanStream <-chan <-chan interface{}) <-chan interface{} {
		valStream := make(chan interface{})
		go func() {
			defer close(valStream)
			// This loop is responsible for pulling channels off of chanStream
			// and providing them to a nested loop for use
			for {
				var stream <-chan interface{}
				select {
				case maybeStream, ok := <-chanStream:
					if ok == false {
						return
					}
					stream = maybeStream

				case <-done:
					return
				}

				for val := range orDone(done, stream) {
					select {
					case valStream <- val:
					case <-done:
					}
				}
			}
		}()

		return valStream
	}

	// Here’s an example that creates a series of 10 channels,
	// each them one element written to them, and passes the channels into bridge function
	genVals := func() <-chan <-chan interface{} {
		chanStream := make(chan (<-chan interface{}))
		go func() {
			defer close(chanStream)
			for i := 0; i < 10; i++ {
				stream := make(chan interface{}, 1)
				stream <- i
				close(stream)
				chanStream <- stream
			}
		}()
		return chanStream
	}

	done := make(chan interface{})
	defer close(done)

	for v := range bridge(done, genVals()) {
		fmt.Printf("%v ", v)
	}

}