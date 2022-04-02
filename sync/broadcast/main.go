package main

import (
    "fmt"
    "sync"
)

type Button struct {
    Clicked *sync.Cond
}

func main() {
    button := Button{
        Clicked: sync.NewCond(&sync.Mutex{}),
    }

    // running on goroutine every function that passed/registered
    // and wait, not exit until that goroutine is confirmed to be running
    subscribe := func(c *sync.Cond, param string, fn func(s string)) {
        var goroutineRunning sync.WaitGroup
        goroutineRunning.Add(1)

        go func(p string) {
            goroutineRunning.Done()
            c.L.Lock() // critical section
            defer c.L.Unlock()

            // fmt.Println("Registered and wait ... ")
            c.Wait()

            fn(p)
        }(param)

        goroutineRunning.Wait()
    }

    var clickRegistered sync.WaitGroup

	for _, v := range []string{
        "Maximizing window.",
        "Displaying annoying dialog box!",
        "Mouse clicked."} {

        clickRegistered.Add(1)

        subscribe(button.Clicked, v, func(s string) {
            fmt.Println(s)
            clickRegistered.Done()
        })
    }

    button.Clicked.Broadcast()

    clickRegistered.Wait()
}