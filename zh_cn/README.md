
# Go Concurrency Guide

This guide is built on top of the some examples of the book `Go Concurrency in Go` and `Go Programming Language`

- [竞赛条件和数据竞赛](#race-condition-and-data-race)
- [Memory Access Synchronization](#memory-access-synchronization)
    -  [Mutex](#mutex)
    -  [WaitGroup](#waitgroup)
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
    - [Error Handling](#error-handling)
    - [Pipelines](#pipelines)
    - [Fan-in and Fan-out](#fan-in-and-fan-out)
    - [Or done channel](#or-done-channel)
    - [Tee channel](#tee-channel)
    - [Bridge channel](#bridge-channel)
    - [Queuing](#queuing)
    - [Context package](#context-package)
    - [HeartBeats](#heartbeats)
    - [Replicated Requests](#replicated-requests)
- [Scheduler Runtime](#scheduler-runtime)
- [References](#references)


## 竞赛条件和数据竞赛

当两个或更多的操作必须以正确的顺序执行时，就会出现竞赛条件，但程序的编写没有保证这个顺序得到维持。

数据竞赛是指一个并发的操作试图读取一个变量，而在某个不确定的时间，另一个并发的操作试图写到同一个变量。主函数（main func）是主程序（goroutine）。

```go
func main() {
    var data int
    go func() {
        data++
    }()

    if data == 0 {
        fmt.Printf("the value is %d", data)
    }
}
```


## 内存访问同步化

The sync package contains the concurrency primitives that are most useful for low-level memory access synchronization.
Critical section is the place in your code that has access to a shared memory

sync包包含了对低级内存访问同步最有用的并发原语。关键部分是你的代码中可以访问共享内存的地方。

### Mutex

Mutex是 "相互排斥 "的意思，是一种保护你的程序的关键部分的方法。

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

调用添加一组goroutines的方法

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

更加细化的内存控制，可以申请只读锁

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
fmt.Fprintf(tw, "Readers\tRWMutex\tMutex\n")

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

如果有某种方法能让一个goroutine有效地睡眠，直到它被信号唤醒并检查其状况，那就更好了。这正是Cond类型为我们所做的。

Cond和Broadcast是提供通知被Wait调用阻塞的goroutine的方法，即条件已经被触发。

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

            fmt.Println("Registered and wait ... ")
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
[cond samples](https://github.com/luk4z7/go-concurrency-guide/blob/main/sync/cond/main.go)


### Once

确保即使在几个goroutine之间也只执行一个程序

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

管理连接池，一个数量

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

    // Get call New function defined in pool if there is no instance started
    myPool.Get()
    instance := myPool.Get()
    fmt.Println("instance", instance)

    // here we put a previously retrieved instance back into the pool, 
    // this increases the number of instances available to one
    myPool.Put(instance)

    // when this call is executed, we will reuse the 
    // previously allocated instance and put it back in the pool
    myPool.Get()

    var numCalcsCreated int
    calcPool := &sync.Pool{
        New: func() interface{} {
            fmt.Println("new calc pool")

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
[sync samples](https://github.com/luk4z7/go-concurrency-guide/tree/main/sync)



## 死锁、活锁和饿死

### 死锁

死锁是指一个程序中所有并发进程都在相互等待。

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


### 活锁

活锁是指正在积极执行并发操作的程序，但这些操作对推进程序的状态毫无作用。

```go
package main

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
}
```


### 饥饿

饥饿是指一个并发进程无法获得执行工作所需的所有资源的任何情况。

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


## Channels通道

channel是Go中从Hoare的CSP派生出来的同步原语之一。虽然它们可以用来同步访问内存，但它们最好是用来在goroutines之间交流信息，channel的默认值为："nil"。

声明一个channel来读取和发送

```go
stream := make(chan interface{})
```

channel

```go
stream := make(<-chan interface{})
```

声明只能发送的单向channel

```go
stream := make(chan<- interface{})
```

并不经常看到实例化channel的单向性，只在函数的参数中出现，之所以常见是因为Go将其转换为内含性

```go
var receiveChan <-chan interface{}
var sendChan chan<- interface{}
dataStream := make(chan interface{})

// Valid statements:
receiveChan = dataStream
sendChan = dataStream
```

接收
```go
<-stream
```

发送
```go
stream <- "Hello world"
```

遍历一个channel
如果channel被关闭，for range会打破循环。

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

**无缓冲的channel**

在一个未缓冲的channel上的发送操作会阻塞发送的goroutine，直到另一个goroutine在同一channel上执行相应的接收操作；这时，值被传递，两个goroutine可以继续。另一方面，如果事先尝试了接收操作，接收的goroutine就会被阻塞，直到另一个goroutine在同一channel上执行发送。在无缓冲channel上的通信使发送和接收的goroutine同步。因为这个原因，无缓冲channel有时被称为同步channel。当一个值在非缓冲channel上被发送时，该值的接收会在发送goroutine再次唤醒之前发生。在讨论并发性时，当我们说x发生在y之前时，我们并不是简单地指x在时间上发生在y之前；我们的意思是这是有保证的，你之前的所有效果，如对变量的更新，都将完成，你可以指望它们。当x没有发生在y之前或y之后时，我们说x与y同时发生。这并不是说x和y一定是同时发生的；这只是意味着我们不能假设你的顺序是什么

**有缓冲的channel**<br/>

读和写一个都是满的或空的有缓冲channel，将被阻塞。

```go
var dataStream chan interface{}
dataStream = make(chan interface{}, 4)
```

both, read and send a channel empty cause deadlock

既读又送的空的channel会造成死锁

```go
var dataStream chan interface{}
<-dataStream // This panics with: fatal error: all goroutines are asleep - deadlock!
```
```
   goroutine 1 [chan receive (nil chan)]:
   main.main()
       /tmp/babel-23079IVB/go-src-23079O4q.go:9 +0x3f
   exit status 2
```

```go
var dataStream chan interface{}
dataStream <- struct{}{} // This produces: fatal error: all goroutines are asleep - deadlock!
```
```
  goroutine 1 [chan send (nil chan)]:
  main.main()
      /tmp/babel-23079IVB/go-src-23079dnD.go:9 +0x77
  exit status 2
```

关闭一个空的channel会panic

```go
var dataStream chan interface{}
close(dataStream) // This produces: panic: close of nil channel
```
```
  goroutine 1 [running]:
  panic(0x45b0c0, 0xc42000a160)
      /usr/local/lib/go/src/runtime/panic.go:500 +0x1a1
  main.main()
      /tmp/babel-23079IVB/go-src-230794uu.go:9 +0x2a
  exit status 2 Yipes! This is probably
```

channel操作的结果表

Operation | Channel State      | Result
----------|--------------------|-------------
Read      | nil                | Block
_         | Open and Not Empty | Value
_         | Open and Empty     | Block
_         | Close              | default value, false
_         | Write Only         | Compilation Error
Write     | nil                | Block
_         | Open and Full      | Block
_         | Open and Not Full  | Write Value
_         | Closed             | panic
_         | Receive Only       | Compilation Error
Close     | nil                | panic 
_         | Open and Not Empty | Closes Channel; reads succeed until channel is drained, then reads produce default value
_         | Open and Empty     | Closes Channel; reads produces default value
_         | Closed             | panic

`TIP: 无法关闭一个只接收的channel`

* 让我们从channel所有者开始。拥有一个channel的goroutine必须。
  * 1 - 实例化该channel。 
  * 2 - 执行写操作，或将所有权传递给另一个goroutine。 
  * 3 - 关闭该channel。 
  * 4 - 封装这个列表中的前三件事，并通过一个读channel暴露它们。

* 当分配channel所有者的职责时，有几件事会发生。
  * 1 - 因为我们是初始化channel的人，所以我们消除了向一个nil channel写东西而造成的死锁风险。 
  * 2 - 因为我们是初始化channel的人，所以我们消除了因关闭一个空的channel而引起的panic的风险。 
  * 3 - 因为我们是决定channel何时被关闭的人，所以我们通过向一个关闭的channel写东西来消除panic的风险。 
  * 4 - 因为我们是决定channel何时被关闭的人，所以我们消除了因多次关闭channel而panic的风险。 
  * 5 - 我们在编译时挥舞着类型检查器，以防止对我们的channel进行不当的写入。

```go
chanOwner := func() <-chan int {
    resultStream := make(chan int, 5) 
    go func() { 
        defer close(resultStream) 
        for i := 0; i <= 5; i++ {
            resultStream <- i
        }
    }()
    return resultStream 
}

resultStream := chanOwner()
for result := range resultStream { 
    fmt.Printf("Received: %d\n", result)
}

fmt.Println("Done receiving!")
```

明确创建channel的所有者，往往能更好地控制该channel何时应被关闭及其操作，避免将这些功能委托给系统的其他方法/功能，避免读取已关闭的channel或向同一已完成的channel发送数据

**select**

select的工作方式与switch不一样，switch是顺序的，如果没有一个条件得到满足，执行不会自动下降。

```go
var c1, c2 <-chan interface{}
var c3 chan<- interface{}
select {
case <- c1:
    // Do something
case <- c2:
    // Do something
case c3<- struct{}{}:

}
```

相反，所有的channel读和写都是同时考虑的，看是否有channel准备好了：在读的情况下，channel被填满或关闭，在写的情况下，channel没有达到容量。如果没有一个channel准备好了，整个选择命令就会被阻断。然后，当其中一个channel准备好时，操作将继续进行，其相应的指令将被执行。

```go
start := time.Now()
c := make(chan interface{})
go func() {
    time.Sleep(5*time.Second)
    close(c) 
}()

fmt.Println("Blocking on read...")
select {
case <-c: 
    fmt.Printf("Unblocked %v later.\n", time.Since(start))
}
```

questions when work with select and channels

select和channel工作时的问题

1 - 当多个channel有东西要读时，会发生什么？ 

```go
c1 := make(chan interface{}); close(c1)
c2 := make(chan interface{}); close(c2)

var c1Count, c2Count int
for i := 1000; i >= 0; i-- {
    select {
    case <-c1:
        c1Count++
    case <-c2:
        c2Count++
    }
}

fmt.Printf("c1Count: %d\nc2Count: %d\n", c1Count, c2Count)
```

This produces:<br/>
c1总数: 505<br/>
c2总数: 496<br/>

一半由c1读取一半由c2读取的Go运行时，无法准确预测各自的读取量，也不会对两者完全相同，可能发生但无法预测，运行时对拥有2个channel接收信息的意图一无所知，或者像我们的例子那样关闭，那么运行时包括一个伪随机的
Go运行时将对select案例语句集进行伪随机统一选择。这只是意味着从你的案例集中，每一个案例都有和其他所有案例一样的机会被选中。

一个好的方法是在你的方程中引入一个随机变量--在这个例子中，是从哪个channel中选择。通过权衡每个channel被平等使用的机会，所有使用select语句的Go程序在平均情况下都会表现良好。

2 - 如果从来没有任何channel准备就绪，怎么办？ 

```go
var c <-chan int
select {
case <-c: 
case <-time.After(1 * time.Second):
    fmt.Println("Timed out.")
}

```

为了解决channel被阻塞的问题，可以使用默认的方式来执行一些其他操作，或者在第一个例子中用time.After进行超时处理

3 - 如果我们想做什么，但目前没有channel准备好，怎么办？ 使用`default`。

```go
start := time.Now()
var c1, c2 <-chan int
select {
case <-c1:
case <-c2:
default:
    fmt.Printf("In default after %v\n\n", time.Since(start))
}
```

退出一个select块 

```go
done := make(chan interface{})
go func() {
    time.Sleep(5*time.Second)
    close(done)
}()

workCounter := 0
loop:
for {
    select {
    case <-done:
        break loop
    default:
    }

    // Simulate work
    workCounter++
    time.Sleep(1*time.Second)
}

fmt.Printf("Achieved %v cycles of work before signalled to stop.\n", workCounter)
```

永远阻塞 

```go
select {}
```

**GOMAXPROCS**<br/>
Prior to Go 1.5, GOMAXPROCS was always set to one, and usually you’d find this snippet in most Go programs:

在Go 1.5之前，GOMAXPROCS总是被设置为1，通常你会在大多数Go程序中发现这个片段。

```go
runtime.GOMAXPROCS(runtime.NumCPU())
```

This function controls the number of operating system threads that will host so-called “Work Queues.”

该功能控制操作系统线程的数量，这些线程将承载所谓的 "工作队列"。

[documentation](https://pkg.go.dev/runtime#GOMAXPROCS)


[Use a sync.Mutex or a channel?](https://github.com/golang/go/wiki/MutexOrChannel)

作为一个一般的指导：

Channel | Mutex
--------|-------
passing ownership of data, <br/>distributing units of work, <br/>communicating async results | caches, <br/>state


_"Do not communicate by sharing memory; instead, share memory by communicating. (copies)"_

_"不要通过共享内存来交流；相反，通过交流来共享内存。(复制原话)"_



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

Lexical confinement involves using lexical scope to expose only the correct data and concurrency primitives for multiple concurrent processes to use.

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
[sample](https://github.com/luk4z7/go-concurrency-guide/tree/main/patterns/confinement)


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
[sample](https://github.com/luk4z7/go-concurrency-guide/tree/main/patterns/cancellation)


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
[sample](https://github.com/luk4z7/go-concurrency-guide/tree/main/patterns/orchannel)


### Error Handling

```go
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
[sample](https://github.com/luk4z7/go-concurrency-guide/tree/main/patterns/errorhandler)


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
[sample](https://github.com/luk4z7/go-concurrency-guide/tree/main/patterns/pipelines)


### Fan-in and Fan-out

Fan-out is a term to describe the process of starting multiple goroutines to handle pipeline input, and fan-in is a term to describe the process of combining multiple outputs into one channel.

```go
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
```
[sample](https://github.com/luk4z7/go-concurrency-guide/tree/main/patterns/fanoutfanin)


### Or done channel

Or done is a way to encapsulate verbosity that can be achieved through for/select breaks to check when a channel has ended, and also avoiding goroutine leakage, the code below could be replaced by a closure that encapsulates that verbosity

```go
for val := range myChan {
    // Do something with val
}

loop:
for {
    select {
    case <-done:
        break loop
    case maybeVal, ok := <-myChan:
        if ok == false {
            return // or maybe break from for
        }
        // Do something with val
    }
}
```

can be created an isolation, a function/method, closure, creating a single goroutine

```go
orDone := func(done, c <-chan interface{}) <-chan interface{} {
    valStream := make(chan interface{})
    go func() {
        defer close(valStream)
        for {
            select {
            case <-done:
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

for val := range orDone(done, myChan) {
    // Do something with val
}
```
[sample](https://github.com/luk4z7/go-concurrency-guide/tree/main/patterns/ordonechannel)


### Tee channel

Pass the it a channel to read from, and it will return two separate channels that will get the same value:

```go
tee := func(done <-chan interface{}, in <-chan interface{}) (_, _ <-chan interface{}) {

    out1 := make(chan interface{})
    out2 := make(chan interface{})

    go func() {
        defer close(out1)
        defer close(out2)
        for val := range orDone(done, in) {
            var out1, out2 = out1, out2
            for i := 0; i < 2; i++ {
                select {
                case <-done:
                case out1 <- val:
                    out1 = nil
                case out2 <- val:
                    out2 = nil
                }
            }
        }
    }()

    return out1, out2
}
```
[sample](https://github.com/luk4z7/go-concurrency-guide/tree/main/patterns/teechannel)


### Bridge channel

With this patterns is possible to create a function that destruct a channel of channels into a single channel

```go
bridge := func(done <-chan interface{}, chanStream <-chan <-chan interface{}) <-chan interface{} {
    valStream := make(chan interface{})
    go func() {
        defer close(valStream)
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
```
[sample](https://github.com/luk4z7/go-concurrency-guide/tree/main/patterns/bridgechannel)


### Queuing

buffered channel is a type of queue, Adding queuing prematurely can hide synchronization issues such as deadlocks, we can use the queue to make
a limit to processing, in this process when the `limit <- struct{}{}` is full the queue is wait to be released `<-limit`, if we remove them the 50 goroutines are created at the same time

```go
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
```
[sample](https://github.com/luk4z7/go-concurrency-guide/tree/main/patterns/queuing)


### Context package

in concurrent programs it’s often necessary to preempt operations because of timeouts, cancellation, or failure of another portion of the system. We’ve looked at the idiom of creating a done channel, which flows through your program and cancels all blocking concurrent operations. This works well, but it’s also somewhat limited.

It would be useful if we could communicate extra information alongside the simple notification to cancel: why the cancellation was occuring, or whether or not our function has a deadline by which it needs to complete.

see below an example to pass value into context, the context package serves two primary purposes: 
- To provide an API for canceling branches of your call-graph.  
- To provide a data-bag for transporting request-scoped data through your call-graph


```go
package main

import (
    "context"
    "fmt"
)

func main() {
    ProcessRequest("jane", "abc123")
}

func ProcessRequest(userID, authToken string) {
    ctx := context.WithValue(context.Background(), "userID", userID)
    ctx = context.WithValue(ctx, "authToken", authToken)
    HandleResponse(ctx)
}

func HandleResponse(ctx context.Context) {
    fmt.Printf(
        "handling response for %v (%v)",
        ctx.Value("userID"),
        ctx.Value("authToken"),
    )
}
```

another example with `Timeout`, cancellation in a function has three aspects:

- A goroutine’s parent may want to cancel it. 
- A goroutine may want to cancel its children.  
- Any blocking operations within a goroutine need to be preemptable so that it may be canceled.

The context package helps manage all three of these.

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"
)

func main() {
    var wg sync.WaitGroup
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    wg.Add(1)
    go func() {
        defer wg.Done()

        if err := printGreeting(ctx); err != nil {
            fmt.Printf("cannot print greeting: %v\n", err)
            cancel()
        }
    }()

    wg.Add(1)
    go func() {
        defer wg.Done()
        if err := printFarewell(ctx); err != nil {
            fmt.Printf("cannot print farewell: %v\n", err)
        }
    }()

    wg.Wait()
}

func printGreeting(ctx context.Context) error {
    greeting, err := genGreeting(ctx)
    if err != nil {
        return err
    }
    fmt.Printf("%s world!\n", greeting)

    return nil
}

func printFarewell(ctx context.Context) error {
    farewell, err := genFarewell(ctx)
    if err != nil {
        return err
    }
    fmt.Printf("%s world!\n", farewell)

    return nil
}

func genGreeting(ctx context.Context) (string, error) {
    ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
    defer cancel()

    switch locale, err := locale(ctx); {
    case err != nil:
        return "", err
    case locale == "EN/US":
        return "hello", nil
    }

    return "", fmt.Errorf("unsupported locale")
}

func genFarewell(ctx context.Context) (string, error) {
    switch locale, err := locale(ctx); {
    case err != nil:
        return "", err
    case locale == "EN/US":
        return "goodbye", nil
    }

    return "", fmt.Errorf("unsupported locale")
}

func locale(ctx context.Context) (string, error) {
    if deadline, ok := ctx.Deadline(); ok {
        if deadline.Sub(time.Now().Add(1*time.Minute)) <= 0 {
            return "", context.DeadlineExceeded
        }
    }

    select {
    case <-ctx.Done():
        return "", ctx.Err()
    case <-time.After(1 * time.Minute):
    }

    return "EN/US", nil
}
```
[sample](https://github.com/luk4z7/go-concurrency-guide/tree/main/patterns/contextpackage)


### HeartBeats

Heartbeats are a way for concurrent processes to signal life to outside parties. They get their name from human anatomy wherein a heartbeat signifies life to an observer. Heartbeats have been around since before Go, and remain useful within it.

There are two different types of heartbeats:
- Heartbeats that occur on a time interval.
- Heartbeats that occur at the beginning of a unit of work

[sample](https://github.com/luk4z7/go-concurrency-guide/tree/main/patterns/heartbeats)


### Replicated Requests

You should only replicate requests like this to handlers that have different runtime conditions: different processes, machines, paths to a data store, or access to different data stores. While this can be expensive to set up and maintain, if speed is your goal this is a valuable technique. Also, this naturally provides fault tolerance and scalability.

The only caveat to this approach is that all handlers need to have equal opportunity to fulfill the request. In other words, you won't have a chance to get the fastest time from a handler that can't fulfill the request. As I mentioned, whatever resources the handlers are using to do their work also need to be replicated. A different symptom of the same problem is uniformity. If your handles are very similar, the chances that either one is an outlier are less.

```go
package main

import (
    "fmt"
    "math/rand"
    "sync"
    "time"
)

func main() {

    doWork := func(done <-chan interface{}, id int, wg *sync.WaitGroup, result chan<- int) {
        started := time.Now()
        defer wg.Done()

        // Simulate random load
        simulatedLoadTime := time.Duration(1+rand.Intn(5)) * time.Second
        select {
        case <-done:
        case <-time.After(simulatedLoadTime):
        }

        select {
        case <-done:
        case result <- id:
        }

        took := time.Since(started)
        // Display how long handlers would have taken
        if took < simulatedLoadTime {
            took = simulatedLoadTime

        }

        fmt.Printf("%v took %v\n", id, took)
    }

    done := make(chan interface{})
    result := make(chan int)

    var wg sync.WaitGroup
    wg.Add(10)

    // Here we start 10 handlers to handle our requests.
    for i := 0; i < 10; i++ {
        go doWork(done, i, &wg, result)
    }

    // This line grabs the first returned value from the group of handlers.
    firstReturned := <-result

    // Here we cancel all the remaining handlers.
    // This ensures they don’t continue to do unnecessary work.
    close(done)
    wg.Wait()

    fmt.Printf("Received an answer from #%v\n", firstReturned)
}
```
[sample](https://github.com/luk4z7/go-concurrency-guide/tree/main/patterns/replicatedrequests)



## Scheduler Runtime

Go will handle multiplexing goroutines onto OS threads for you.

The algorithm it uses to do this is known as a work `stealing strategy`.

fair scheduling. In an effort to ensure all processors were equally utilized, we could evenly distribute the load between all available processors. Imagine there are n processors and x tasks to perform. In the fair scheduling strategy, each processor would get x/n tasks:

Go models concurrency using a fork-join model.

As a refresher, remember that Go follows a fork-join model for concurrency. Forks are when goroutines are started, and join points are when two or more goroutines are synchronized through channels or types in the sync package. The work stealing algorithm follows a few basic rules. Given a thread of execution:

At a fork point, add tasks to the tail of the deque associated with the thread.


Go scheduler’s job is to distribute runnable goroutines over multiple worker OS threads that runs on one or more processors. In multi-threaded computation, two paradigms have emerged in scheduling: work sharing and work stealing.

- Work-sharing: When a processor generates new threads, it attempts to migrate some of them to the other processors with the hopes of them being utilized by the idle/underutilized processors.
- Work-stealing: An underutilized processor actively looks for other processor’s threads and “steal” some.

The migration of threads occurs less frequently with work stealing than with work sharing. When all processors have work to run, no threads are being migrated. And as soon as there is an idle processor, migration is considered.

Go has a work-stealing scheduler since 1.1, contributed by Dmitry Vyukov. This article will go in depth explaining what work-stealing schedulers are and how Go implements one.


**Scheduling basics**

Go has an M:N scheduler that can also utilize multiple processors. At any time, M goroutines need to be scheduled on N OS threads that runs on at most GOMAXPROCS numbers of processors. Go scheduler uses the following terminology for goroutines, threads and processors:

- G: goroutine<br/>
- M: OS thread (machine)<br/>
- P: processor<br/>

There is a P-specific local and a global goroutine queue. Each M should be assigned to a P. Ps may have no Ms if they are blocked or in a system call. At any time, there are at most GOMAXPROCS number of P. At any time, only one M can run per P. More Ms can be created by the scheduler if required.
[runtime doc](https://github.com/golang/go/blob/master/src/runtime/proc.go)


**Why have a scheduler?**

goroutines are user-space threads
conceptually similar to kernel threads managed by the OS, but managed entirely by the Go runtime

lighter-weight  and cheaper than kernel threads.

* smaller memory footprint:
    * initial goroutine stack = 2KB; default thread stack = 8KB
    * state tracking overhead
    * faster creation, destruction, context switchesL
    * goroutines switches = ~tens of ns; thread switches = ~ a us.

Go schedule put her  goroutines on kernel threads which run on the CPU



### References:


[Go Programming Language](https://www.gopl.io)


[Go Concurrency in Go](https://katherine.cox-buday.com/concurrency-in-go)

