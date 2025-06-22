package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/shivanshkc/llmb/pkg/api"
	"github.com/shivanshkc/llmb/pkg/bench"
	"github.com/shivanshkc/llmb/pkg/streams"
	"github.com/shivanshkc/llmb/pkg/utils"
)

var (
	benchBaseURL, benchModel, benchPrompt *string
	benchRequestCount, benchConcurrency   *int
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
		streamFunc := func(ctx context.Context) (*streams.Stream[bench.Event], error) {
			// Single message chat.
			messages := []api.ChatMessage{{Role: api.RoleUser, Content: *benchPrompt}}
			// Get the stream.
			cceStream, err := client.ChatCompletionStream(ctx, *benchModel, messages)
			if err != nil {
				return nil, fmt.Errorf("error in ChatCompletionStream call: %w", err)
			}
			// Convert to compatible channel type and return.
			return streams.Map(cceStream, func(x api.ChatCompletionEvent) bench.Event { return x }), nil
		}

		// Run benchmark.
		results, err := bench.BenchmarkStream(cmd.Context(), *benchRequestCount, *benchConcurrency, streamFunc)
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

	benchModel = benchCmd.Flags().StringP("model", "m",
		"gpt-4.1", "Name of the model to use.")

	benchPrompt = benchCmd.Flags().StringP("prompt", "p",
		"", "Prompt to use.")

	benchRequestCount = benchCmd.Flags().IntP("request-count", "n",
		12, "Number of requests to perform.")

	benchConcurrency = benchCmd.Flags().IntP("concurrency", "c",
		3, "Number of multiple requests to make at a time.")
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
	fd := utils.FormatDuration

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
