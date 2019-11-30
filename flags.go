package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

// Flags
var quiet bool
var list_devs bool
var monit bool
var override_subsystems []string
var configfile string

const usage_text = `Usage: %s [options] [subsystem ...]

Monitor and run scripts based on udev events. Primary run with a configation
file running in the background of your session, with other options available
to help configuring it. Configuration file defaults to standard XDG location
(usually ~/.config/udev-notify/config.toml).

Options:

  subsystem - One or more udev subsystems filters (overrides configured list).
              Use the term 'all' to not filter at all (list everything).

`

func flagParse() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, usage_text, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "  -h    Help\n")
		os.Exit(1)
	}
	flag.StringVar(&configfile, "c", "", "Use `configfile` for your config")
	flag.BoolVar(&list_devs, "l", false, "List devices connected")
	flag.BoolVar(&monit, "w", false,
		"Watch and write device events to STDOUT")
	flag.BoolVar(&quiet, "q", false, "Quiet all normal output")
	flag.Parse()
	log.SetFlags(0)
	log.SetOutput(os.Stdout)
	if quiet {
		log.SetOutput(ioutil.Discard)
	}
	override_subsystems = flag.Args()
}
