package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "llmb",
	Short: "A tool to interact with and benchmark Open AI compatible REST APIs.",
	Long:  `A tool to interact with and benchmark Open AI compatible REST APIs.`,
	// Uncomment the following line if your bare application has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
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
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.llmb.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
