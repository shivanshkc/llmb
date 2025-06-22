// Package cli contains all the command-line interface logic for the application,
// powered by the cobra library. It defines the root command, subcommands,
// and their respective flags.
package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	// rootBaseURL and rootModel hold the values from the root command's persistent flags.
	// Defining them at the package level allows all subcommands within this
	// package (like `chat` and `bench`) to access these shared values directly and safely.
	rootBaseURL string
	rootModel   string
)

// rootCmd represents the base command when called without any subcommands.
// It serves as the entry point and parent for all other commands.
var rootCmd = &cobra.Command{
	Use:   "llmb",
	Short: "A tool to interact with and benchmark Open AI compatible REST APIs.",
	Long: `A tool to interact with and benchmark Open AI compatible REST APIs.
This CLI provides subcommands for interactive chat sessions and performance benchmarking.`,
}

// Execute is the primary entry point for the CLI application, called by main.go.
//
// It sets up a single, root cancellable context and wires it up to respond
// to OS interruption signals (like Ctrl+C or SIGTERM). This context is then passed down
// to all cobra commands, enabling graceful shutdown across the entire application.
func Execute() error {
	// Create a root context that can be canceled.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cancel is called on exit to clean up context resources.

	// Set up a channel to listen for specific OS signals.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	// Unregister the signal handler on exit. This is good hygiene and
	// prevents resource leaks in more complex application lifecycles.
	defer signal.Stop(signals)

	// Launch a goroutine to cancel the context upon receiving a signal.
	go func() {
		<-signals
		cancel()
	}()

	// Execute the root command with the cancellable context.
	return rootCmd.ExecuteContext(ctx)
}

// init configures the application's flags.
//
// Using `PersistentFlags` on the root command is the ideal way to handle
// flags that are shared across multiple subcommands, like configuration settings.
// This avoids code duplication and provides a consistent user experience.
func init() {
	rootCmd.PersistentFlags().StringVarP(&rootBaseURL, "base-url", "u",
		"http://localhost:8080", "Base URL of the API.")

	rootCmd.PersistentFlags().StringVarP(&rootModel, "model", "m",
		"gpt-4.1", "Name of the model to use.")
}
