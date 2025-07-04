package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"

	"github.com/shivanshkc/llmb/pkg/api"
)

// chatCmd represents the `chat` command, providing an interactive, REPL-style
// interface for conversing with a language model.
//
// It maintains a persistent chat history for the session, allowing for
// follow-up questions. It also gracefully handles interruptions (like Ctrl+C)
// at any point, including while waiting for user input.
var chatCmd = &cobra.Command{
	Use:     "chat",
	Short:   "Start an interactive chat with the LLM.",
	Long:    "Starts an interactive chat session with the specified language model, maintaining conversation history.",
	PreRunE: func(cmd *cobra.Command, args []string) error { return validateChatFlags() },
	RunE: func(cmd *cobra.Command, args []string) error {
		// chatMessages holds the full conversation history for the current session.
		var chatMessages []api.ChatMessage
		client := api.NewClient(rootBaseURL)
		reader := bufio.NewReader(os.Stdin)

		// The main chat loop.
		for {
			fmt.Print(text.FgBlue.Sprint("You: "))

			// Read user input with context-awareness. This call will unblock and
			// return an error if the command's context is canceled (e.g., by Ctrl+C).
			input, err := readStringContext(cmd.Context(), reader)
			if err != nil {
				// Ignore context cancellation errors.
				if errors.Is(err, context.Canceled) {
					return nil
				}
				return fmt.Errorf("failed to read input: %w", err)
			}

			// Parse the raw input into a role and message content.
			role, message := parseInput(input)
			if message == "" {
				continue // Ignore empty inputs.
			}

			// Add the user's input to the chat history.
			chatMessages = append(chatMessages, api.ChatMessage{Role: role, Content: message})

			// Begin the streaming API call.
			eventStream, err := client.ChatCompletionStream(cmd.Context(), rootModel, chatMessages)
			if err != nil {
				// End if the context was canceled, otherwise log the error and continue chat.
				if errors.Is(err, context.Canceled) {
					return nil
				}
				fmt.Println("Failed to stream response:", err)
				// Don't consider this message since the call failed.
				chatMessages = chatMessages[:len(chatMessages)-1]
				continue
			}

			// Consume the response stream token-by-token.
			fmt.Print(text.FgGreen.Sprint("Assistant: "))
			var answer string
			for {
				event, ok, err := eventStream.NextContext(cmd.Context())
				if err != nil {
					return nil // Context canceled.
				}

				// Stream ended.
				if !ok {
					break
				}

				if len(event.Choices) > 0 {
					token := event.Choices[0].Delta.Content
					answer += token
					fmt.Print(token)
				}
			}
			fmt.Println("") // Newline after the full response.

			// Add the assistant's complete response to the chat history.
			chatMessages = append(chatMessages, api.ChatMessage{Role: api.RoleAssistant, Content: answer})
		}
	},
}

func init() {
	rootCmd.AddCommand(chatCmd)
}

// readStringContext reads a line of text from a Reader but aborts early
// if the provided context is canceled. This is essential for making the
// blocking read from os.Stdin responsive to interruptions like Ctrl+C.
//
// This pattern is the standard Go idiom for making a synchronous, blocking call
// cancellable. It works by wrapping the blocking call in a goroutine and racing
// its result against the context's Done channel.
// A known trade-off is that if the context is canceled, the producer goroutine
// will remain blocked on `reader.ReadString` until the read completes, but this
// goroutine leak is temporary and harmless for a CLI application.
func readStringContext(ctx context.Context, reader *bufio.Reader) (string, error) {
	// A struct to hold the result of the I/O operation.
	type readResult struct {
		input string
		err   error
	}

	// This buffered channel of size 1 is crucial. It holds the result and
	// prevents the producer goroutine from leaking by ensuring its send will
	// complete even if the consumer has already returned due to cancellation.
	resultChan := make(chan readResult, 1)

	// Launch a goroutine to perform the blocking read.
	go func() {
		input, err := reader.ReadString('\n')
		resultChan <- readResult{input: input, err: err}
	}()

	// Race the read operation against context cancellation.
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case result := <-resultChan:
		return result.input, result.err
	}
}

// parseInput sanitizes raw user input and parses it to determine the message
// content and the intended role (system, user, or assistant).
// If no role prefix (e.g., "system:") is found, it defaults to the "user" role.
func parseInput(input string) (role, message string) {
	message = strings.TrimSpace(input)
	if message == "" {
		return "", ""
	}

	const (
		systemPrefix    = api.RoleSystem + ":"
		assistantPrefix = api.RoleAssistant + ":"
		userPrefix      = api.RoleUser + ":"
	)

	if strings.HasPrefix(strings.ToLower(message), systemPrefix) {
		return api.RoleSystem, strings.TrimSpace(message[len(systemPrefix):])
	}
	if strings.HasPrefix(strings.ToLower(message), assistantPrefix) {
		return api.RoleAssistant, strings.TrimSpace(message[len(assistantPrefix):])
	}
	if strings.HasPrefix(strings.ToLower(message), userPrefix) {
		return api.RoleUser, strings.TrimSpace(message[len(userPrefix):])
	}

	// Default to the user role if no prefix is provided.
	return api.RoleUser, message
}
