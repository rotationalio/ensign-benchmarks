/*
The blast package implements a benchmark where a fixed number of requests are sent to
the server in their own thread (e.g. blasting the server with requests) and the total
amount of time to respond to all requests is measured. This benchmark is primarily
intended for computing throughput in messages per second.
*/
package blast

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
	benchmarks "github.com/rotationalio/ensign-benchmarks/pkg"
	"github.com/rotationalio/ensign-benchmarks/pkg/metrics"
	"github.com/rotationalio/ensign-benchmarks/pkg/options"
	"github.com/rotationalio/ensign-benchmarks/pkg/stats"
	"github.com/rotationalio/go-ensign"
	api "github.com/rotationalio/go-ensign/api/v1beta1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	// Initializes zerolog with our default logging requirements
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.DurationFieldInteger = false
	zerolog.DurationFieldUnit = time.Millisecond
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
}

// Blast implements the benchmark interface and runs a throughput-oriented benchmark
// that fires off a workload with a fixed number of requests and measures the amount of
// time that the server responds to all requests.
type Blast struct {
	opts          *options.Options
	client        *ensign.Client
	topicID       ulid.ULID
	pubs          api.Ensign_PublishClient
	subs          api.Ensign_SubscribeClient
	started       time.Time
	duration      time.Duration
	events        uint64
	failures      uint64
	latencies     []time.Duration
	serverVersion string
	serverID      string
}

func New(opts *options.Options) *Blast {
	return &Blast{opts: opts}
}

// Note: this is prototype trash-pumpkin code.
func (b *Blast) Run(ctx context.Context) (err error) {
	if err = b.Prepare(ctx); err != nil {
		return err
	}
	defer b.Close()

	// Setup workload
	N := b.opts.Operations
	b.events = 0
	b.failures = 0
	b.latencies = make([]time.Duration, N)

	factory := MakeEventFactory(int(b.opts.DataSize), b.topicID)

	sentat := make([]time.Time, N)
	recvat := make([]time.Time, N)
	requests := make([]*api.PublisherRequest, N)
	responses := make([]*api.PublisherReply, N)

	for i := uint64(0); i < N; i++ {
		requests[i] = &api.PublisherRequest{
			Embed: &api.PublisherRequest_Event{
				Event: factory(),
			},
		}
	}

	log.Info().
		Str("topic", b.opts.Topic).
		Str("topic_id", b.topicID.String()).
		Str("server_id", b.serverID).
		Msg("blast benchmark starting")

	var wg sync.WaitGroup
	wg.Add(2)

	b.started = time.Now()
	go func() {
		defer wg.Done()
		for i, req := range requests {
			if err := b.pubs.Send(req); err != nil {
				log.Error().Err(err).Int("index", i).Msg("benchmark failed to send")
				return
			}
			sentat[i] = time.Now()
		}
	}()

	go func() {
		defer wg.Done()
		for i := uint64(0); i < N; i++ {
			rep, err := b.pubs.Recv()
			if err != nil {
				log.Error().Err(err).Uint64("index", i).Msg("benchmark failed to recv")
				return
			}
			responses[i] = rep
			recvat[i] = time.Now()
		}
	}()

	wg.Wait()
	b.duration = time.Since(b.started)

	// TODO: correlate requests and responses to ensure ordering from server is correct
	for i, recv := range recvat {
		b.latencies[i] = recv.Sub(sentat[i])
	}
	return nil
}

func (b *Blast) Prepare(ctx context.Context) (err error) {
	// Initialize the client
	if b.client, err = ensign.New(b.opts.Ensign()...); err != nil {
		return err
	}

	// Get the server version
	var rep *api.ServiceState
	if rep, err = b.client.Status(ctx); err != nil {
		return err
	}
	b.serverVersion = rep.Version

	// Get the topic ID for the specified topic
	// TODO: create topic if it doesn't exist
	var id string
	if id, err = b.client.TopicID(ctx, b.opts.Topic); err != nil {
		return err
	}

	if b.topicID, err = ulid.Parse(id); err != nil {
		return err
	}

	// Open the publish and subscribe streams
	clientID := fmt.Sprintf("benchmarks-%s", ulid.Make())
	if err = b.openPublisher(clientID); err != nil {
		return err
	}

	if err = b.openSubscriber(clientID); err != nil {
		return err
	}

	return nil
}

func (b *Blast) openPublisher(clientID string) (err error) {
	if b.pubs, err = b.client.PublishStream(context.Background()); err != nil {
		return err
	}

	req := &api.PublisherRequest{
		Embed: &api.PublisherRequest_OpenStream{
			OpenStream: &api.OpenStream{
				ClientId: clientID,
			},
		},
	}

	if err = b.pubs.Send(req); err != nil {
		return err
	}

	var rep *api.PublisherReply
	if rep, err = b.pubs.Recv(); err != nil {
		return err
	}

	var ready *api.StreamReady
	if ready = rep.GetReady(); ready == nil {
		return errors.New("did not get publisher ready message")
	}

	b.serverID = ready.ServerId
	return nil
}

func (b *Blast) openSubscriber(clientID string) (err error) {
	if b.subs, err = b.client.SubscribeStream(context.Background()); err != nil {
		return err
	}

	req := &api.SubscribeRequest{
		Embed: &api.SubscribeRequest_Subscription{
			Subscription: &api.Subscription{
				ClientId: clientID,
				Topics:   []string{b.topicID.String()},
			},
		},
	}

	if err = b.subs.Send(req); err != nil {
		return err
	}

	var rep *api.SubscribeReply
	if rep, err = b.subs.Recv(); err != nil {
		return err
	}

	var ready *api.StreamReady
	if ready = rep.GetReady(); ready == nil {
		return errors.New("did not get a subscriber ready message")
	}

	return nil
}

func (b *Blast) Close() {
	defer func() {
		b.client = nil
		b.pubs = nil
		b.subs = nil
	}()

	if err := b.pubs.CloseSend(); err != nil {
		if err != io.EOF {
			log.Error().Err(err).Msg("could not close publisher")
		}
	}

	if err := b.subs.CloseSend(); err != nil {
		if err != io.EOF {
			log.Error().Err(err).Msg("could not close subscriber")
		}
	}

	if err := b.client.Close(); err != nil {
		log.Error().Err(err).Msg("could not close ensign client")
	}

	log.Info().Msg("blast benchmark completed")
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
		"server_id":      b.serverID,
		"endpoint":       b.opts.Endpoint,
		"operations":     b.opts.Operations,
		"data_size":      b.opts.DataSize,
	}

	return results, nil
}

func (b *Blast) Client() (_ *ensign.Client, err error) {
	if b.client == nil {
		if b.client, err = ensign.New(b.opts.Ensign()...); err != nil {
			return nil, err
		}
	}
	return b.client, nil
}
