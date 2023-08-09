package benchmarks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
)

// Benchmark is an interface for running a benchmark test against a system and getting
// the results back out. Generally speaking, benchmarks are composed of a client, a
// workload, and metrics. The client is used to connect to the system and executes
// requests that are generated from the workload. The results of the benchmark are
// monitored and stored in the metrics, which can be output either using a system tool
// such as prometheus or monitored in memory and saved to disk as CSV or JSON output.
type Benchmark interface {
	fmt.Stringer

	// Execute the benchmark, may return an error if already run; this method should
	// block for the duration of the execution and is usually run from the main routine.
	// A context may be passed into the run method to pass deadlines or other values.
	Run(context.Context) error

	// Attempt to stop the execution of the benchmark.
	Stop(context.Context) error

	// Returns the results of the benchmark.
	Results() (Metrics, error)

	// Access the workload and the client
	Client() Client
	Workload() Workload
}

// Metrics contains the results of a benchmark and is essentially a collection of
// named measurements. Every benchmark has at most one collection of metrics that is
// used to serialize results either to disk or over the network.
type Metrics interface {
	json.Marshaler

	// Returns the names of the measurements contained by the metrics.
	Measurements() ([]string, error)

	// Returns the underlying metric for the given name, e.g. a distribution, counter,
	// gauge, throughput or Prometheus bridge/pusher.
	// TODO: unify the interface for the measurement object.
	Measurement(string) interface{}
}

// Client executes a workload request and returns a response and an error. For example
// to test the throughput of a single RPC call, the Exec method may simply be a wrapper
// for a gRPC client that executes a single RPC request against a server.
type Client interface {
	fmt.Stringer

	// Execute a single workflow item and return output.
	Exec(context.Context, interface{}) (interface{}, error)

	// Connect is called at the start of a benchmark run and Close is called on Stop.
	Connect() error
	Close() error
}

// Workload is used to generate requests. The workload is essentially an iterator that
// can return a fixed set of data or can generate data on demand forever.
type Workload interface {
	fmt.Stringer

	// Called before the workflow is executed to do any work that should not be measured
	// in the benchmark (e.g. preparing requests or in-memory buffers or connecting to
	// external resources such as databases)
	Prepare() error

	// Should advance the workload iterator and return true if new data is available.
	Next() bool

	// Value should return the next item in the workload for the client to execute
	Value() interface{}

	// Release closes the workload, which should clean up after itself
	Release() error
}

// Run is the primary entrypoint and conducts a single benchmark test. This function
// connects the client and prepares the workload before executing the benchmark, then
// releases the workload and closes the client and returns the metrics. This function
// also listens for OS signals such as interrupt to stop the benchmark in the middle of
// a run and works to respect the deadlines in the given context.
func Run(ctx context.Context, bench Benchmark) (_ Metrics, err error) {
	// Connect the client and prepare the workload
	if err = bench.Client().Connect(); err != nil {
		return nil, err
	}

	if err = bench.Workload().Prepare(); err != nil {
		return nil, err
	}

	// Ensure that the client and the workload are cleaned up when the function is done.
	defer func() {
		bench.Workload().Release()
		bench.Client().Close()
	}()

	// Listen for OS signals to stop the benchmark run
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	go func() {
		<-quit
		bench.Stop(ctx)
	}()

	// Execute the benchmark
	if err = bench.Run(ctx); err != nil {
		return nil, err
	}

	return bench.Results()
}
