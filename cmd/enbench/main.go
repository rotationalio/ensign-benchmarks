package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	benchmarks "github.com/rotationalio/ensign-benchmarks/pkg"
	"github.com/rotationalio/ensign-benchmarks/pkg/blast"
	"github.com/rotationalio/ensign-benchmarks/pkg/options"
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
	}
	app.Commands = []*cli.Command{
		{
			Name:   "blast",
			Usage:  "run a blast benchmark",
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
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}

func runBlast(c *cli.Context) (err error) {
	opts := options.New()
	opts.Addr = c.String("addr")
	if n := c.Uint64("operations"); n > 0 {
		opts.Operations = n
	}
	if s := c.Int64("data-size"); s > 0 {
		opts.DataSize = s
	}

	b := blast.New(opts)
	if err = b.Run(context.Background()); err != nil {
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
