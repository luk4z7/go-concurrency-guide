package main

import (
	"fmt"
	"math/rand"
	"time"
)

type typedef int

const (
	doWork1 typedef = iota
	doWork2
	doWork3
	doWork4
	doWork5
)

var (
	activeDoWork typedef = doWork1

	doWork1Fn = func(done <-chan interface{}, pulseInterval time.Duration) (<-chan interface{}, <-chan time.Time) {
		// Here we set up a channel to send heartbeats on. We return this out of doWork.
		heartbeat := make(chan interface{})
		results := make(chan time.Time)

		go func() {
			defer close(heartbeat)
			defer close(results)

			// Here we set the heartbeat to pulse at the pulseInterval we were given.
			// Every pulseInterval there will be something to read on this channel.
			pulse := time.Tick(pulseInterval)

			// This is just another ticker used to simulate work coming in.
			// We choose a duration greater than the pulseInterval so that we
			// can see some heartbeats coming out of the goroutine.
			workGen := time.Tick(2 * pulseInterval)

			sendPulse := func() {
				select {
				case heartbeat <- struct{}{}:
				// Note that we include a default clause.
				// We must always guard against the fact that no one
				// may be listening to our heartbeat.
				// The results emitted from the goroutine are critical, but the pulses are not.
				default:
					fmt.Println("without listener")
				}
			}
			_ = sendPulse

			sendResult := func(r time.Time) {
				for {
					select {
					case <-done:
						return

					// note 5
					// Just like with done channels, anytime you perform a send or receive,
					// you also need to include a case for the heartbeat’s pulse.
					case <-pulse:
						sendPulse()
						// is the same below to use sendPulse(),
						// but with more critical if the heartbeat that are no listener
						// heartbeat <- struct{}{}
					case results <- r:
						return
					}
				}
			}

			// simulate an erro to send healthy of the heartbeat
			// for i := 0; i < 2; i++ {
			for {
				select {
				case <-done:
					return

				// the same as note 5
				case <-pulse:
					sendPulse()
					// heartbeat <- struct{}{}
				case r := <-workGen:
					sendResult(r)

				}
			}
		}()

		return heartbeat, results
	}

	doWork2Fn = func(done <-chan interface{}) (<-chan interface{}, <-chan int) {
		// Here we create the heartbeat channel with a buffer of one.
		// This ensures that there’s always at least one pulse sent out
		// even if no one is listening in time for the send to occur.
		heartbeatStream := make(chan interface{}, 1)
		workStream := make(chan int)

		go func() {
			defer close(heartbeatStream)
			defer close(workStream)

			for i := 0; i < 10; i++ {
				// Here we set up a separate select block for the heartbeat.
				// We don’t want to include this in the same select block as
				// the send on results because if the receiver isn’t ready for the result,
				// they’ll receive a pulse instead, and the current value of the result will be lost.
				// We also don’t include a case statement for the done channel since we have a
				// default case that will just fall through.
				select {
				case heartbeatStream <- struct{}{}:

				// Once again we guard against the fact that no one may be
				// listening to our heartbeats.
				// Because our heartbeat channel was created with a buffer of one,
				// if someone is listening, but not in time for the first pulse,
				// they’ll still be notified of a pulse.
				default:

				}

				select {
				case <-done:
					return
				case workStream <- rand.Intn(10):
				}
			}
		}()

		return heartbeatStream, workStream
	}

	doWork3Fn = func(done <-chan interface{}, nums ...int) (<-chan interface{}, <-chan int) {
		heartbeat := make(chan interface{}, 1)
		intStream := make(chan int)

		go func() {
			defer close(heartbeat)
			defer close(intStream)

			// Here we simulate some kind of delay before the goroutine can begin working.
			// In practice this can be all kinds of things and is nondeterministic.
			// I’ve seen delays caused by CPU load, disk contention, network latency, and goblins.
			time.Sleep(2 * time.Second)

			for _, n := range nums {
				select {
				case heartbeat <- struct{}{}:
				default:
				}

				select {
				case <-done:
					return
				case intStream <- n:
				}
			}

		}()

		return heartbeat, intStream
	}

	doWork4Fn = func(done <-chan interface{}, pulseInterval time.Duration, nums ...int) (
		<-chan interface{}, <-chan int) {

		heartbeat := make(chan interface{}, 1)

		intStream := make(chan int)
		go func() {
			defer close(heartbeat)
			defer close(intStream)

			time.Sleep(2 * time.Second)
			pulse := time.Tick(pulseInterval)

			// We’re using a label here to make continuing from the inner loop a little simpler.
		numLoop:
			for _, n := range nums {

				// We require two loops: one to range over our list of numbers,
				// and this inner loop to run until the number is successfully sent on the intStream.
				for {
					select {
					case <-done:
						return
					case <-pulse:
						select {
						case heartbeat <- struct{}{}:
						default:
						}
					case intStream <- n:
						// Here we continue executing the outer loop.
						continue numLoop
					}
				}
			}
		}()

		return heartbeat, intStream
	}

	receive1 = func(heartbeat <-chan interface{}, results <-chan time.Time, timeout time.Duration) {
		for {
			select {

			// Here we select on the heartbeat.
			// When there are no results,
			// we are at least guaranteed a message from the
			// heartbeat channel every timeout/2.
			// If we don’t receive it, we know there’s
			// something wrong with the goroutine itself.
			case _, ok := <-heartbeat:
				// os pulsos que são enviados por sendPulse(), somente serão retornados
				// caso tenha um listener, caso contrário entrada no default do switch
				if ok == false {
					return
				}

				fmt.Println("pulse")

			// Here we select from the results channel;
			// nothing fancy going on here.
			case r, ok := <-results:
				if ok == false {
					return
				}

				fmt.Printf("results in second %v \n", r.Second())

			// Here we time out if we haven’t received
			// either a heartbeat or a new result.
			case <-time.After(timeout):
				fmt.Println("worker goroutine is not healthy!")

				return
			}
		}
	}

	receive2 = func(heartbeat <-chan interface{}, results <-chan int) {
		for {
			select {
			case _, ok := <-heartbeat:
				if ok {
					fmt.Println("pulse")
				} else {
					return
				}
			case r, ok := <-results:
				if ok {
					fmt.Printf("results %v\n", r)
				} else {
					return
				}
			}
		}
	}

	receive3 = func(heartbeat <-chan interface{}, results <-chan int, intSlice []int) {
		for i, expected := range intSlice {
			select {
			case r := <-results:
				if r != expected {
					fmt.Printf("index %v: expected %v, but received %v \n", i, expected, r)
				}

			// Here we time out after what we think is a reasonable duration
			// to prevent a broken goroutine from deadlocking our test.
			case <-time.After(1 * time.Second):
				fmt.Printf("test timed out")
			}
		}
	}

	receive4 = func(heartbeat <-chan interface{}, results <-chan int, intSlice []int) {
		i := 0
		for r := range results {
			if expected := intSlice[i]; r != expected {
				fmt.Sprintf("index %v: expected %v, but received %v,", i, expected, r)
			}
			i++
		}
	}

	receive5 = func(heartbeat <-chan interface{}, results <-chan int, timeout time.Duration, intSlice []int) {
		i := 0
		for {
			select {
			case r, ok := <-results:
				if ok == false {
					return
				} else if expected := intSlice[i]; r != expected {
					fmt.Sprintf("index %v: expected %v, but received %v,", i, expected, r)
				}
				i++

			// We also select on the heartbeat here to keep the timeout from occuring.
			case <-heartbeat:
			case <-time.After(timeout):
				fmt.Println("test timed out")
			}
		}
	}
)

