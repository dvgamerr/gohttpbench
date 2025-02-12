package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"time"
)

func PrintHeader() {
	// 	fmt.Println(`
	// This is GoHttpBench, Version ` + GBVersion + `, https://github.com/parkghost/gohttpbench
	// Author: Brandon Chen, Email: parkghost@gmail.com
	// Licensed under the MIT license
	// `)
}

func PrintReport(context *Context, stats *Stats) {

	var buffer bytes.Buffer
	var notify bytes.Buffer

	config := context.config
	responseTimeData := stats.responseTimeData
	totalFailedReqeusts := stats.totalFailedReqeusts
	totalRequests := stats.totalRequests
	totalExecutionTime := stats.totalExecutionTime
	totalReceived := stats.totalReceived

	URL, _ := url.Parse(config.url)

	fmt.Fprint(&buffer, "\n\n")
	fmt.Fprintf(&notify, "[%s] ", time.Now().Format("15:04:05.000"))
	fmt.Fprintf(&notify, "%s → ", URL.RequestURI())
	// fmt.Fprintf(&buffer, "Server Software:        %s\n", context.GetString(FieldServerName))
	fmt.Fprintf(&buffer, "Server:        %s:%d\n", config.host, config.port)
	fmt.Fprintf(&buffer, "Path:          %s\n", URL.RequestURI())
	fmt.Fprintf(&buffer, "Length:        %d bytes\n\n", context.GetInt(FieldContentSize))
	fmt.Fprintf(&buffer, "Concurrency Level:      %d\n", config.concurrency)
	fmt.Fprintf(&buffer, "Time taken for tests:   %.2f seconds\n", totalExecutionTime.Seconds())
	fmt.Fprintf(&buffer, "Complete requests:      %d\n", totalRequests)
	if totalFailedReqeusts == 0 {
		fmt.Fprintln(&buffer, "Failed requests:        0")
	} else {
		fmt.Fprintf(&buffer, "Failed requests:        %d\n", totalFailedReqeusts)
		fmt.Fprintf(&buffer, "   (Connect: %d, Receive: %d, Length: %d, Exceptions: %d)\n", stats.errConnect, stats.errReceive, stats.errLength, stats.errException)
	}
	if stats.errResponse > 0 {
		fmt.Fprintf(&buffer, "Non-2xx responses:      %d\n", stats.errResponse)
	}
	fmt.Fprintf(&buffer, "HTML transferred:       %d bytes\n", totalReceived)

	if len(responseTimeData) > 0 && totalExecutionTime > 0 {
		stdDevOfResponseTime := stdDev(responseTimeData) / 1000000
		sort.Sort(durationSlice(responseTimeData))

		meanOfResponseTime := int64(totalExecutionTime) / int64(totalRequests) / 1000000
		medianOfResponseTime := responseTimeData[len(responseTimeData)/2] / 1000000
		minResponseTime := responseTimeData[0] / 1000000
		maxResponseTime := responseTimeData[len(responseTimeData)-1] / 1000000

		reqPerSec := float64(totalRequests) / totalExecutionTime.Seconds()
		reqMin := float64(totalExecutionTime.Nanoseconds()) / 1000000 / float64(totalRequests)
		reqMax := float64(config.concurrency) * reqMin
		fmt.Fprintf(&notify, "`%.2f req/sec (%.3f ~ %.3f [ms])`\\n", reqPerSec, reqMin, reqMax)

		fmt.Fprintf(&buffer, "Requests per second:    %.2f [#/sec] (mean)\n", reqPerSec)
		fmt.Fprintf(&buffer, "Time per request:       %.3f [ms] (mean)\n", reqMax)
		fmt.Fprintf(&buffer, "Time per request:       %.3f [ms] (mean, across all concurrent requests)\n", reqMin)
		fmt.Fprintf(&buffer, "HTML Transfer rate:     %.2f [Kbytes/sec] received\n\n", float64(totalReceived/1024)/totalExecutionTime.Seconds())

		fmt.Fprint(&buffer, "Connection Times (ms)\n")
		fmt.Fprint(&buffer, "              min\tmean[+/-sd]\tmedian\tmax\n")
		fmt.Fprintf(&buffer, "Total:        %d     \t%d   %.2f \t%d \t%d\n\n",
			minResponseTime,
			meanOfResponseTime,
			stdDevOfResponseTime,
			medianOfResponseTime,
			maxResponseTime)

		fmt.Fprintln(&buffer, "Percentage of the requests served within a certain time (ms)")

		percentages := []int{50, 66, 75, 80, 90, 95, 98, 99}

		for _, percentage := range percentages {
			fmt.Fprintf(&buffer, " %d%%\t %d\n", percentage, responseTimeData[percentage*len(responseTimeData)/100]/1000000)
		}
		fmt.Fprintf(&buffer, " %d%%\t %d (longest request)\n", 100, maxResponseTime)
	}

	fmt.Fprintf(&notify, "estimated : %.2f seconds [ *%d* Connections / *%d (%d)* Transections]", totalExecutionTime.Seconds(), config.concurrency, totalRequests, totalFailedReqeusts)

	// fmt.Println(buffer.String())

	if os.Getenv("NOTIFY") != "" {
		var body bytes.Buffer
		fmt.Fprintf(&body, "{\"message\":\"%s\"}", notify.String())
		req, err := http.NewRequest("PUT", os.Getenv("NOTIFY"), &body)
		req.Header.Add("Content-Type", "application/json")
		if err != nil {
			fmt.Println("request:", err)
		}
		client := &http.Client{}
		res, err := client.Do(req)
		defer res.Body.Close()

		if err != nil {
			fmt.Println("client:", err)
		}

		raw, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Println("Body:", err)
		}

		fmt.Println(string(raw))

	} else {
		fmt.Println("")
		fmt.Println(notify.String())
	}
}

type durationSlice []time.Duration

func (s durationSlice) Len() int           { return len(s) }
func (s durationSlice) Less(i, j int) bool { return s[i] < s[j] }
func (s durationSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// StdDev calculate standard deviation
func stdDev(data []time.Duration) float64 {
	var sum int64
	for _, d := range data {
		sum += int64(d)
	}
	avg := float64(sum / int64(len(data)))

	sumOfSquares := 0.0
	for _, d := range data {

		sumOfSquares += math.Pow(float64(d)-avg, 2)
	}
	return math.Sqrt(sumOfSquares / float64(len(data)))

}
