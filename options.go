package main

import (
	"fmt"
	"os"
	// TODO: write custom implementation
	"github.com/akamensky/argparse"
)

var options struct {
	pidfile  string
	cell     string
	cellName string
}

func ParseArgs() {
	parser := argparse.NewParser("", "HLHV cell controller")

	startCommand := parser.NewCommand("start", "Start a cell")
	stopCommand := parser.NewCommand("stop", "Stop a cell")
	restartCommand := parser.NewCommand("restart", "Restart a cell")
	statusCommand := parser.NewCommand("status", "Display cell status")

	pidfile := parser.String("p", "pidfile", &argparse.Options{
		Required: false,
		Help: "Specify the location the pidfile. Defaults to a " +
			"file named hlhv-<cell name>.pid located at /run",
	})

	cell := parser.String("c", "cell", &argparse.Options{
		Required: false,
		Default:  "queen",
		Help:     "The cell to control",
	})

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}

	options.pidfile = *pidfile
	options.cellName = *cell
	options.cell = "hlhv-" + *cell

	if options.pidfile == "" {
		options.pidfile = "/run/" + options.cell + ".pid"
	}

	if startCommand.Happened() {
		doStart()
	} else if stopCommand.Happened() {
		doStop()
	} else if restartCommand.Happened() {
		doRestart()
	} else if statusCommand.Happened() {
		doStatus()
	}
}
