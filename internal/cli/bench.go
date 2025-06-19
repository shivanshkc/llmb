package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/shivanshkc/llmb/pkg/api"
	"github.com/shivanshkc/llmb/pkg/bench"
	"github.com/shivanshkc/llmb/pkg/utils/miscutils"
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
			// Single message chat.
			messages := []api.ChatMessage{{Role: api.RoleUser, Content: *benchPrompt}}
			// Get the stream.
			cceChan, err := client.ChatCompletionStream(context.TODO(), messages)
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

// displayBenchmarkResults prints the given results in a human-readable format.
func displayBenchmarkResults(results bench.StreamBenchmarkResults) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	// Set style
	t.SetStyle(table.StyleColoredDark)
	// Header
	t.AppendHeader(table.Row{"Metric", "Average", "Minimum", "Median", "Maximum", "P90", "P95"})
	// Shorthand.
	fd := miscutils.FormatDuration

	// Add rows
	t.AppendRows([]table.Row{
		{"TTFT", fd(results.TTFT.Avg), fd(results.TTFT.Min),
			fd(results.TTFT.Med), fd(results.TTFT.Max),
			fd(results.TTFT.P90), fd(results.TTFT.P95)},
		{"TBT", fd(results.TBT.Avg), fd(results.TBT.Min),
			fd(results.TBT.Med), fd(results.TBT.Max),
			fd(results.TBT.P90), fd(results.TBT.P95)},
		{"TT", fd(results.TT.Avg), fd(results.TT.Min),
			fd(results.TT.Med), fd(results.TT.Max),
			fd(results.TT.P90), fd(results.TT.P95)},
	})

	// Render
	fmt.Println()
	t.Render()
	fmt.Println()
}
