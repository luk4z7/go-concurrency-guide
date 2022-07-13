package go_concurrency_guide

import "fmt"

func main() {
	var data int
	go func() {
		data++
	}()

	if data == 0 {
		fmt.Println("the value is %d", data)
	}
}
