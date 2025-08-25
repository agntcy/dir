# Routing Constants Usage Guide

This document describes how to use the routing constants consistently across the codebase.

## Import

```go
import "github.com/agntcy/dir/server/routing"
```

## Available Constants

### Timing Constants

```go
// DHT Record TTL (48 hours)
routing.DHTRecordTTL

// Label Republishing Interval (36 hours)  
routing.LabelRepublishInterval

// Remote Label Cleanup Interval (48 hours)
routing.RemoteLabelCleanupInterval

// Provider Record TTL (48 hours)
routing.ProviderRecordTTL

// DHT Refresh Interval (30 seconds)
routing.RefreshInterval
```

### Protocol Constants

```go
// Protocol prefix for DHT
routing.ProtocolPrefix // "dir"

// Rendezvous string for peer discovery
routing.ProtocolRendezvous // "dir/connect"
```

### Validation Constants

```go
// Maximum hops for distributed queries
routing.MaxHops // 20

// Notification channel buffer size
routing.NotificationChannelSize // 1000

// Minimum parts required in label keys (after string split)
routing.MinLabelKeyParts // 4
```

## Usage Examples

### Setting up periodic tasks

```go
// Cleanup task using consistent interval
ticker := time.NewTicker(routing.RemoteLabelCleanupInterval)
defer ticker.Stop()

// Republishing task using consistent interval  
republishTicker := time.NewTicker(routing.LabelRepublishInterval)
defer republishTicker.Stop()
```

### DHT Configuration

```go
// Configure DHT with consistent TTL
dht, err := dht.New(ctx, host, 
    dht.MaxRecordAge(routing.DHTRecordTTL),
    dht.ProtocolPrefix(protocol.ID(routing.ProtocolPrefix)),
)
```

### Validation

```go
// Check hop count limit
if req.GetMaxHops() > routing.MaxHops {
    return errors.New("max hops exceeded")
}

// Validate label key format
parts := strings.Split(labelKey, "/")
if len(parts) < routing.MinLabelKeyParts {
    return errors.New("invalid label key format")
}
```

### Channel Creation

```go
// Create notification channel with consistent buffer size
notifyCh := make(chan *handlerSync, routing.NotificationChannelSize)
```

## Benefits

1. **Consistency**: All components use the same timing values
2. **Maintainability**: Changes only need to be made in one place
3. **Documentation**: Constants are clearly documented with their purpose
4. **Type Safety**: Compile-time checking ensures values are used correctly
5. **Coordination**: Related intervals are properly coordinated (e.g., republish < TTL)