func main() {
	// Notice that because we might be sending out multiple pulses
	// while we wait for input, or multiple pulses while waiting to send results,
	// all the select statements need to be within for loops.
	// Looking good so far; how do we utilize this function and consume
	// the events it emits? Let’s take a look:
	done := make(chan interface{})

	// We set up the standard done channel and close it after 10 seconds.
	// This gives our goroutine time to do some work.
	time.AfterFunc(10*time.Second, func() {
		close(done)
	})

	// Here we set our timeout period.
	// We’ll use this to couple our heartbeat interval to our timeout.
	const timeout = 2 * time.Second

	switch activeDoWork {
	case doWork1:
		// We pass in timeout/2 here.
		// This gives our heartbeat an extra tick to
		// respond so that our timeout isn’t too sensitive.
		heartbeat, results := doWork1Fn(done, timeout/2)
		receive1(heartbeat, results, timeout)

	case doWork2:
		heartbeat, results := doWork2Fn(done)
		receive2(heartbeat, results)

	case doWork3:
		intSlice := []int{0, 1, 2, 3, 5}
		heartbeat, results := doWork3Fn(done, intSlice...)

		receive3(heartbeat, results, intSlice)

	case doWork4:
		intSlice := []int{0, 1, 2, 3, 5}
		heartbeat, results := doWork3Fn(done, intSlice...)

		<-heartbeat
		receive4(heartbeat, results, intSlice)
	case doWork5:
		intSlice := []int{0, 1, 2, 3, 5}
		const timeout = 2 * time.Second

		heartbeat, results := doWork4Fn(done, timeout/2, intSlice...)

		// We still wait for the first heartbeat to occur to indicate we’ve entered the goroutine’s loop.
		<-heartbeat
		receive5(heartbeat, results, timeout, intSlice)
	}
}