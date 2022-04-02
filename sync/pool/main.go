package main

import (
    "fmt"
    "sync"
)

func main() {
    myPool := &sync.Pool{
        New: func() interface{} {
            fmt.Println("Creating new instance.")
            return struct{}{}
        },
    }

    // Get invokes New function defined in the pool if there is no instance started
    myPool.Get()
    instance := myPool.Get()
    fmt.Println("instance", instance)

    // here we put a previously retrieved instance back in the pool, this
    // increases the number of instances available for a
    myPool.Put(instance)
    // when this call is executed, we will reuse the previously allocated instance
    // and put it back in the pool
    myPool.Get()

    var numCalcsCreated int
    calcPool := &sync.Pool{
        New: func() interface{} {
            // fmt.Println("new calc pool")

            numCalcsCreated += 1
            mem := make([]byte, 1024)
            return &mem
        },
    }

    fmt.Println("calcPool.New", calcPool.New())

    calcPool.Put(calcPool.New())
    calcPool.Put(calcPool.New())
    calcPool.Put(calcPool.New())
    calcPool.Put(calcPool.New())

    calcPool.Get()

    const numWorkers = 1024 * 1024
    var wg sync.WaitGroup
    wg.Add(numWorkers)
    for i := numWorkers; i > 0; i-- {
        go func() {
            defer wg.Done()

            mem := calcPool.Get().(*[]byte)
            defer calcPool.Put(mem)

            // Assume something interesting, but quick is being done with
            // this memory.
        }()
    }

    wg.Wait()
    fmt.Printf("%d calculators were created.", numCalcsCreated)
}