
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

在Go 1.5之前，GOMAXPROCS总是被设置为1，通常你会在大多数Go程序中发现这个片段：

```go
runtime.GOMAXPROCS(runtime.NumCPU())
```

该功能控制操作系统线程的数量，这些线程将承载所谓的 "工作队列"。

[documentation](https://pkg.go.dev/runtime#GOMAXPROCS)

[Use a sync.Mutex or a channel?](https://github.com/golang/go/wiki/MutexOrChannel)

作为一个一般的指导：

Channel | Mutex
--------|-------
passing ownership of data, <br/>distributing units of work, <br/>communicating async results | caches, <br/>state


_"Do not communicate by sharing memory; instead, share memory by communicating. (copies)"_

_"不要通过共享内存来交流；相反，通过交流来共享内存。(复制原话)"_



## 模式

### Confinement限制

限制是一个简单而强大的想法，即确保信息只能从一个并发进程中获得。
有两种可能的限制：临时性的和词汇性的。



临时限制是指你通过一个公约实现限制。

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

词法限制涉及到使用词法范围，只暴露正确的数据和并发原语，供多个并发进程使用。

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


### 取消

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

有时，你可能会发现自己想把一个或多个已完成的通道合并成一个单一的已完成的通道，如果它的任何一个组成通道关闭，这个通道就会关闭。

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


### 错误处理

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

管道只是另一种工具，你可以用来在你的系统中形成一个抽象的概念。

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


### 扇入和扇出

扇出是一个术语，用来描述启动多个goroutines来处理channel输入的过程，而扇入是一个术语，用来描述将多个输出合并成一个channel的过程。

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

或者做是一种封装口令的方法，可以通过for/select断点续传来检查channel何时结束，也可以避免goroutine泄露，下面的代码可以用一个封装口令的闭包来代替。

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

可以创建一个隔离，一个函数/方法，封闭，创建一个单一的goroutine

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

传递给它一个要读取的channel，它将返回两个独立的channel，得到相同的值：

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

有了这个模式，就可以创建一个函数，将一个通道分解为一个单一的channel。

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

缓冲通道是队列的一种，过早地加入队列可以隐藏同步问题，比如死锁，我们可以用队列来做处理的限制，在这个过程中，当`limit <- struct{}{}`满的时候，队列就会等待释放`<-limit`，如果我们删除它们，就会同时创建50个goroutine

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


### 上下文包

在并发程序中，由于超时、取消或系统中另一部分的失败，常常需要抢占操作。我们已经研究过创建一个 "完成"channel的习性，它流经你的程序并取消所有阻塞的并发操作。这很有效，但也有一定的局限性。

如果我们能在简单的取消通知的同时传递额外的信息，那将是非常有用的：为什么会发生取消，或者我们的函数是否有一个需要完成的最后期限。

请看下面一个例子，将值传递到上下文中，上下文包有两个主要目的。

- 为取消你的调用图的分支提供一个API。 
- 提供一个数据包，用于通过你的调用图传输请求范围内的数据。


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

另一个例子是 `Timeout`，一个函数中的取消有三个方面。

- 一个goroutine的父程序可能想取消它。
- 一个goroutine可能想取消它的子程序。 
- 在一个goroutine中的任何阻塞操作都需要是可抢占的，这样它就可以被取消了。

上下文包可以帮助管理所有这三种情况。

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


### 心跳

心脏跳动是并发进程向外界发出生命信号的一种方式。它们的名字来自于人体解剖学，在那里，心跳对观察者来说意味着生命。心跳在Go之前就已经存在了，并且在Go中仍然有用。

有两种不同类型的心跳。

- 在一个时间间隔内发生的心跳。
- 在一个工作单元开始时发生的心跳

[sample](https://github.com/luk4z7/go-concurrency-guide/tree/main/patterns/heartbeats)


### 复制的请求

你应该只把这样的请求复制到具有不同运行时间条件的处理程序：不同的进程、机器、通往数据存储的路径，或对不同数据存储的访问。虽然这可能是昂贵的设置和维护，但如果速度是你的目标，这是一个宝贵的技术。另外，这自然提供了容错和可扩展性。

这种方法唯一需要注意的是，所有的处理程序都需要有平等的机会来满足请求。换句话说，你不会有机会从一个不能满足请求的处理程序那里得到最快的时间。正如我所提到的，无论处理程序使用什么资源来做他们的工作，也需要被复制。同一问题的另一个症状是统一性。如果你的处理程序非常相似，那么任何一个处理程序是异常点的机会就会减少。

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



## 调度器运行时间

Go将为你处理多路复用goroutines到操作系统线程上。

它用来做这件事的算法被称为工作 `窃取策略`。

公平调度。为了确保所有的处理器都得到平等的利用，我们可以在所有可用的处理器之间均匀地分配负载。想象一下，有n个处理器和x个任务要执行。在公平调度策略中，每个处理器会得到x/n个任务。

Go使用分叉连接模型对并发进行建模。

作为复习，请记住Go是采用叉接模型进行并发的。叉点是goroutines启动的时候，而连接点是两个或多个goroutines通过channel或同步包中的类型进行同步的时候。窃取算法遵循一些基本规则。给定一个执行的线程：

在分叉点，将任务添加到与该线程相关的deque的尾部。


Go调度器的工作是将可运行的goroutines分配到多个运行在一个或多个处理器上的工人操作系统线程上。在多线程计算中，调度出现了两种范式：工作共享和工作偷窃。

- 工作共享。当一个处理器产生新的线程时，它试图将其中一些线程迁移到其他处理器上，希望它们能被闲置/未被充分利用的处理器所利用。
- 窃取工作。一个未被充分利用的处理器主动寻找其他处理器的线程并 "偷"走一些。

与工作共享相比，线程的迁移在工作偷窃中发生的频率较低。当所有处理器都有工作要运行时，没有线程被迁移。而只要有一个空闲的处理器，就会考虑迁移。

Go从1.1开始就有一个工作窃取的调度器，由Dmitry Vyukov贡献。本文将深入解释什么是工作窃取的调度器，以及Go是如何实现的。

**调度基本知识**

Go有一个M:N调度器，也可以使用多个处理器。在任何时候，需要在最多运行GOMAXPROCS数量的处理器的N个OS线程上调度M个goroutine。Go调度器对goroutine、线程和处理器使用以下术语：

- G: goroutine<br/>
- M: OS thread (machine)<br/>
- P: processor<br/>

有一个针对P的本地和一个全局的goroutine队列。每个M应该被分配给一个P。如果P被阻塞或处于系统调用中，则可能没有M。在任何时候，最多只有GOMAXPROCS数量的P。在任何时候，每个P只能运行一个M。如果需要，调度器可以创建更多的Ms。

[runtime doc](https://github.com/golang/go/blob/master/src/runtime/proc.go)

**为什么要有一个调度器?**

goroutines是用户空间线程
概念上类似于由操作系统管理的内核线程，但完全由Go运行时管理

比内核线程更轻巧、更便宜。

* 更小的内存占用。
  * 初始goroutine栈=2KB；默认线程栈=8KB
  * 状态跟踪开销
  * 更快的创建、销毁和上下文切换。
  * goroutines开关=~几十ns；线程开关=~一个us。

把她的程序放在内核线程上，在CPU上运行。



### References:


[Go Programming Language](https://www.gopl.io)


[Go Concurrency in Go](https://katherine.cox-buday.com/concurrency-in-go)

