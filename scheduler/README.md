# scheduler

Simple background task scheduler for running periodic tasks at intervals.

## Installation

```go
import "github.com/status-im/proxy-common/scheduler"
```

## Key Types

- `Scheduler` - Manages background tasks with intervals

## Quick Start

```go
import (
    "context"
    "fmt"
    "time"
    
    "github.com/status-im/proxy-common/scheduler"
)

// Create scheduler
ctx := context.Background()
sched := scheduler.New(ctx)

// Add tasks with intervals
sched.AddTask("cleanup", 5*time.Minute, func(ctx context.Context) {
    fmt.Println("Running cleanup task")
    // Your cleanup logic here
})

sched.AddTask("health-check", 30*time.Second, func(ctx context.Context) {
    fmt.Println("Running health check")
    // Your health check logic here
})

// Start the scheduler
sched.Start()

// Check if running
if sched.IsRunning() {
    fmt.Println("Scheduler is running")
}

// Stop gracefully when done
defer sched.Stop()
```

## Full Example

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/status-im/proxy-common/scheduler"
)

func main() {
    ctx := context.Background()
    sched := scheduler.New(ctx)
    
    // Task 1: Cache cleanup every 5 minutes
    sched.AddTask("cache-cleanup", 5*time.Minute, func(ctx context.Context) {
        fmt.Println("Cleaning cache...")
        // Cleanup logic
    })
    
    // Task 2: Metrics collection every minute
    sched.AddTask("collect-metrics", 1*time.Minute, func(ctx context.Context) {
        fmt.Println("Collecting metrics...")
        // Metrics collection
    })
    
    // Start scheduler
    sched.Start()
    
    // Run for some time
    time.Sleep(10 * time.Minute)
    
    // Stop scheduler gracefully
    sched.Stop()
}
```

## Graceful Shutdown

The scheduler supports graceful shutdown via context cancellation:

```go
ctx, cancel := context.WithCancel(context.Background())
sched := scheduler.New(ctx)

// Start tasks
sched.Start()

// Later: cancel context to stop all tasks
cancel()
sched.Stop()
```
