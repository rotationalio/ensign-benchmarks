package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	benchmarks "github.com/rotationalio/ensign-benchmarks/pkg"
	"github.com/rotationalio/ensign-benchmarks/pkg/blast"
	"github.com/rotationalio/ensign-benchmarks/pkg/options"
	"github.com/rotationalio/go-ensign"
	api "github.com/rotationalio/go-ensign/api/v1beta1"
	"github.com/urfave/cli/v2"
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
			Name:   "check",
			Usage:  "check that the benchmarks can run successfully",
			Before: configure,
			Action: check,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
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
