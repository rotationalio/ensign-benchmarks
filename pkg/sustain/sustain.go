package sustain

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/rotationalio/ensign-benchmarks/pkg/options"
	"github.com/rotationalio/go-ensign"
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

// Sustain runs a benchmark that continuously sends events at the server until stopped.
type Sustain struct {
	opts   *options.Options
	client *ensign.Client
}

func New(opts *options.Options) *Sustain {
	return &Sustain{opts: opts}
}

// Note: this is prototype trash-pumpkin code.
func (b *Sustain) Run(ctx context.Context) (err error) {
	if err = b.Prepare(ctx); err != nil {
		return err
	}
	defer b.Close()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	N := b.opts.Operations
	nevents := uint64(0)
	ticker := time.NewTicker(b.opts.Interval)
	factory := MakeEventFactory(int(b.opts.DataSize))

sustain:
	for {
		select {
		case <-ticker.C:
			event := factory()
			b.client.Publish(b.opts.Topic, event)
			log.Info().Str("count", event.Metadata["counter"]).Str("id", event.Metadata["local_id"]).Msg("event published")

			// Wait for the event to be acked
			var acked bool
			if acked, err = event.Acked(); err != nil {
				log.Error().Err(err).Msg("could not get ack")
			}

			var nacked bool
			if !acked {
				if nacked, err = event.Nacked(); err != nil {
					log.Error().Err(err).Msg("event was nacked")
				}
			}
			log.Debug().Bool("acked", acked).Bool("nacked", nacked).Msg("publish result")

			// Check exit criteria
			nevents++
			if N > 0 {
				if nevents >= N {
					break sustain
				}
			}

		case <-quit:
			break sustain

		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func (b *Sustain) Prepare(ctx context.Context) (err error) {
	// Initialize the client
	if b.client, err = ensign.New(b.opts.Ensign()...); err != nil {
		return err
	}
	return nil
}

func (b *Sustain) Close() {
	defer func() {
		b.client = nil
	}()

	if err := b.client.Close(); err != nil {
		log.Error().Err(err).Msg("could not close ensign client")
	}

	log.Info().Msg("sustain benchmark closed")
}
