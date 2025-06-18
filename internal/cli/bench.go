package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/shivanshkc/llmb/pkg/api"
	"github.com/shivanshkc/llmb/pkg/bench"
)

var (
	benchBaseURL, benchPrompt           *string
	benchRequestCount, benchConcurrency *int
)

// benchCmd represents the bench command
var benchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Benchmark an Open AI compatible REST API.",
	Long:  "Benchmark an Open AI compatible REST API.",
	Run: func(cmd *cobra.Command, args []string) {
		// Validate flags before using them.
		if message := validateBenchFlags(); message != "" {
			fmt.Println(message)
			os.Exit(1)
		}

		// Client for the LLM REST API.
		client := api.NewClient(*benchBaseURL)

		// Benchmark-able function.
		streamFunc := func() (<-chan bench.Event, error) {
			// Get the stream.
			cceChan, err := client.ChatCompletionStream(context.TODO(), *benchPrompt)
			if err != nil {
				return nil, fmt.Errorf("error in ChatCompletionStream call: %w", err)
			}
			// Convert to compatible channel type and return.
			return convertEventChannel(cceChan), nil
		}

		// Run benchmark.
		results, err := bench.BenchmarkStream(*benchRequestCount, *benchConcurrency, streamFunc)
		if err != nil {
			fmt.Println("Error in benchmarking:", err)
			os.Exit(1)
		}

		// Display to caller.
		displayBenchmarkResults(results)
	},
}

func init() {
	rootCmd.AddCommand(benchCmd)

	benchBaseURL = benchCmd.Flags().StringP("base-url", "u",
		"http://localhost:8080", "Base URL of the API.")

	benchPrompt = benchCmd.Flags().StringP("prompt", "p",
		"", "Prompt to use.")

	benchRequestCount = benchCmd.Flags().IntP("request-count", "n",
		12, "Number of requests to perform.")

	benchConcurrency = benchCmd.Flags().IntP("concurrency", "c",
		3, "Number of multiple requests to make at a time.")
}

// convertEventChannel essentially converts "<-chan implementation" to "<-chan interface".
//
// While the `api.ChatCompletionEvent` type implements the `bench.Event` interface,
// it doesn't mean that `chan api.ChatCompletionEvent` is the same as `chan bench.Event`.
// So, this conversion has to be manual.
func convertEventChannel(cceChan <-chan api.ChatCompletionEvent) <-chan bench.Event {
	benchEventChan := make(chan bench.Event, 100)

	// Pipe without blocking.
	go func() {
		// Both channels close together.
		defer close(benchEventChan)
		for event := range cceChan {
			benchEventChan <- event
		}
	}()

	return benchEventChan
}

// displayBenchmarkResults prints the given results to stdout in a human-readable format.
func displayBenchmarkResults(results bench.StreamBenchmarkResults) {
	// Tab writer for clean output.
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintln(w, "--------------------------")
	// Write some data to the Writer.
	fmt.Fprintln(w, "Metric\tAvg\tMin\tMed\tMax\tP90\tP95")

	fmt.Fprintf(w, "TTFT\t%v\t%v\t%v\t%v\t%v\t%v\n",
		results.TTFT.Avg, results.TTFT.Min, results.TTFT.Med, results.TTFT.Max, results.TTFT.P90, results.TTFT.P95)

	fmt.Fprintf(w, "TBT\t%v\t%v\t%v\t%v\t%v\t%v\n",
		results.TBT.Avg, results.TBT.Min, results.TBT.Med, results.TBT.Max, results.TBT.P90, results.TBT.P95)

	fmt.Fprintf(w, "TT\t%v\t%v\t%v\t%v\t%v\t%v\n",
		results.TT.Avg, results.TT.Min, results.TT.Med, results.TT.Max, results.TT.P90, results.TT.P95)

	// Flush the Writer to ensure all data is written to the output.
	w.Flush()
}
