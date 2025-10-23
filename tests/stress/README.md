# Stress Tests

This directory contains stress tests for the Kephas WebSocket server, designed to validate performance and stability under high load conditions.

## Test Scenarios

### 1. TestStress5000Connections
Tests the server's ability to handle **5,000 simultaneous connections**.

**What it tests:**
- Connection establishment rate
- Concurrent connection handling
- Message broadcasting to 5,000+ clients
- Message latency under load
- Connection stability

**Metrics reported:**
- Connection success rate
- Messages sent/received
- Average message latency
- Messages per second throughput

### 2. TestStress10000Connections
Tests extreme load with **10,000 simultaneous connections**.

**What it tests:**
- Server limits with very high connection count
- Resource handling under extreme load
- Connection failure tolerance

**Metrics reported:**
- Connection success rate (90%+ threshold)
- Total messages sent
- Connections established per second

### 3. TestStressConcurrentMessaging
Tests high-frequency messaging with **100 clients sending 1,000 messages each**.

**What it tests:**
- Message throughput (100,000 total messages)
- Rate limiting behavior
- Message broadcasting performance
- Network bandwidth utilization

**Metrics reported:**
- Messages per second
- Throughput in MB/sec
- Message delivery success rate

## Running the Tests

### Prerequisites

Before running stress tests, you may need to increase your system's file descriptor limit:

```bash
# Check current limit
ulimit -n

# Increase limit (temporary, for current session)
ulimit -n 65536

# For permanent changes, edit /etc/security/limits.conf (Linux)
```

### Run All Stress Tests

```bash
cd tests/stress
go test -v -timeout 30m
```

### Run Specific Test

```bash
# 5,000 connections test
go test -v -run TestStress5000Connections -timeout 10m

# 10,000 connections test
go test -v -run TestStress10000Connections -timeout 15m

# Concurrent messaging test
go test -v -run TestStressConcurrentMessaging -timeout 5m
```

### Skip Stress Tests (Short Mode)

```bash
go test -v -short
```

## Performance Tuning

### System Tuning (Linux)

For optimal performance with 5,000+ connections:

```bash
# Increase file descriptors
ulimit -n 65536

# TCP tuning
sudo sysctl -w net.core.somaxconn=4096
sudo sysctl -w net.ipv4.tcp_max_syn_backlog=4096

# Port range (if running many clients from same machine)
sudo sysctl -w net.ipv4.ip_local_port_range="1024 65535"

# TCP reuse
sudo sysctl -w net.ipv4.tcp_tw_reuse=1
```

### Go Runtime

```bash
# Increase GOMAXPROCS if needed (usually automatic)
export GOMAXPROCS=8

# Run test with race detector (slower, but useful for development)
go test -race -v -run TestStress5000Connections
```

### Server Configuration

Adjust rate limiting in the tests if needed:

```go
rateLimitConfig := &ws.RateLimitConfig{
    MessagesPerSecond: 1000,  // Increase for higher throughput
    Burst:             2000,  // Increase for burst traffic
    Enabled:           true,
}
```

## Expected Results

### Typical Performance (on modern hardware)

**5,000 Connections Test:**
- Connection success rate: >95%
- Message delivery rate: >90%
- Average latency: <100ms
- Throughput: >10,000 messages/sec

**10,000 Connections Test:**
- Connection success rate: >90%
- Demonstrates server stability at extreme scale

**Concurrent Messaging Test:**
- Messages per second: >50,000
- Throughput: >5 MB/sec
- Message send success rate: >95%

## Troubleshooting

### "too many open files" Error

```bash
# Increase file descriptor limit
ulimit -n 65536
```

### Connection Timeouts

- Reduce the number of concurrent connections
- Increase connection timeout in the test
- Check network/firewall settings
- Ensure sufficient system resources (RAM, CPU)

### High Memory Usage

This is expected with thousands of connections. Monitor with:

```bash
go test -v -run TestStress5000Connections -memprofile=mem.prof
go tool pprof mem.prof
```

### Rate Limit Errors

If you see many "Rate limit exceeded" messages:
- Increase `MessagesPerSecond` in rate limit config
- Adjust `Burst` parameter
- Add delays between messages in test code

## Continuous Integration

These tests are disabled in short mode by default due to their duration and resource requirements.

To run in CI:

```yaml
# GitHub Actions example
- name: Run stress tests
  run: |
    ulimit -n 65536
    cd tests/stress
    go test -v -timeout 30m
  timeout-minutes: 35
```

## Customization

You can create your own stress test scenarios:

```go
func TestCustomStress(t *testing.T) {
    const numClients = 1000
    const messagesPerClient = 100
    
    // Your custom test logic here
}
```

## Monitoring

While tests run, monitor system resources:

```bash
# Terminal 1: Run tests
go test -v -run TestStress5000Connections

# Terminal 2: Monitor resources
watch -n 1 'ps aux | grep go; netstat -an | grep 8765 | wc -l'
```

## Notes

- Tests use port `8765` by default (different from main server examples on `8080`)
- Each test starts its own server instance
- Tests are designed to be independent and can run in parallel
- Connection attempts are staggered to avoid overwhelming the server during startup
- Tests include cleanup logic to close all connections properly

## Performance Profiling

Generate CPU and memory profiles:

```bash
# CPU profile
go test -cpuprofile=cpu.prof -run TestStress5000Connections

# Memory profile
go test -memprofile=mem.prof -run TestStress5000Connections

# Analyze
go tool pprof cpu.prof
go tool pprof mem.prof
```
