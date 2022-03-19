package main

import (
	"fmt"
)

type data int

// distribute work items to multiple uniform actors
// no data shall be processed twice!
// received wch
// response res
func worker(wch <-chan data, res chan<- data) {
	for {
		w, ok := <-wch
		if !ok {
			return // return when is closed
		}

		w *= 2
		res <- w
	}
}

func main() {
	work := []data{1, 2, 3, 4, 5}

	const numWorkers = 3

	wch := make(chan data, len(work))
	res := make(chan data, len(work))

	// fan-out, one input channel for all actors
	for i := 0; i < numWorkers; i++ {
		go worker(wch, res)
	}

	// fan-out, one input channel for all actors
	for _, w := range work {
		fmt.Println("send to wch : ", w)
		wch <- w
	}
	close(wch)

	// fan-in, one result channel
	for range work {
		w := <-res
		fmt.Println("receive from res : ", w)
	}
}