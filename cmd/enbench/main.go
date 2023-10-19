package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"
	benchmarks "github.com/rotationalio/ensign-benchmarks/pkg"
	"github.com/rotationalio/ensign-benchmarks/pkg/blast"
	"github.com/rotationalio/ensign-benchmarks/pkg/options"
	"github.com/rotationalio/ensign-benchmarks/pkg/sustain"
	"github.com/rotationalio/ensign-benchmarks/pkg/workload"
	"github.com/rotationalio/go-ensign"
	api "github.com/rotationalio/go-ensign/api/v1beta1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	// Load dotenv files for easy configuration
	godotenv.Load()

	// Create a multi-command CLI application
	app := cli.NewApp()
	app.Name = "enbench"
	app.Version = benchmarks.Version()
	app.Usage = "run and manage ensign server benchmarks"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "credentials",
			Aliases: []string{"c"},
			Usage:   "path to json credentials file",
		},
		&cli.StringFlag{
			Name:    "endpoint",
			Aliases: []string{"e"},
			Value:   "staging.ensign.world:443",
			Usage:   "specify an ensign endpoint other than staging",
			EnvVars: []string{"ENSIGN_ENDPOINT"},
		},
		&cli.StringFlag{
			Name:    "auth-url",
			Aliases: []string{"a"},
			Value:   "https://auth.ensign.world",
			Usage:   "specify an ensign auth url other than staging",
			EnvVars: []string{"ENSIGN_AUTH_URL"},
		},
		&cli.StringFlag{
			Name:    "topic",
			Aliases: []string{"t"},
			Value:   "benchmarks",
			Usage:   "specify the topic to perform the benchmarks on",
		},
	}
	app.Commands = []*cli.Command{
		{
			Name:   "blast",
			Usage:  "run a blast benchmark",
			Before: configure,
			Action: runBlast,
			Flags: []cli.Flag{
				&cli.Uint64Flag{
					Name:    "operations",
					Aliases: []string{"N"},
					Usage:   "the number of events to send at the server",
				},
				&cli.Int64Flag{
					Name:    "data-size",
					Aliases: []string{"S"},
					Usage:   "the size in bytes of the payloads to send",
				},
			},
		},
		{
			Name:   "sustain",
			Usage:  "run a sustain benchmark",
			Before: configure,
			Action: runSustain,
			Flags: []cli.Flag{
				&cli.DurationFlag{
					Name:    "interval",
					Aliases: []string{"i"},
					Value:   1250 * time.Millisecond,
					Usage:   "the interval between publishing events",
				},
				&cli.Uint64Flag{
					Name:    "operations",
					Aliases: []string{"N"},
					Value:   0,
					Usage:   "the maximum number of events to publish (0 runs until stopped)",
				},
				&cli.Int64Flag{
					Name:    "data-size",
					Aliases: []string{"S"},
					Value:   256,
					Usage:   "the size in bytes of the payloads to send",
				},
			},
		},
		{
			Name:   "listen",
			Usage:  "listen for events on the specified topic",
			Before: configure,
			Action: listen,
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name:    "topic",
					Aliases: []string{"t"},
					Usage:   "specify the topics to subscribe to",
				},
			},
		},
		{
			Name:   "check",
			Usage:  "check that the benchmarks can run successfully",
			Before: configure,
			Action: check,
		},
		{
			Name:      "mktopic",
			Usage:     "create the specified topic(s) in your project",
			ArgsUsage: "topic [topic ...]",
			Before:    configure,
			Action:    createTopic,
		},
		{
			Name:   "testdata",
			Usage:  "generate testdata with duplicates",
			Action: mktestdata,
			Flags: []cli.Flag{
				&cli.IntFlag{
					Name:    "size",
					Aliases: []string{"s"},
					Usage:   "number of events to generate for the test",
					Value:   1000,
				},
				&cli.IntFlag{
					Name:    "num-keys",
					Aliases: []string{"k"},
					Usage:   "specify the number of available keys to add to events",
					Value:   20,
				},
				&cli.Float64Flag{
					Name:    "new-key-prob",
					Aliases: []string{"p"},
					Usage:   "specify the probability a new key value is generated",
					Value:   0.6,
				},
				&cli.Float64Flag{
					Name:    "duplicate-prob",
					Aliases: []string{"d"},
					Usage:   "specify the probability of duplicates in the dataset",
					Value:   0.1,
				},
				&cli.StringFlag{
					Name:    "out",
					Aliases: []string{"o"},
					Usage:   "the location to write the data out to",
					Value:   "events.pb.json",
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("could not start cli app")
	}
}

var conf *options.Options

func configure(c *cli.Context) error {
	conf = options.New()
	if creds := c.String("credentials"); creds != "" {
		conf.Credentials = creds
	}
	if endpoint := c.String("endpoint"); endpoint != "" {
		conf.Endpoint = endpoint
	}
	if authURL := c.String("auth-url"); authURL != "" {
		conf.AuthURL = authURL
	}
	return nil
}

func runBlast(c *cli.Context) (err error) {
	if n := c.Uint64("operations"); n > 0 {
		conf.Operations = n
	}
	if s := c.Int64("data-size"); s > 0 {
		conf.DataSize = s
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	b := blast.New(conf)
	if err = b.Run(ctx); err != nil {
		return cli.Exit(err, 1)
	}

	var results benchmarks.Metrics
	if results, err = b.Results(); err != nil {
		return cli.Exit(err, 1)
	}

	var data []byte
	if data, err = json.Marshal(results); err != nil {
		return cli.Exit(err, 1)
	}

	fmt.Println(string(data))
	return nil
}

func runSustain(c *cli.Context) (err error) {
	conf.Interval = c.Duration("interval")
	conf.Operations = c.Uint64("operations")
	conf.DataSize = c.Int64("data-size")

	b := sustain.New(conf)
	if err = b.Run(context.Background()); err != nil {
		return cli.Exit(err, 1)
	}
	return nil
}

func listen(c *cli.Context) (err error) {
	var client *ensign.Client
	if client, err = ensign.New(conf.Ensign()...); err != nil {
		return cli.Exit(err, 1)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	var sub *ensign.Subscription
	topics := c.StringSlice("topic")
	if sub, err = client.Subscribe(topics...); err != nil {
		return cli.Exit(err, 1)
	}

	for {
		select {
		case <-quit:
			return nil
		case event := <-sub.C:
			lgc := zerolog.Dict()
			for key, val := range event.Metadata {
				lgc.Str(key, val)
			}

			log.Info().
				Dict("metadata", lgc).
				Str("type", fmt.Sprintf("%s v%s", event.Type.Name, event.Type.Semver())).
				Str("mimetype", event.Mimetype.MimeType()).
				Int("data_size", len(event.Data)).
				Time("created", event.Created).
				Msg("event recv")

			if _, err := event.Ack(); err != nil {
				log.Error().Err(err).Msg("could not ack event")
			}
		}
	}
}

func check(c *cli.Context) (err error) {
	var client *ensign.Client
	if client, err = ensign.New(conf.Ensign()...); err != nil {
		return cli.Exit(err, 1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	output := make(map[string]string)

	var state *api.ServiceState
	if state, err = client.Status(ctx); err != nil {
		return cli.Exit(err, 1)
	}

	output["endpoint"] = conf.Endpoint
	output["status"] = state.Status.String()
	output["version"] = state.Version
	output["topic"] = conf.Topic

	var exists bool
	if exists, err = client.TopicExists(ctx, conf.Topic); err != nil {
		return cli.Exit(err, 1)
	}

	output["topic exists"] = fmt.Sprintf("%t", exists)

	var data []byte
	if data, err = json.MarshalIndent(output, "", "  "); err != nil {
		return cli.Exit(err, 1)
	}

	fmt.Println(string(data))
	return nil
}

func createTopic(c *cli.Context) (err error) {
	var client *ensign.Client
	if client, err = ensign.New(conf.Ensign()...); err != nil {
		return cli.Exit(err, 1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := 0; i < c.NArg(); i++ {
		var topicID string
		topic := c.Args().Get(i)
		if topicID, err = client.CreateTopic(ctx, topic); err != nil {
			return cli.Exit(err, 1)
		}

		log.Printf("topic %s created with id %s\n", topic, topicID)
	}

	return nil
}

func mktestdata(c *cli.Context) (err error) {
	nEvents := c.Int("size")
	nKeys := c.Int("num-keys")
	newKeyProb := c.Float64("new-key-prob")
	dupProb := c.Float64("duplicate-prob")
	out := c.String("out")

	pbjson := protojson.MarshalOptions{
		UseProtoNames:   true,
		UseEnumNumbers:  false,
		EmitUnpopulated: false,
	}

	data := make([]map[string]interface{}, 0, nEvents)
	gen := workload.NewRandomDuplicates(nKeys, newKeyProb, dupProb)

	for i := 0; i < nEvents; i++ {
		event := gen.Next()

		var serial []byte
		if serial, err = pbjson.Marshal(event); err != nil {
			return cli.Exit(err, 1)
		}

		obj := make(map[string]interface{})
		if err = json.Unmarshal(serial, &obj); err != nil {
			return cli.Exit(err, 1)
		}

		data = append(data, obj)
	}

	var f *os.File
	if f, err = os.Create(out); err != nil {
		return cli.Exit(err, 1)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(data); err != nil {
		return cli.Exit(err, 1)
	}

	return nil
}
