package main

import (
	"fmt"
	"time"
)

func main() {
	chanOwner := func() <-chan int {
		results := make(chan int, 5)
		go func() {
			defer close(results)

			for i := 0; i <= 5; i++ {
				time.Sleep(1 * time.Second)
				results <- i
			}
		}()

		return results
	}

	consumer := func(results <-chan int) {
		fmt.Println(results)

		for result := range results {
			fmt.Printf("Received: %d\n", result)
		}

		fmt.Println("Done receiving!")
	}

	fmt.Println("initiate channel")
	results := chanOwner()

	fmt.Println("consumer ready - OK")
	consumer(results)
}