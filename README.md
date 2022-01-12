# Go Concurrency Guide

This guide is built on top of the some examples of the book `Go Concurrency in Go` and `Go Programming Language`

- [Race Condition and Data Race](#race-condition-and-data-race)
- [Memory Access Synchonization](#memory-access-synchonization)
    -  [Mutex](#mutex)
    -  [waitgroup](#waitgroup)
    -  [RWMutex](#rwmutex)
    -  [Cond](#cond)
    -  [Pool](#pool)
- [Deadlocks, Livelocks and Starvation](#deadlocks-livelocks-and-starvation)
    -  [Deadlocks](#deadlocks)
    -  [Livelocks](#livelocks)
    -  [Starvation](#starvation)
- [Channels](#channels)
- [Patterns](#patterns)
    - [Confinement](#confinement)
    - [Cancellation](#cancellation)
    - [OR Channel](#or-channel)
    - [OR Done Channel](#or-done-channel)
    - [Error Handling](#error-handling)
    - [Pipelines](#pipelines)
    - [Fan In](#fan-in)
    - [Fan Out](#fan-out)
- [References](#references)


## Race Condition and Data Race

Race condition occur when two or more operations must execute in the correct order, but the program has not been written so that this order is guaranteed  to maintained.

Data race is when one concurrent operation attempts to read a variable while at some undetermined time another concurrent operation is attempting to write to the same variable, the main func is the main goroutine.

```go
func main() {
    var data int
    go func() {
        data++
    }()

    if data == 0 {
        fmt.Println("the value is %d", data)
    }
}
```

## Memory Access Synchonization

The sync package contains the concurrency primitives that are most useful for low-level memory access synchronization.
Critical section is the place in your code that has access to a shared memory

### Mutex

Mutex stands for “mutual exclusion” and is a way to protect critical sections of your program.

```go
type Counter struct {
    mu sync.Mutex
    value int
}

func (c *Counter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.value++
}
```

### WaitGroup

Call to add a group of goroutines

```go
var wg sync.WaitGroup
for _, salutation := range []string{"hello", "greetings", "good day"} {
    wg.Add(1)
    go func(salutation string) { 
        defer wg.Done()
        fmt.Println(salutation)
    }(salutation) 
}
wg.Wait()
```

### RWMutex

More fine-grained memory control, being possible to request read-only lock

```go
producer := func(wg *sync.WaitGroup, l sync.Locker) { 
    defer wg.Done()
    for i := 5; i > 0; i-- {
        l.Lock()
        l.Unlock()
        time.Sleep(1) 
    }
}

observer := func(wg *sync.WaitGroup, l sync.Locker) {
    defer wg.Done()
    l.Lock()
    defer l.Unlock()
}

test := func(count int, mutex, rwMutex sync.Locker) time.Duration {
    var wg sync.WaitGroup
    wg.Add(count+1)
    beginTestTime := time.Now()
    go producer(&wg, mutex)
    for i := count; i > 0; i-- {
        go observer(&wg, rwMutex)
    }

    wg.Wait()
    return time.Since(beginTestTime)
}

tw := tabwriter.NewWriter(os.Stdout, 0, 1, 2, ' ', 0)
defer tw.Flush()

var m sync.RWMutex
fmt.Fprintf(tw, "Readers\tRWMutext\tMutex\n")
for i := 0; i < 20; i++ {
    count := int(math.Pow(2, float64(i)))
    fmt.Fprintf(
        tw,
        "%d\t%v\t%v\n",
        count,
        test(count, &m, m.RLocker()),
        test(count, &m, &m),
    )
}
```

### Cond

It would be better if there were some kind of way for a goroutine to efficiently sleep until it was signaled to wake and check its condition. This is exactly what the Cond type does for us.

```go
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

            //          fmt.Println("Registered and wait ... ")
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
```

### Once

Ensuring that only one execution will be carried out even among several goroutines

```go
var count int

increment := func() {
    count++
}

var once sync.Once

var increments sync.WaitGroup
increments.Add(100)
for i := 0; i < 100; i++ {
    go func() {
        defer increments.Done()
        once.Do(increment)
    }()
}

increments.Wait()
fmt.Printf("Count is %d\n", count)
```

### Pool

Manager the pool of connections, a quantity

```go
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

    // Get invoca New function definida no pool caso não existir uma instância iniciada
    myPool.Get()
    instance := myPool.Get()
    fmt.Println("instance", instance)

    // aqui colocamos uma instancia previamente recuperada de volta no pool, isso
    // aumenta o número de instancias disponiveis para um
    myPool.Put(instance)
    // quando esta chamada for executada, iremos reutilizar a instancia previamente alocada
    // e coloca-la de volta no pool
    myPool.Get()

    var numCalcsCreated int
    calcPool := &sync.Pool{
        New: func() interface{} {
            //          fmt.Println("new calc pool")

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
```


## Deadlocks, Livelocks, and Starvation

### Deadlocks

Deadlocks is a program is one in which all concurrent processes are waiting on one another.

```go
package main

import (
    "fmt"
    "sync"
    "time"
)

type value struct {
    mu    sync.Mutex
    value int
}

func main() {
    var wg sync.WaitGroup
    printSum := func(v1, v2 *value) {
        defer wg.Done()
        v1.mu.Lock()
        defer v1.mu.Unlock()

        // deadlock
        time.Sleep(2 * time.Second)
        v2.mu.Lock()
        defer v2.mu.Unlock()

        fmt.Printf("sum=%v\n", v1.value+v2.value)
    }

    var a, b value
    wg.Add(2)
    go printSum(&a, &b)
    go printSum(&b, &a)

    wg.Wait()
}
```

```go
package main

func main() {
    message := make(chan string)

    // A goroutine ( main goroutine ) trying to send message to channel
    message <- "message" // fatal error: all goroutines are asleep - deadlock!
}
```

```go
package main

func main() {
    message := make(chan string)

    // A goroutine ( main goroutine ) trying to receive message from channel
    <-message // fatal error: all goroutines are asleep - deadlock!
}
```


### Livelocks

Livelocks are programs that are actively performing concurrent operations, but these operations do nothing to move the state of the program forward.

```go
ackage main

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	cadence := sync.NewCond(&sync.Mutex{})
	go func() {
		for range time.Tick(1 * time.Millisecond) {
			cadence.Broadcast()
		}
	}()

	takeStep := func() {
		cadence.L.Lock()
		cadence.Wait()
		cadence.L.Unlock()
	}

	tryDir := func(dirName string, dir *int32, out *bytes.Buffer) bool {
		fmt.Fprintf(out, " %v", dirName)
		atomic.AddInt32(dir, 1)
		takeStep()

		if atomic.LoadInt32(dir) == 1 {
			fmt.Fprint(out, " . Success!")
			return true
		}

		takeStep()
		atomic.AddInt32(dir, -1)
		return false
	}

	var left, right int32
	tryLeft := func(out *bytes.Buffer) bool {
		return tryDir("left", &left, out)
	}

	tryRight := func(out *bytes.Buffer) bool {
		return tryDir("right", &right, out)
	}

	walk := func(walking *sync.WaitGroup, name string) {
		var out bytes.Buffer
		defer func() {
			fmt.Println(out.String())
		}()
		defer walking.Done()

		fmt.Fprintf(&out, "%v is trying to scoot:", name)
		for i := 0; i < 5; i++ {
			if tryLeft(&out) || tryRight(&out) {
				return
			}
		}

		fmt.Fprintf(&out, "\n%v tosses her hands up in exasperation", name)
	}

	var peopleInHallway sync.WaitGroup
	peopleInHallway.Add(2)

	go walk(&peopleInHallway, "Alice")
	go walk(&peopleInHallway, "Barbara")
	peopleInHallway.Wait()

	fmt.Println("vim-go")
}
```

### Starvation

Starvation is any situation where a concurrent process cannot get all the resources it needs to perform work.

```go
package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	fmt.Println("vim-go")

	var wg sync.WaitGroup
	var sharedLock sync.Mutex
	const runtime = 1 * time.Second

	greedyWorker := func() {
		defer wg.Done()

		var count int
		for begin := time.Now(); time.Since(begin) <= runtime; {
			sharedLock.Lock()
			time.Sleep(3 * time.Nanosecond)
			sharedLock.Unlock()
			count++
		}

		fmt.Printf("Greedy worker was able to execute %v work loops\n", count)
	}

	politeWorker := func() {
		defer wg.Done()

		var count int
		for begin := time.Now(); time.Since(begin) <= runtime; {
			sharedLock.Lock()
			time.Sleep(1 * time.Nanosecond)
			sharedLock.Unlock()

			sharedLock.Lock()
			time.Sleep(1 * time.Nanosecond)
			sharedLock.Unlock()

			sharedLock.Lock()
			time.Sleep(1 * time.Nanosecond)
			sharedLock.Unlock()

			count++
		}

		fmt.Printf("Polite worker was able to execute %v work loops \n", count)
	}

	wg.Add(2)
	go greedyWorker()
	go politeWorker()
	wg.Wait()
}
```

## Channels

Channels are one of the synchronization primitives in Go derived from Hoare’s CSP. While they can be used to synchronize access of the memory, they are best used to communicate information between goroutines, default value for channel: nil.

To declare a channel to read and send
```go
stream := make(chan interface{})
```

To declare unidirectional channel that only can read
```go
stream := make(<-chan interface{})
```

To declare unidirectional channel that only can send
```go
stream := make(chan<- interface{})
```

is not often see the instantiates channels unidirectional, only in parameters in functions, is common because Go convert them implicity
```go
var receiveChan <-chan interface{}
var sendChan chan<- interface{}
dataStream := make(chan interface{})

// Valid statements:
receiveChan = dataStream
sendChan = dataStream
```

To receive
```go
<-stream
```

to send
```go
stream <- "Hello world"
```

Ranging o ver

the for range break the loop if the channel is closed

```go
intStream := make(chan int)
go func() {
    defer close(intStream) 
    for i := 1; i <= 5; i++ {
        intStream <- i
    }
}()

for integer := range intStream {
    fmt.Printf("%v ", integer)
}
```

buffered channel
both, read and write of a channel full or empty it will block, on the buffered channel

```go
var dataStream chan interface{}
dataStream = make(chan interface{}, 4)
```

both, read and send a channel empty cause deadlock

```go

var dataStream chan interface{}
<-dataStream This panics with: fatal error: all goroutines are asleep - deadlock!

  goroutine 1 [chan receive (nil chan)]:
  main.main()
      /tmp/babel-23079IVB/go-src-23079O4q.go:9 +0x3f
  exit status 2

var dataStream chan interface{}
dataStream <- struct{}{} This produces: fatal error: all goroutines are asleep - deadlock!

  goroutine 1 [chan send (nil chan)]:
  main.main()
      /tmp/babel-23079IVB/go-src-23079dnD.go:9 +0x77
  exit status 2
```

and a close channel cause a panic

```go
var dataStream chan interface{}
close(dataStream) This produces: panic: close of nil channel

  goroutine 1 [running]:
  panic(0x45b0c0, 0xc42000a160)
      /usr/local/lib/go/src/runtime/panic.go:500 +0x1a1
  main.main()
      /tmp/babel-23079IVB/go-src-230794uu.go:9 +0x2a
  exit status 2 Yipes! This is probably
```

## Patterns

### Confinement

Confinement is the simple yet powerful idea of ensuring information is only ever available from one concurrent process.
There are two kinds of confinement possible: ad hoc and lexical.

Ad hoc confinement is when you achieve confinement through a convention

```go
data := make([]int, 4)

loopData := func(handleData chan<- int) {
    defer close(handleData)
    for i := range data {
        handleData <- data[i]
    }
}

handleData := make(chan int)
go loopData(handleData)

for num := range handleData {
    fmt.Println(num)
}
```

Lexical

```go
chanOwner := func() <-chan int {
    results := make(chan int, 5) 
    go func() {
        defer close(results)

        for i := 0; i <= 5; i++ {
            results <- i
        }
    }()
    return results
}

consumer := func(results <-chan int) { 
    for result := range results {
        fmt.Printf("Received: %d\n", result)
    }
    fmt.Println("Done receiving!")
}

results := chanOwner()
consumer(results)
```

### Cancellation

```go
package main

func main() {
    doWork := func(
        done <-chan interface{},
        strings <-chan string,
    ) <-chan interface{} {
        terminated := make(chan interface{})
        go func() {
            defer fmt.Println("doWork exited.")
            defer close(terminated)
            for {

                select {
                case s := <-strings:
                    // Do something interesting
                    fmt.Println(s)
                case <-done:
                    return
                }
            }
        }()
        return terminated
    }

    done := make(chan interface{})
    terminated := doWork(done, nil)

    go func() {
        // Cancel the operation after 1 second.
        time.Sleep(1 * time.Second)
        fmt.Println("Canceling doWork goroutine...")
        close(done)
    }()

    <-terminated
    fmt.Println("Done.")

}
```

### OR Channel

At times you may find yourself wanting to combine one or more done channels into a single done channel that closes if any of its component channels close.

```go
package main

import (
    "fmt"
    "time"
)

func main() {
    var or func(channels ...<-chan interface{}) <-chan interface{}

    or = func(channels ...<-chan interface{}) <-chan interface{} {
        switch len(channels) {
        case 0:
            return nil
        case 1:
            return channels[0]
        }

        orDone := make(chan interface{})
        go func() {
            defer close(orDone)

            switch len(channels) {
            case 2:
                select {
                case <-channels[0]:
                case <-channels[1]:
                }
            default:
                select {
                case <-channels[0]:
                case <-channels[1]:
                case <-channels[2]:

                case <-or(append(channels[3:], orDone)...):
                }
            }
        }()

        return orDone
    }

    sig := func(after time.Duration) <-chan interface{} {
        c := make(chan interface{})
        go func() {
            defer close(c)
            time.Sleep(after)
        }()
        return c
    }

    start := time.Now()
    <-or(
        sig(2*time.Hour),
        sig(5*time.Minute),
        sig(1*time.Second),
        sig(1*time.Hour),
        sig(1*time.Minute),
    )

    fmt.Printf("done after %v", time.Since(start))
}
```

### OR Done Channel

### Error Handling

```bash
package main

import (
    "fmt"
    "net/http"
)

type Result struct {
    Error    error
    Response *http.Response
}

func main() {
    checkStatus := func(done <-chan interface{}, urls ...string) <-chan Result {
        results := make(chan Result)
        go func() {
            defer close(results)

            for _, url := range urls {
                var result Result
                resp, err := http.Get(url)
                result = Result{Error: err, Response: resp}
                select {
                case <-done:
                    return
                case results <- result:
                }
            }
        }()

        return results
    }

    done := make(chan interface{})
    defer close(done)

    urls := []string{"https://www.google.com", "https://badhost"}
    for result := range checkStatus(done, urls...) {
        if result.Error != nil {
            fmt.Printf("error: %v", result.Error)
            continue
        }
        fmt.Printf("Response: %v\n", result.Response.Status)
    }
}
```

### Pipelines

A pipeline is just another tool you can use to form an abstraction in your system.

```go
multiply := func(values []int, multiplier int) []int {
    multipliedValues := make([]int, len(values))
    for i, v := range values {
        multipliedValues[i] = v * multiplier
    }
    return multipliedValues
}

add := func(values []int, additive int) []int {
    addedValues := make([]int, len(values))
    for i, v := range values {
        addedValues[i] = v + additive
    }
    return addedValues
}

ints := []int{1, 2, 3, 4}
for _, v := range add(multiply(ints, 2), 1) {
    fmt.Println(v)
}
```

### Fan In

### Fan Out



### References:


[Go Programming Language](http://www.gopl.io)


[Go Concurrency in Go](https://katherine.cox-buday.com/concurrency-in-go)

