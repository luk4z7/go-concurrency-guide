package main

import (
	"fmt"
	"math/rand"
	"time"
)

// send the same message to different actors
// all actors receive the same work item
// example, send a single signal
// security scanner
type goFile struct {
	name    string
	content string
}

func mockScan() string {
	if rand.Intn(100) > 90 {
		return "ALERT - vulnerability found"
	}

	return "OK - All Correct"
}

func scanSQLInjection(data goFile, res chan<- string) {
	res <- fmt.Sprintf("SQL injection scan: %s scanned, result: %s", data.name, mockScan())
}

func scanTimingExploits(data goFile, res chan<- string) {
	res <- fmt.Sprintf("Timing exploits scan: %s scanned, result: %s", data.name, mockScan())
}

func scanAuth(data goFile, res chan<- string) {
	res <- fmt.Sprintf("Authentication scan: %s scanned, result: %s", data.name, mockScan())
}

func main() {
	si := []goFile{
		{name: "utils.go", content: "package utils\n\nfunc Util() {}"},
		{name: "helper.go", content: "package Helper\n\nfunc Helper() {}"},
		{name: "misc.go", content: "package Misc\n\nfunc Misc() {}"},
		{name: "various.go", content: "package Various\n\nfunc Various() {}"},
	}

	res := make(chan string, len(si)*3)

	for _, d := range si {
		d := d

		// fan-out pass the input directly
		go scanSQLInjection(d, res) // fan-in common result channel
		go scanTimingExploits(d, res)
		go scanAuth(d, res)
	}

	// Scatter-Gather
	for i := 0; i < cap(res); i++ {
		fmt.Println(<-res)
	}

	fmt.Println("main: done")

	NumberOfTheWeekInMonth(time.Now())
}

func NumberOfTheWeekInMonth(now time.Time) int {

	//	beginningOfTheMonth := time.Date(now.Year(), now.Month(), 1, 1, 1, 1, 1, time.UTC)
	//	_, thisWeek := now.ISOWeek()
	//	_, beginningWeek := beginningOfTheMonth.ISOWeek()

	_, w := now.ISOWeek()
	data := fmt.Sprintf("%s%d", now.Format("200601"), w)

	fmt.Println(data)

	return 1
}