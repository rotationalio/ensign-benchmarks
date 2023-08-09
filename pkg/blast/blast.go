/*
The blast package implements a benchmark where a fixed number of requests are sent to
the server in their own thread (e.g. blasting the server with requests) and the total
amount of time to respond to all requests is measured. This benchmark is primarily
intended for computing throughput in messages per second.
*/
package blast

import (
	"context"
	"crypto/rand"
	"sync"
	"time"

	benchmarks "github.com/rotationalio/ensign-benchmarks/pkg"
	"github.com/rotationalio/ensign-benchmarks/pkg/metrics"
	"github.com/rotationalio/ensign-benchmarks/pkg/options"
	"github.com/rotationalio/ensign-benchmarks/pkg/stats"
	api "github.com/rotationalio/ensign/pkg/api/v1beta1"
	mimetype "github.com/rotationalio/ensign/pkg/mimetype/v1beta1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Blast implements the benchmark interface and runs a throughput-oriented benchmark
// that fires off a workload with a fixed number of requests and measures the amount of
// time that the server responds to all requests.
type Blast struct {
	opts          *options.Options
	started       time.Time
	duration      time.Duration
	events        uint64
	failures      uint64
	latencies     []time.Duration
	serverVersion string
}

var _ benchmarks.Benchmark = &Blast{}

func New(opts *options.Options) *Blast {
	return &Blast{opts: opts}
}

type item struct {
	i     int
	start time.Time
}

// Note: this is prototype trash-pumpkin code.
func (b *Blast) Run(ctx context.Context) (err error) {
	// TODO: create a workload for the blast benchmark
	N := b.opts.Operations
	S := b.opts.DataSize

	b.events = 0
	b.failures = 0
	sentat := make(chan item, N)
	b.latencies = make([]time.Duration, N)
	results := make([]bool, N)

	// Create requests so they don't interfere with the benchmark
	requests := make([]*api.Event, N)
	for i := uint64(0); i < N; i++ {
		requests[i] = &api.Event{
			TopicId:  "benchmark",
			Mimetype: mimetype.ApplicationOctetStream,
			Type: &api.Type{
				Name:    "Random",
				Version: 1,
			},
			Data:    generateRandomBytes(int(S)),
			Created: timestamppb.Now(),
		}
	}

	// TODO: initialize the client as a separate object
	var cc *grpc.ClientConn
	if cc, err = grpc.Dial(b.opts.Addr, grpc.WithTransportCredentials(insecure.NewCredentials())); err != nil {
		return err
	}
	client := api.NewEnsignClient(cc)

	sctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get the server version
	var rep *api.ServiceState
	if rep, err = client.Status(ctx, &api.HealthCheck{}); err != nil {
		return err
	}
	b.serverVersion = rep.Version

	var stream api.Ensign_PublishClient
	if stream, err = client.Publish(sctx); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)

	b.started = time.Now()
	go func() {
		defer wg.Done()
		for ts := range sentat {
			if _, err := stream.Recv(); err != nil {
				results[ts.i] = false
			} else {
				results[ts.i] = true
				b.latencies[ts.i] = time.Since(ts.start)
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i, event := range requests {
			ts := item{
				i:     i,
				start: time.Now(),
			}

			if err := stream.Send(event); err == nil {
				sentat <- ts
			}

		}
		close(sentat)
	}()

	wg.Wait()

	b.duration = time.Since(b.started)
	for _, success := range results {
		if success {
			b.events++
		} else {
			b.failures++
		}
	}

	return nil
}

func (b *Blast) Stop(ctx context.Context) error {
	return nil
}

func (b *Blast) Results() (benchmarks.Metrics, error) {
	results := make(metrics.Metrics)
	results["events"] = b.events
	results["failures"] = b.failures

	latencies := &stats.Latencies{}
	latencies.Update(b.latencies...)
	latencies.SetDuration(b.duration)
	results["latencies"] = latencies

	// TODO: this is a hack just to get a number in for now
	results["bandwidth"] = float64(b.opts.DataSize*int64(b.opts.Operations)) / b.duration.Seconds()

	// TODO: these things are params that need to be output with the results but not metrics
	results["experiment"] = map[string]interface{}{
		"client_version": benchmarks.Version(),
		"server_version": b.serverVersion,
		"addr":           b.opts.Addr,
		"operations":     b.opts.Operations,
		"data_size":      b.opts.DataSize,
	}

	return results, nil
}

func (b *Blast) Client() benchmarks.Client {
	return nil
}

func (b *Blast) Workload() benchmarks.Workload {
	return nil
}

func (b *Blast) String() string {
	return "blast"
}

func generateRandomBytes(n int) (b []byte) {
	b = make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}
