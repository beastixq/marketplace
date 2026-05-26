package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
)

type result struct {
	Index      int
	StatusCode int
	Bytes      int64
	LatencyMS  float64
	Error      string
}

func main() {
	url := flag.String("url", "", "URL to benchmark")
	label := flag.String("label", "scenario", "scenario label")
	requests := flag.Int("requests", 500, "total request count")
	concurrency := flag.Int("concurrency", 20, "concurrent workers")
	timeout := flag.Duration("timeout", 5*time.Second, "per request timeout")
	csvPath := flag.String("csv", "", "per-request CSV path")
	summaryPath := flag.String("summary", "", "summary CSV path")
	flag.Parse()

	if *url == "" || *csvPath == "" || *summaryPath == "" {
		fmt.Fprintln(os.Stderr, "url, csv, and summary are required")
		os.Exit(2)
	}
	if *requests <= 0 || *concurrency <= 0 {
		fmt.Fprintln(os.Stderr, "requests and concurrency must be positive")
		os.Exit(2)
	}

	client := &http.Client{Timeout: *timeout}
	jobs := make(chan int)
	results := make(chan result, *requests)
	startAll := time.Now()

	var wg sync.WaitGroup
	for worker := 0; worker < *concurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				results <- runRequest(client, *url, idx)
			}
		}()
	}

	go func() {
		for idx := 1; idx <= *requests; idx++ {
			jobs <- idx
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	rows := make([]result, 0, *requests)
	for r := range results {
		rows = append(rows, r)
	}
	elapsed := time.Since(startAll)
	sort.Slice(rows, func(i, j int) bool { return rows[i].Index < rows[j].Index })

	if err := writeDetails(*csvPath, *label, rows); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := appendSummary(*summaryPath, *label, *requests, *concurrency, elapsed, rows); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runRequest(client *http.Client, url string, idx int) result {
	ctx, cancel := context.WithTimeout(context.Background(), client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return result{Index: idx, Error: err.Error()}
	}
	req.Header.Set("Accept", "application/json")

	start := time.Now()
	resp, err := client.Do(req)
	latency := float64(time.Since(start).Microseconds()) / 1000.0
	if err != nil {
		return result{Index: idx, LatencyMS: latency, Error: err.Error()}
	}
	defer resp.Body.Close()

	n, readErr := io.Copy(io.Discard, resp.Body)
	if readErr != nil {
		return result{Index: idx, StatusCode: resp.StatusCode, Bytes: n, LatencyMS: latency, Error: readErr.Error()}
	}
	return result{Index: idx, StatusCode: resp.StatusCode, Bytes: n, LatencyMS: latency}
}

func writeDetails(path string, label string, rows []result) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()
	if err := w.Write([]string{"label", "request", "status_code", "bytes", "latency_ms", "error"}); err != nil {
		return err
	}
	for _, r := range rows {
		if err := w.Write([]string{
			label,
			strconv.Itoa(r.Index),
			strconv.Itoa(r.StatusCode),
			strconv.FormatInt(r.Bytes, 10),
			fmt.Sprintf("%.3f", r.LatencyMS),
			r.Error,
		}); err != nil {
			return err
		}
	}
	return nil
}

func appendSummary(path string, label string, requests int, concurrency int, elapsed time.Duration, rows []result) error {
	exists := false
	if _, err := os.Stat(path); err == nil {
		exists = true
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()
	if !exists {
		if err := w.Write([]string{
			"label",
			"requests",
			"concurrency",
			"success",
			"errors",
			"elapsed_ms",
			"rps",
			"min_ms",
			"avg_ms",
			"p50_ms",
			"p90_ms",
			"p95_ms",
			"p99_ms",
			"max_ms",
		}); err != nil {
			return err
		}
	}

	latencies := make([]float64, 0, len(rows))
	success := 0
	errors := 0
	for _, r := range rows {
		if r.Error == "" && r.StatusCode >= 200 && r.StatusCode < 300 {
			success++
			latencies = append(latencies, r.LatencyMS)
		} else {
			errors++
		}
	}
	sort.Float64s(latencies)
	if len(latencies) == 0 {
		latencies = []float64{0}
	}
	avg := 0.0
	for _, v := range latencies {
		avg += v
	}
	avg /= float64(len(latencies))

	elapsedMS := float64(elapsed.Microseconds()) / 1000.0
	rps := float64(success) / elapsed.Seconds()
	return w.Write([]string{
		label,
		strconv.Itoa(requests),
		strconv.Itoa(concurrency),
		strconv.Itoa(success),
		strconv.Itoa(errors),
		fmt.Sprintf("%.3f", elapsedMS),
		fmt.Sprintf("%.2f", rps),
		fmt.Sprintf("%.3f", latencies[0]),
		fmt.Sprintf("%.3f", avg),
		fmt.Sprintf("%.3f", percentile(latencies, 0.50)),
		fmt.Sprintf("%.3f", percentile(latencies, 0.90)),
		fmt.Sprintf("%.3f", percentile(latencies, 0.95)),
		fmt.Sprintf("%.3f", percentile(latencies, 0.99)),
		fmt.Sprintf("%.3f", latencies[len(latencies)-1]),
	})
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	pos := p * float64(len(sorted)-1)
	lower := int(pos)
	upper := lower + 1
	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	weight := pos - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}
