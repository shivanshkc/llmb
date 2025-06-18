package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// chatCmd represents the chat command
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start a chat with the LLM.",
	Long:  "Start a chat with the LLM.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("chat called")
	},
}

func init() {
	rootCmd.AddCommand(chatCmd)

	chatCmd.Flags().StringP("base-url", "u",
		"http://localhost:8080", "Base URL of the API.")
}
