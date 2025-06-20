package cli

import (
	"bufio"
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
			// Respect context expiry.
			select {
			case <-cmd.Context().Done():
				return
			default:
			}

			// Prompt user for input.
			fmt.Print(text.FgBlue.Sprint("You: "))

			// Read and parse user input.
			role, message, err := readChatInput(reader)
			if err != nil {
				fmt.Println("Failed to read input:", err)
				continue
			}

			// Ignore empty inputs.
			if message == "" {
				continue
			}

			// Update chat with the user's message.
			chatMessages = append(chatMessages, api.ChatMessage{Role: role, Content: message})

			// Start LLM response stream.
			eventChan, err := client.ChatCompletionStream(cmd.Context(), *chatModel, chatMessages)
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

// readChatInput reads the user's input from stdin for the chat command.
//
// It allows the user to assume any role, system, user or assistant.
//
// If the user input starts with a role and colon, like: "system: Hello" or "assistant: Hello", then the mentioned
// role is used to communicate with the LLM. Role matching is case-insensitive.
func readChatInput(reader *bufio.Reader) (string, string, error) {
	// Read user input.
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", "", fmt.Errorf("error reading input: %w", err)
	}

	// Ignore empty inputs.
	if input = strings.TrimSpace(input); input == "" {
		return "", "", nil
	}

	const systemPrefix, assistantPrefix, userPrefix = api.RoleSystem + ":", api.RoleAssistant + ":", api.RoleUser + ":"

	// Respect system role if provided.
	if len(input) >= len(systemPrefix) && strings.EqualFold(input[:len(systemPrefix)], systemPrefix) {
		return api.RoleSystem, strings.TrimSpace(input[len(systemPrefix):]), nil
	}

	// Respect assistant role if provided.
	if len(input) >= len(assistantPrefix) && strings.EqualFold(input[:len(assistantPrefix)], assistantPrefix) {
		return api.RoleAssistant, strings.TrimSpace(input[len(assistantPrefix):]), nil
	}

	// Respect user role if provided.
	if len(input) >= len(userPrefix) && strings.EqualFold(input[:len(userPrefix)], userPrefix) {
		return api.RoleUser, strings.TrimSpace(input[len(userPrefix):]), nil
	}

	// Could be unknown role or no role. Assume default role.
	return api.RoleUser, input, nil
}
