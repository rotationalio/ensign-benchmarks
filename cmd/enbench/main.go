package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	benchmarks "github.com/rotationalio/ensign-benchmarks"
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
	app.Flags = []cli.Flag{}
	app.Commands = []*cli.Command{}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}
