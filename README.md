# 🚀 LLMB - Large Language Model Benchmarker

A powerful CLI tool for benchmarking and chatting with OpenAI-compatible REST APIs. Whether you're testing performance,
comparing models, or just having a conversation, LLMB has you covered.

## Features

- **Performance Benchmarking**: Comprehensive metrics including Time to First Token (TTFT), Time Between Tokens (TBT), 
    and Total Time (TT).
- **Interactive Chat**: Full-featured terminal chat experience with any compatible LLM.
- **Concurrent Testing**: Multi-threaded benchmarking for realistic load testing.
- **Detailed Statistics**: P90, P95, median, min, max, and average metrics.
- **Universal Compatibility**: Works with any OpenAI-compatible API.

## Installation

```bash
# Install from source
go install github.com/shivanshkc/llmb@latest
```

## Usage

### Benchmark Command

Test your LLM's performance with comprehensive metrics:

```bash
llmb bench --prompt "List all prime numbers between 30 and 60" --request-count 10 --concurrency 3
```

**Available Flags:**
- `-u, --base-url`: API base URL (default: `http://localhost:8080`)
- `-p, --prompt`: Test prompt to send
- `-n, --request-count`: Number of requests (default: `12`)
- `-c, --concurrency`: Concurrent requests (default: `3`)

**Sample Output:**
```
[1/6] requests complete.
[2/6] requests complete.
[3/6] requests complete.
[4/6] requests complete.
[5/6] requests complete.
[6/6] requests complete.

 METRIC  AVERAGE   MINIMUM  MEDIAN    MAXIMUM   P90       P95      
 TTFT    125.72ms  77.61ms  146.64ms  153.08ms  152.43ms  152.43ms 
 TBT     23.37ms   1.00μs   12.98μs   118.06ms  48.61ms   48.65ms  
 TT      3.52s     1.37s    2.78s     8.71s     3.74s     3.74s    
```

### Chat Command

Start an interactive conversation with your LLM:

```bash
llmb chat
```

**Available Flags:**
- `-u, --base-url`: API base URL (default: `http://localhost:8080`)

## 📊 Metrics Explained

| Metric   | Description                                                         |
|----------|---------------------------------------------------------------------|
| **TTFT** | **Time to First Token** - Latency until the first token arrives.    |
| **TBT**  | **Time Between Tokens** - Average delay between consecutive tokens. |
| **TT**   | **Total Time** - Complete request duration from start to finish.    |

Each metric includes comprehensive statistics:
- **Average**: Mean value across all requests.
- **Minimum**: Fastest recorded time.
- **Median**: 50th percentile value.
- **Maximum**: Slowest recorded time.
- **P90**: 90th percentile (10% of requests were slower).
- **P95**: 95th percentile (5% of requests were slower).

## Use Cases

- **Model Comparison**: Benchmark different LLM providers and models.
- **Performance Monitoring**: Track API performance over time.
- **Load Testing**: Simulate concurrent user scenarios.
- **Development**: Quick testing during local development.
- **Interactive Testing**: Chat with models to test responses.