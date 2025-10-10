# Streaming Package

A generic, type-safe streaming package for gRPC client operations. This package provides three distinct streaming patterns optimized for different use cases.

## 📦 Package Structure

```
client/streaming/
├── types.go          # Common interfaces and callback types
├── client.go         # Client streaming (many → one)
├── sequential.go     # Sequential bidirectional (request-response pairs)
└── bidirectional.go  # Concurrent bidirectional (true streaming)
```

## 🎯 Streaming Patterns

### 1. Client Streaming (Many → One)

**File:** `client.go`

**Pattern:** `Send → Send → Send → CloseAndRecv()`

**When to use:**
- Multiple inputs, single final response
- Aggregation operations
- Examples: Delete operations

**Example:**
```go
stream, _ := client.Delete(ctx)
doneCh, err := streaming.NewClientStreamProcessor(
    ctx, 
    stream, 
    recordRefsCh,
    func(result *emptypb.Empty, err error) error {
        if err != nil {
            return err
        }
        fmt.Println("Delete completed")
        return nil
    },
)
<-doneCh
```

**Performance:** Low latency for small batches, single round-trip overhead.

---

### 2. Sequential Bidirectional (Request-Response Pairs)

**File:** `sequential.go`

**Pattern:** `Send → Recv → Send → Recv`

**When to use:**
- Server responds immediately to each request
- Strict ordering required
- Simple synchronous processing
- Examples: Lookup operations where order matters

**Example:**
```go
stream, _ := client.Lookup(ctx)
doneCh, err := streaming.NewSequentialStreamProcessor(
    ctx,
    stream,
    recordRefsCh,
    func(ref *RecordRef, meta *RecordMeta, err error) error {
        if err != nil {
            return err
        }
        fmt.Printf("Ref: %v, Meta: %v\n", ref, meta)
        return nil
    },
)
<-doneCh
```

**Performance:** 
- **Latency per item:** ~10-20ms (network RTT)
- **1000 items:** ~10-20 seconds
- Good for: Small batches, strict ordering

---

### 3. Concurrent Bidirectional (True Streaming)

**File:** `bidirectional.go`

**Pattern:** `Sender || Receiver` (parallel goroutines)

**When to use:**
- High-performance batch operations
- Large volumes of data
- Server can batch/pipeline
- Order doesn't need to be preserved
- Examples: Pull/Push operations

**Example:**
```go
stream, _ := client.Pull(ctx)
recordsCh, errCh, err := streaming.NewBidirectionalStreamProcessor(
    ctx,
    stream,
    recordRefsCh,
    func(record *Record) error {
        if record == nil {
            return errors.New("nil record")
        }
        return nil
    },
)

for record := range recordsCh {
    process(record)
}

if err := <-errCh; err != nil {
    log.Fatal(err)
}
```

**Performance:**
- **Latency:** ~10-20ms (one-time)
- **1000 items:** ~1-2 seconds
- **10x faster** than sequential for large batches
- Good for: High throughput, large batches

---

## 📊 Performance Comparison

| Pattern | 100 Items | 1000 Items | 10000 Items | Use Case |
|---------|-----------|------------|-------------|----------|
| **Client** | ~15ms | ~20ms | ~30ms | Aggregation |
| **Sequential** | ~1s | ~10s | ~100s | Ordered pairs |
| **Bidirectional** | ~100ms | ~1s | ~10s | High throughput |

*Assumes 10ms network latency, server processing negligible*

---

## 🔧 Type Definitions

### Interfaces

```go
// ClientStream: Many inputs → One output
type ClientStream[InT, OutT any] interface {
    Send(*InT) error
    CloseAndRecv() (*OutT, error)
    grpc.ClientStream
}

// BidirectionalStream: Independent send/receive
type BidirectionalStream[InT, OutT any] interface {
    Send(*InT) error
    Recv() (*OutT, error)
    CloseSend() error
    grpc.ClientStream
}
```

### Callbacks

```go
// Client streaming callback
type ClientReceiverFn[OutT any] func(*OutT, error) error

// Sequential streaming callback
type SequentialReceiverFn[InT, OutT any] func(*InT, *OutT, error) error

// Output validator for bidirectional streaming
type OutputValidatorFn[OutT any] func(*OutT) error
```

---

## ✅ Features

All processors include:
- ✅ **Context cancellation** - Respects context throughout
- ✅ **Input validation** - Nil checks for all parameters
- ✅ **Error propagation** - Clear error handling and reporting
- ✅ **Goroutine safety** - No leaks, proper cleanup
- ✅ **Type safety** - Generic implementation with compile-time checks

---

## 🎓 Best Practices

### 1. Choose the Right Pattern

```go
// ❌ Don't use Sequential for large batches
for i := 0; i < 10000; i++ {
    // 10000 × 10ms = 100 seconds!
}

// ✅ Use Bidirectional for large batches
streaming.NewBidirectionalStreamProcessor(...)
// 10ms + processing = ~1 second!
```

### 2. Always Check Errors

```go
// ❌ Don't ignore error channel
recordsCh, _, _ := streaming.NewBidirectionalStreamProcessor(...)

// ✅ Always check errors
recordsCh, errCh, err := streaming.NewBidirectionalStreamProcessor(...)
if err != nil {
    return err
}
for record := range recordsCh {
    process(record)
}
if err := <-errCh; err != nil {
    return err
}
```

### 3. Use Context for Cleanup

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

recordsCh, errCh, _ := streaming.NewBidirectionalStreamProcessor(ctx, ...)
```

---

## 🧪 Testing

Each processor is production-ready with:
- Input validation
- Context cancellation
- Error handling
- Goroutine cleanup

Run with race detector:
```bash
go test -race ./client/streaming/...
```

---

## 📚 Additional Resources

- [gRPC Streaming Guide](https://grpc.io/docs/what-is-grpc/core-concepts/#streaming)
- [Go Context Package](https://pkg.go.dev/context)
- [Proto Definitions](../proto/agntcy/dir/store/v1/store_service.proto)

---

## 🤝 Contributing

When adding new streaming patterns:
1. Add the pattern to the appropriate file
2. Include comprehensive documentation
3. Add performance characteristics
4. Provide usage examples
5. Run linter and race detector

---

*Last updated: October 2025*

