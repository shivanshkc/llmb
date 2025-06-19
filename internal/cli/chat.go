package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"

	"github.com/shivanshkc/llmb/pkg/api"
)

var (
	chatBaseURL, chatModel *string
)

// chatCmd represents the chat command
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start a chat with the LLM.",
	Long:  "Start a chat with the LLM.",
	Run: func(cmd *cobra.Command, args []string) {
		// Validate flags before using them.
		if message := validateChatFlags(); message != "" {
			fmt.Println(message)
			os.Exit(1)
		}

		// List of all messages in the chat.
		var chatMessages []api.ChatMessage
		// Client to interact with the LLM.
		client := api.NewClient(*chatBaseURL)

		// Stdin reader to read user's input.
		reader := bufio.NewReader(os.Stdin)

		for {
			// Prompt user for input.
			fmt.Print(text.FgBlue.Sprint("You: "))

			// Read user input.
			input, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error reading input:", err)
				break
			}

			// Ignore empty inputs.
			if input = strings.TrimSpace(input); input == "" {
				continue
			}

			// Update chat with the user's message.
			chatMessages = append(chatMessages, api.ChatMessage{Role: api.RoleUser, Content: input})

			// Start LLM response stream.
			eventChan, err := client.ChatCompletionStream(context.TODO(), *chatModel, chatMessages)
			if err != nil {
				fmt.Println("Error streaming response:", err)
				continue
			}

			// Start showing assistant's response.
			fmt.Print(text.FgGreen.Sprint("Assistant: "))

			var answer string
			// Display streaming response and collect it to update chat.
			for event := range eventChan {
				for _, choice := range event.Choices {
					if choice.Delta.Content != "" {
						answer += choice.Delta.Content
						fmt.Print(choice.Delta.Content)
					}
				}
			}

			// Newline after assistant's response.
			fmt.Println("")

			// Update chat with the assistant's message.
			chatMessages = append(chatMessages, api.ChatMessage{Role: api.RoleAssistant, Content: answer})
		}
	},
}

func init() {
	rootCmd.AddCommand(chatCmd)

	chatBaseURL = chatCmd.Flags().StringP("base-url", "u",
		"http://localhost:8080", "Base URL of the API.")

	chatModel = chatCmd.Flags().StringP("model", "m",
		"gpt-4.1", "Name of the model to use.")
}
