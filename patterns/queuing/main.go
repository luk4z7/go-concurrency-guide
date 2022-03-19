package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

func main() {
	var wg sync.WaitGroup
	limit := make(chan interface{}, runtime.NumCPU())

	fmt.Printf("Started, Limit %d\n", cap(limit))

	workers := func(l chan<- interface{}, wg *sync.WaitGroup) {
		for i := 0; i <= 50; i++ {
			i := i

			limit <- struct{}{}
			wg.Add(1)

			go func(x int, w *sync.WaitGroup) {
				defer w.Done()

				time.Sleep(1 * time.Second)
				fmt.Printf("Process %d\n", i)

				<-limit
			}(i, wg)
		}
	}

	workers(limit, &wg)
	wg.Wait()

	fmt.Println("Finished")
}