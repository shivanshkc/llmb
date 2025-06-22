package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	rootBaseURL, rootModel string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "llmb",
	Short: "A tool to interact with and benchmark Open AI compatible REST APIs.",
	Long:  `A tool to interact with and benchmark Open AI compatible REST APIs.`,
}

// Execute executes the root command.
func Execute() error {
	// Cancellable context to handle interruptions.
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// Listen to interruption signals.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	// Cancel the context upon interruption.
	go func() {
		<-signals
		cancelFunc()
	}()

	return rootCmd.ExecuteContext(ctx)
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&rootBaseURL, "base-url", "u",
		"http://localhost:8080", "Base URL of the API.")

	rootCmd.PersistentFlags().StringVarP(&rootModel, "model", "m",
		"gpt-4.1", "Name of the model to use.")
}
