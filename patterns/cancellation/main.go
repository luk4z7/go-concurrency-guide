package main

import (
	"fmt"
	"time"
)

func main() {

	doWork := func(done <-chan interface{}, strings <-chan string) <-chan interface{} {
		terminated := make(chan interface{})

		go func() {
			// será executado o defer quando for feito close de done
			// para que a goroutine seja finalizada
			defer fmt.Println("doWork exited.")
			defer close(terminated)

			for {
				select {
				case s := <-strings:
					// Do something interesting
					fmt.Println(s)
				case <-done:
					// return vai finalizar a goroutine
					return
				}
			}
		}()

		fmt.Println("doWork initiate ...")

		// somente é retornado depois do close, enquanto estiver vazio não será enviado
		return terminated
	}

	done := make(chan interface{})
	terminated := doWork(done, nil)

	go func() {
		// Cancel the operation after 1 second.
		time.Sleep(5 * time.Second)
		fmt.Println("Canceling doWork goroutine...")
		// if it has no cancel produces a deadlock
		close(done)
	}()

	fmt.Println("initiate ...")
	d := <-terminated
	fmt.Println(d)
	fmt.Println("Done.")
}