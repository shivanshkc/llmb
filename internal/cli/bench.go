package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// benchCmd represents the bench command
var benchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Benchmark an Open AI compatible REST API.",
	Long:  "Benchmark an Open AI compatible REST API.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("bench called")
	},
}

func init() {
	rootCmd.AddCommand(benchCmd)

	benchCmd.Flags().StringP("base-url", "u",
		"http://localhost:8080", "Base URL of the API.")

	benchCmd.Flags().StringP("prompt", "p",
		"", "Prompt to use.")

	benchCmd.Flags().IntP("request-count", "n",
		12, "Number of requests to perform.")

	benchCmd.Flags().IntP("concurrency", "c",
		3, "Number of multiple requests to make at a time.")
}
