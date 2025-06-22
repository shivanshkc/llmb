package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/shivanshkc/llmb/pkg/api"
	"github.com/shivanshkc/llmb/pkg/bench"
	"github.com/shivanshkc/llmb/pkg/streams"
)

var (
	benchPrompt       string
	benchRequestCount int
	benchConcurrency  int
)

// benchCmd represents the `bench` command for running performance benchmarks
// against an OpenAI-compatible API.
//
// This command acts as an orchestrator: it sets up the client and the function
// to be benchmarked, then delegates all concurrent execution and metric calculation
// to the `pkg/bench` package. Finally, it formats and displays the results.
//
// This command leverages persistent flags (`--base-url`, `--model`)
// defined on the root command for shared configuration.
var benchCmd = &cobra.Command{
	Use:     "bench",
	Short:   "Benchmark an Open AI compatible REST API.",
	Long:    "Concurrently executes requests against a streaming API and reports performance metrics.",
	PreRunE: func(cmd *cobra.Command, args []string) error { return validateBenchFlags() },
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient(rootBaseURL)

		// streamFunc is the core function to be benchmarked. It's a factory that
		// captures user flags and creates a cancellable API stream each time it's
		// called by the benchmark runner.
		//
		// This closure is a clean "adapter" between the CLI layer and the reusable
		// benchmark package. It adapts the specific `api.ChatCompletionEvent`
		// stream into the generic `bench.Event` stream required by the runner.
		streamFunc := func(ctx context.Context) (*streams.Stream[bench.Event], error) {
			messages := []api.ChatMessage{{Role: api.RoleUser, Content: benchPrompt}}
			cceStream, err := client.ChatCompletionStream(ctx, rootModel, messages)
			if err != nil {
				return nil, fmt.Errorf("error in ChatCompletionStream call: %w", err)
			}
			// Adapt the concrete event type to the generic benchmark interface.
			return streams.Map(cceStream, func(e api.ChatCompletionEvent) bench.Event { return e }), nil
		}

		// Delegate all concurrent execution and aggregation to the benchmark package.
		results, err := bench.BenchmarkStream(cmd.Context(), benchRequestCount, benchConcurrency, streamFunc)
		if err != nil {
			fmt.Println("Error during benchmarking:", err)
			os.Exit(1)
		}

		displayBenchmarkResults(results)
	},
}

// init registers the bench command with the root command and defines its local flags.
func init() {
	rootCmd.AddCommand(benchCmd)

	benchCmd.Flags().StringVarP(&benchPrompt, "prompt", "p",
		"", "Prompt to use for all requests.")

	benchCmd.Flags().IntVarP(&benchRequestCount, "request-count", "n",
		12, "Total number of requests to perform.")

	benchCmd.Flags().IntVarP(&benchConcurrency, "concurrency", "c",
		3, "Number of multiple requests to make at a time.")
}

// displayBenchmarkResults formats and prints the given benchmark results in a
// human-readable table to standard output.
//
// Using a dedicated table library like `go-pretty/table` provides a
// professional and easy-to-read output for CLI tools.
func displayBenchmarkResults(results bench.StreamBenchmarkResults) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleColoredDark)

	t.AppendHeader(table.Row{"Metric", "Average", "Minimum", "Median", "Maximum", "P90", "P95"})

	// Shorthand.
	fd := formatDuration

	// AppendRows is formatted vertically to adhere to the line length limit
	// and improve readability.
	t.AppendRows([]table.Row{
		{
			"Time To First Token (TTFT)",
			fd(results.TTFT.Avg),
			fd(results.TTFT.Min),
			fd(results.TTFT.Med),
			fd(results.TTFT.Max),
			fd(results.TTFT.P90),
			fd(results.TTFT.P95),
		},
		{
			"Time Between Tokens (TBT)",
			fd(results.TBT.Avg),
			fd(results.TBT.Min),
			fd(results.TBT.Med),
			fd(results.TBT.Max),
			fd(results.TBT.P90),
			fd(results.TBT.P95),
		},
		{
			"Total Time (TT)",
			fd(results.TT.Avg),
			fd(results.TT.Min),
			fd(results.TT.Med),
			fd(results.TT.Max),
			fd(results.TT.P90),
			fd(results.TT.P95),
		},
	})

	fmt.Println()
	t.Render()
	fmt.Println()
}

// FormatDuration formats a time.Duration into a human-readable string with an
// appropriate unit (ns, μs, ms, or s).
//
// This function is designed to produce concise, readable output for display in
// user interfaces like tables or logs, where the default `time.Duration.String()`
// method (e.g., "1m23.456s") might be too verbose or precise.
//
// The unit is chosen based on the duration's magnitude:
//   - Less than 1 microsecond: formatted in whole nanoseconds (e.g., "750ns").
//   - Less than 1 millisecond: formatted in microseconds with 2 decimal places (e.g., "123.45μs").
//   - Less than 1 second: formatted in milliseconds with 2 decimal places (e.g., "89.12ms").
//   - 1 second or more: formatted in seconds with 2 decimal places (e.g., "5.78s").
//
// A zero duration is formatted as "0s".
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	// Format based on magnitude.
	switch {
	case d < time.Microsecond:
		return fmt.Sprintf("%.0fns", float64(d.Nanoseconds()))
	case d < time.Millisecond:
		return fmt.Sprintf("%.2fμs", float64(d.Nanoseconds())/1000)
	case d < time.Second:
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1000000)
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}
