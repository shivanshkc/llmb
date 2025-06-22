# llmb: A High-Performance Go CLI for Language Models

[![Go Report Card](https://goreportcard.com/badge/github.com/shivanshkc/llmb)](https://goreportcard.com/report/github.com/shivanshkc/llmb)

`llmb` is a robust, modern command-line tool built with Go for interacting with and benchmarking OpenAI-compatible streaming language model APIs. It provides two primary commands:

*   **`chat`**: An interactive, REPL-style chat session that maintains conversation history and supports streaming responses.
*   **`bench`**: A powerful benchmarking tool that executes concurrent requests against a streaming API and reports detailed performance metrics like Time To First Token (TTFT), Time Between Tokens (TBT), and Total Time.

The entire application is designed with performance, robustness, and a clean user experience as first principles.

## Demo

<details>
<summary>Click to see a demo of the <code>chat</code> and <code>bench</code> commands in action.</summary>

```text
$ llmb chat --base-url http://localhost:8080 --model gpt-4
You: Hello! Who are you?
Assistant: I am a large language model trained by Google.

You: system: You are a pirate who says "Ahoy!" a lot.
You: What is Go?
Assistant: Ahoy! Go, also known as Golang, is a statically typed, compiled programming language designed at Google. Ahoy! It's known for its simplicity, efficiency, and strong support for concurrent programming.

$ llmb bench -u http://localhost:8080 -m gpt-4 -n 20 -c 5 -p "write a haiku about servers"
[1/20] requests complete.
[2/20] requests complete.
...
[19/20] requests complete.
[20/20] requests complete.

+-----------------------------+---------+---------+---------+---------+---------+---------+
| METRIC                      | AVERAGE | MINIMUM | MEDIAN  | MAXIMUM | P90     | P95     |
+-----------------------------+---------+---------+---------+---------+---------+---------+
| Time To First Token (TTFT)  | 251.48ms| 180.12ms| 245.89ms| 350.67ms| 320.11ms| 341.55ms|
| Time Between Tokens (TBT)   | 45.88ms | 20.45ms | 46.12ms | 90.33ms | 75.90ms | 82.14ms |
| Total Time (TT)             | 2.15s   | 1.88s   | 2.12s   | 2.54s   | 2.48s   | 2.51s   |
+-----------------------------+---------+---------+---------+---------+---------+---------+

```

</details>

## Features

*   **Interactive Chat**: Start a conversation with any model. Chat history is maintained throughout the session.
*   **Concurrent Benchmarking**: Stress-test your API with a configurable number of requests and concurrency level.
*   **Detailed Metrics**: Get a clear picture of your model's performance with key metrics including **TTFT**, **TBT**, and **TT**, aggregated across all requests.
*   **Graceful Shutdown**: All operations are context-aware. A single `Ctrl+C` will cleanly and immediately terminate any running chat or benchmark.
*   **Robust & Resilient**: Built-in, cancellable retry logic handles transient network errors automatically.
*   **Efficient Stream Processing**: Built on a zero-overhead iterator pattern for high-performance, low-latency data handling directly in the consumer's goroutine.

## Installation

```sh
go install github.com/shivanshkc/llmb/cmd/llmb@latest
```

## Usage

### Configuration

The following flags are persistent and can be used with any command:

*   `--base-url, -u`: The base URL of your OpenAI-compatible API (e.g., `http://localhost:8080`).
*   `--model, -m`: The name of the model to use (e.g., `gpt-4.1`).

### Chat Command

Start an interactive chat session.

```sh
llmb chat [flags]
```

**Features:**
*   Type your message and press Enter. The assistant's response will be streamed back token-by-token.
*   To send a message with a specific role, prefix your input with `role:`, for example:
    *   `system: You are a helpful assistant.`
    *   `assistant: How can I help you today?`

### Bench Command

Run a performance benchmark.

```sh
llmb bench --prompt "Your prompt here" [flags]
```

**Flags:**
*   `--prompt, -p`: The prompt to use for all benchmark requests. (Required)
*   `--request-count, -n`: The total number of requests to perform. (Default: 12)
*   `--concurrency, -c`: The number of requests to make at a time. (Default: 3)

## Design Philosophy

`llmb` was built not only to be a useful tool but also as an exercise in writing high-quality, idiomatic Go. The design focuses on three core principles:

1.  **Efficiency**: The core stream processing logic is designed as a zero-overhead abstraction. It avoids the resource cost of traditional channel adapters by using a synchronous, pull-based iterator, ensuring minimal latency and memory footprint.

2.  **Robustness**: The benchmark engine is a concurrent orchestrator designed for safe, leak-free operation. It ensures all concurrent tasks are gracefully managed and shut down on error or user interruption. The underlying HTTP client automatically handles transient network failures.

3.  **Composability**: The project is structured into clean, reusable packages. The `pkg/streams` iterator can be used to build complex data pipelines, and the `pkg/bench` runner can benchmark any function that produces a compatible stream.

## License

This project is licensed under the MIT License.