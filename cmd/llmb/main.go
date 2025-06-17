package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/shivanshkc/llmb/pkg/api"
	"github.com/shivanshkc/llmb/pkg/bench"
)

// Input errors.
var (
	errTotal       = errors.New("total number of requests (-t) must be greater than zero")
	errConcurrency = errors.New("number of concurrent requests (-c) must be greater than zero")
	errPrompt      = errors.New("prompt (-p) must not be empty")
)

func main() {
	// Inputs.
	total, concurrency, prompt := defineAndParseFlags()
	if err := validateFlags(total, concurrency, prompt); err != nil {
		fmt.Println("ERROR:", err.Error())
		flag.Usage()
		os.Exit(1)
	}

	// Create the API client for the LLM API.
	client := api.NewClient("http://localhost:8080")
	//printStream(client.ChatCompletionStream(context.Background(), *prompt))
	//return

	// Benchmark-able function.
	streamFunc := func() (<-chan api.ChatCompletionEvent, error) {
		return client.ChatCompletionStream(context.Background(), *prompt)
	}

	// Run benchmarks.
	resp, err := bench.BenchmarkStream(*total, *concurrency, streamFunc)
	if err != nil {
		fmt.Println("ERROR:", err.Error())
		os.Exit(1)
	}

	// Tab writer for clean output.
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintln(w, "--------------------------")
	// Write some data to the Writer.
	fmt.Fprintln(w, "Metric\tAvg\tMin\tMed\tMax\tP90\tP95")

	fmt.Fprintf(w, "TFTT\t%v\t%v\t%v\t%v\t%v\t%v\n",
		resp.TTFT.Average(), resp.TTFT.Minimum(),
		resp.TTFT.Median(), resp.TTFT.Maximum(),
		resp.TTFT.Percentile(90), resp.TTFT.Percentile(95))

	fmt.Fprintf(w, "TBT\t%v\t%v\t%v\t%v\t%v\t%v\n",
		resp.TBT.Average(), resp.TBT.Minimum(),
		resp.TBT.Median(), resp.TBT.Maximum(),
		resp.TBT.Percentile(90), resp.TBT.Percentile(95))

	fmt.Fprintf(w, "TT\t%v\t%v\t%v\t%v\t%v\t%v\n",
		resp.TT.Average(), resp.TT.Minimum(),
		resp.TT.Median(), resp.TT.Maximum(),
		resp.TT.Percentile(90), resp.TT.Percentile(95))

	// Flush the Writer to ensure all data is written to the output.
	w.Flush()
}

// defineAndParseFlags defines all flags required by the tool,
// calls flag.Parse and returns the flag variable pointers.
func defineAndParseFlags() (*int, *int, *string) {
	// Flag definitions.
	total := flag.Int("t", 1, "Total number of requests (must be > 0)")
	concurrency := flag.Int("c", 1, "Number of concurrent requests (must be > 0)")
	prompt := flag.String("p", "", "The prompt string (must not be empty)")

	flag.Parse()
	return total, concurrency, prompt
}

// validateFlags ...
func validateFlags(total, concurrency *int, prompt *string) error {
	// At least 1 request required.
	if *total <= 0 {
		return errTotal
	}
	// At least 1 request should be executed concurrently.
	if *concurrency <= 0 {
		return errConcurrency
	}
	// Check if -p is a non-empty string.
	if *prompt == "" {
		return errPrompt
	}

	return nil
}

// printStream prints the given stream to stdout.
func printStream(stream <-chan api.ChatCompletionEvent, err error) {
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	for event := range stream {
		if event.Error != nil {
			fmt.Printf("<error>%s</error>", event.Error.Error())
			continue
		}
		fmt.Print(event.Choices[0].Delta.Content)
	}
}
