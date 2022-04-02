package main

import (
    "fmt"
    "sync"
    "time"
)

// a goroutine that is waiting for a signal, and a goroutine that is sending signals.
// Say we have a queue of fixed length 2, and 10 items we want to push onto the queue
func main() {
    c := sync.NewCond(&sync.Mutex{})
    queue := make([]interface{}, 0, 10)

    removeFromQueue := func(delay time.Duration) {
        time.Sleep(delay)
        c.L.Lock() // critical section

        queue = queue[1:]

        fmt.Println("Removed from queue")

        c.L.Unlock()
        c.Signal() // let a goroutine waiting on the condition know that something has ocurred
    }

    for i := 0; i < 10; i++ {
        c.L.Lock() // critical section

        // When the queue is equal to two the main goroutine is suspend
        // until a signal on the condition has been sent
        for len(queue) == 2 {
            c.Wait()
        }

        fmt.Println("Adding to queue")
        queue = append(queue, struct{}{})

        go removeFromQueue(1 * time.Second)

        c.L.Unlock()
    }
}