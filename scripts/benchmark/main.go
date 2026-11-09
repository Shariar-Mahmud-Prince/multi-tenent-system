package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	targetURL := flag.String("url", "http://localhost:80/ ", "Target URL for load testing")
	hostHeader := flag.String("host", "ecom.local", "Host header for multi-tenant resolution")
	concurrency := flag.Int("c", 25, "Number of concurrent workers")
	totalReqs := flag.Int("n", 2000, "Total number of HTTP requests to send")
	flag.Parse()

	trimmedURL := strings.TrimSpace(*targetURL)

	fmt.Println("=================================================================")
	fmt.Printf("   🚀 MULTI-TENANT LOAD & STRESS TEST BENCHMARK\n")
	fmt.Println("=================================================================")
	fmt.Printf(" Target URL     : %s\n", trimmedURL)
	fmt.Printf(" Host Header    : %s\n", *hostHeader)
	fmt.Printf(" Concurrency    : %d workers\n", *concurrency)
	fmt.Printf(" Total Requests : %d requests\n", *totalReqs)
	fmt.Println("-----------------------------------------------------------------")

	// Check container stats before test
	fmt.Println("📊 Initial Docker Resource Usage:")
	printDockerStats()
	fmt.Println("-----------------------------------------------------------------")

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second,
	}

	var successfulReqs int64
	var failedReqs int64
	var statusCounts sync.Map

	latencies := make([]time.Duration, *totalReqs)
	var latencyIndex int64

	jobs := make(chan int, *totalReqs)
	for i := 0; i < *totalReqs; i++ {
		jobs <- i
	}
	close(jobs)

	startTime := time.Now()

	var wg sync.WaitGroup
	for w := 0; w < *concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				req, err := http.NewRequest(http.MethodGet, trimmedURL, nil)
				if err != nil {
					atomic.AddInt64(&failedReqs, 1)
					continue
				}
				if *hostHeader != "" {
					req.Host = *hostHeader
				}
				req.Header.Set("User-Agent", "LoadTestBenchmark/1.0")

				t0 := time.Now()
				resp, err := client.Do(req)
				dur := time.Since(t0)

				idx := atomic.AddInt64(&latencyIndex, 1) - 1
				if idx < int64(*totalReqs) {
					latencies[idx] = dur
				}

				if err != nil {
					atomic.AddInt64(&failedReqs, 1)
					continue
				}

				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				statusKey := resp.StatusCode
				val, _ := statusCounts.LoadOrStore(statusKey, new(int64))
				atomic.AddInt64(val.(*int64), 1)

				if resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&successfulReqs, 1)
				} else {
					atomic.AddInt64(&failedReqs, 1)
				}
			}
		}()
	}

	wg.Wait()
	totalElapsed := time.Since(startTime)

	// Sort latencies to compute percentiles
	validCount := int(latencyIndex)
	validLatencies := latencies[:validCount]
	sort.Slice(validLatencies, func(i, j int) bool {
		return validLatencies[i] < validLatencies[j]
	})

	rps := float64(validCount) / totalElapsed.Seconds()

	var p50, p95, p99, minDur, maxDur time.Duration
	if len(validLatencies) > 0 {
		minDur = validLatencies[0]
		maxDur = validLatencies[len(validLatencies)-1]
		p50 = validLatencies[int(float64(len(validLatencies))*0.50)]
		p95 = validLatencies[int(float64(len(validLatencies))*0.95)]
		p99 = validLatencies[int(float64(len(validLatencies))*0.99)]
	}

	fmt.Println("\n📈 TEST RESULTS & LATENCY PROFILE:")
	fmt.Printf(" Elapsed Time        : %v\n", totalElapsed)
	fmt.Printf(" Throughput (RPS)    : %.2f requests/sec\n", rps)
	fmt.Printf(" Successful (200 OK) : %d (%.1f%%)\n", successfulReqs, float64(successfulReqs)/float64(validCount)*100)
	fmt.Printf(" Other/Failed        : %d (%.1f%%)\n", failedReqs, float64(failedReqs)/float64(validCount)*100)
	fmt.Printf(" Latency Min         : %v\n", minDur)
	fmt.Printf(" Latency Avg (P50)   : %v\n", p50)
	fmt.Printf(" Latency P95         : %v\n", p95)
	fmt.Printf(" Latency P99         : %v\n", p99)
	fmt.Printf(" Latency Max         : %v\n", maxDur)

	fmt.Println("\nStatus Code Distribution:")
	statusCounts.Range(func(key, value interface{}) bool {
		code := key.(int)
		count := *(value.(*int64))
		fmt.Printf("   HTTP %d: %d\n", code, count)
		return true
	})

	fmt.Println("\n📊 Post-Test Docker Resource Usage:")
	printDockerStats()
	fmt.Println("=================================================================")
}

func printDockerStats() {
	cmd := exec.Command("docker", "stats", "--no-stream", "--format", "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}")
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("  (docker stats unavailable: %v)\n", err)
		return
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "tenant_") || strings.Contains(line, "NAME") {
			fmt.Println("  " + line)
		}
	}
}
