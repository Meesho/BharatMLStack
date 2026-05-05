# Circuit Breaker

This module provides circuit breaker implementations to improve application resilience. It supports two types of circuit breakers: an automatic one that wraps function calls, and a manual one that requires explicit signaling of success or failure.

Both circuit breakers are built on top of the [failsafe-go](https://github.com/failsafe-go/failsafe-go) library.

## Implementations

There are two main circuit breaker interfaces:

1.  `CircuitBreaker[Request, Response]`: An automatic circuit breaker that wraps a function call. It automatically records successes and failures based on the error returned by the function.
2.  `ManualCircuitBreaker`: A manual circuit breaker where you must explicitly check if an operation is permitted and then record its outcome.

Currently, only `version 1` is supported for both implementations, which uses `failsafe-go`.

## Configuration

The circuit breakers are configured using environment variables. The `BuildConfig(serviceName string)` function reads these variables, prefixed with the `serviceName`.

### Configuration Parameters

Here are the available configuration options (replace `SERVICE_NAME` with your service prefix):

*   `SERVICE_NAME_CB_ENABLED`: (boolean) Enable or disable the circuit breaker. Defaults to `false`.
*   `SERVICE_NAME_CB_NAME`: (string) A unique name for the circuit breaker instance, used for metrics and logging.
*   `SERVICE_NAME_CB_VERSION`: (int) The version of the circuit breaker implementation to use. Currently only `1` is supported.
*   `SERVICE_NAME_CB_WITH_DELAY_IN_MS`: (int) The delay in milliseconds before transitioning from `OPEN` to `HALF-OPEN` state.

#### Failure Thresholding

You can configure either time-based or count-based failure thresholds.

**Time-based:**
*   `SERVICE_NAME_CB_FAILURE_RATE_THRESHOLD`: (int) Failure rate percentage (1-100) to open the circuit.
*   `SERVICE_NAME_CB_FAILURE_RATE_MINIMUM_WINDOW`: (int) Minimum number of executions in the time window before the failure rate is calculated.
*   `SERVICE_NAME_CB_FAILURE_RATE_WINDOW_IN_MS`: (int) The time window in milliseconds for calculating the failure rate.

**Count-based:**
*   `SERVICE_NAME_CB_FAILURE_COUNT_THRESHOLD`: (int) Number of failures in the last `FAILURE_COUNT_WINDOW` executions to open the circuit.
*   `SERVICE_NAME_CB_FAILURE_COUNT_WINDOW`: (int) The size of the execution window for count-based thresholding.

#### Success Thresholding (for recovery)

*   `SERVICE_NAME_CB_SUCCESS_COUNT_THRESHOLD`: (int) Success executions needed in `HALF-OPEN` state to close the circuit.
*   `SERVICE_NAME_CB_SUCCESS_COUNT_WINDOW`: (int) The size of the execution window in `HALF-OPEN` state.

---

## Usage

### Automatic Circuit Breaker

This circuit breaker wraps your function calls. It's useful when you can determine the success or failure of an operation from a returned error.

**1. Get a Circuit Breaker instance:**

First, create a config and then get the circuit breaker.

```go
import "github.com/Meesho/go-core/v2/circuitbreaker"

// BuildConfig reads from environment variables
config := circuitbreaker.BuildConfig("MY_SERVICE") 

// Get the circuit breaker
cb := circuitbreaker.GetCircuitBreaker[MyRequest, MyResponse](config)
```

**2. Execute a function:**

Use the `Execute` or `ExecuteForGrpc` method to protect a function call.

```go
// Sample request and task
request := MyRequest{...}
task := func(req MyRequest) (MyResponse, error) {
    // a potentially failing operation
    return callMyDependency(req)
}

// Execute the task through the circuit breaker
response, err := cb.Execute(request, task)

if err != nil {
    // Handle error. This could be the error from the task
    // or a circuitbreaker.ErrCircuitBreakerOpen error.
}
```

### Manual Circuit Breaker

This circuit breaker is for scenarios where you need more control. You manually check if a request should be allowed and then later record if it was a success or failure. This is useful for protocols or operations where success/failure isn't determined by a simple error return from a single function call.

**1. Using the Manager (Recommended)**

The `Manager` can create and manage instances of manual circuit breakers, which is useful when you need multiple circuit breakers based on some key (e.g., a destination host).

```go
import "github.com/Meesho/go-core/v2/circuitbreaker"

// Create a new manager for your service
manager := circuitbreaker.NewManager("MY_SERVICE")

// Get or create a circuit breaker for a specific key
cb, err := manager.GetOrCreateManualCB("some-unique-key")
if err != nil {
    // handle error
}

// ... use the circuit breaker
```

**2. Direct Instantiation**

You can also get a single manual circuit breaker instance directly.

```go
import "github.com/Meesho/go-core/v2/circuitbreaker"

// BuildConfig reads from environment variables
config := circuitbreaker.BuildConfig("MY_SERVICE")

// Get the manual circuit breaker
cb := circuitbreaker.GetManualCircuitBreaker(config)
```

**3. Usage Pattern**

```go
if !cb.IsAllowed() {
    // The circuit is open, reject the request or return a fallback.
    return errors.New("circuit breaker is open")
}

// Proceed with the operation
err := doSomething()

if err != nil {
    cb.RecordFailure()
    // handle operation error
} else {
    cb.RecordSuccess()
}
```

If the circuit breaker is disabled (`SERVICE_NAME_CB_ENABLED=false`), `GetManualCircuitBreaker` will return a "pass-through" breaker where `IsAllowed()` always returns `true`, and `RecordSuccess()`/`RecordFailure()` do nothing. 