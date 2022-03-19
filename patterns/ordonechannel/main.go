package main

import (
	"fmt"
)

func main() {
	orDone := func(done, c <-chan interface{}) <-chan interface{} {
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

	done := make(chan interface{})
	myChan := make(chan interface{})
	go func() {
		done <- struct{}{}
	}()

	for val := range orDone(done, myChan) {
		// Do something with val
		fmt.Println(val)
	}
}