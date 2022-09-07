package benchmarks

import "context"

// Benchmark is an interface for running a benchmark test against a system and getting
// the results back out. Generally speaking, benchmarks are composed of a client, a
// workload, and metrics. The client is used to connect to the system and executes
// requests that are generated from the workload. The results of the benchmark are
// monitored and stored in the metrics, which can be output either using a system tool
// such as prometheus or monitored in memory and saved to disk as CSV or JSON output.
type Benchmark interface {
	// Execute the benchmark, may return an error if already run; this method should
	// block for the duration of the execution and is usually run from the main routine.
	// A context may be passed into the run method to pass deadlines or other values.
	Run(context.Context) error

	// Attempt to stop the execution of the benchmark.
	Stop(context.Context) error

	// Returns the results of the benchmark.
	Results() (Metrics, error)
}

// Metrics contains the results of a benchmark and is essentially a collection of
// named measurements. Every benchmark has at most one collection of metrics that is
// used to serialize results either to disk or over the network.
type Metrics interface {
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
	// Execute a single workflow item and return output.
	Exec(context.Context, interface{}) (interface{}, error)

	// Connect is called at the start of a benchmark run and Close is called on Stop.
	Connect() error
	Close() error
}

// Workload is used to generate requests. The workload is essentially an iterator that
// can return a fixed set of data or can generate data on demand forever.
type Workload interface {
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
