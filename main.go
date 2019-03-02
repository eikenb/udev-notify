package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	udev "github.com/jochenvg/go-udev"
)

// ---------------------------------------------------------------------
// Workers to run triggers
var Workers = 3

// Workers delay running scripts slightly to give OS time to register device
var WorkerDelay = 200 * time.Millisecond

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

func init() {
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

// abstract the *Device type so I can create test entries
type device interface {
	Action() string
	Properties() map[string]string
	PropertyValue(string) string
}

// ---------------------------------------------------------------------
func main() {
	conf := getConfig(configfile, override_subsystems)
	if list_devs {
		displayDeviceList(conf)
	} else if monit {
		devchan := deviceChan(conf)
		printerChan(devchan)
	} else {
		devchan := deviceChan(conf)
		matchchan := commandRunners(conf)
		watchLoop(devchan, matchchan, conf)
	}
}

// Print device stream
func printerChan(devchan <-chan device) {
	for d := range devchan {
		log.Println(devString(d))
	}
}

// main loop
// monitors udev events, looks for matches and runs commands
func watchLoop(devchan <-chan device, matchchan chan<- rule, conf *Config) {
	watched_actions := map[string]bool{}
	for _, rule := range conf.Rules {
		watched_actions[rule.Action] = true
	}
	for d := range devchan {
		if watched_actions[d.Action()] {
			for _, rule := range conf.Rules {
				pval := strings.TrimSpace(d.PropertyValue(rule.PropName))
				prop_test := strings.HasSuffix(pval, rule.PropValue)
				action_test := rule.Action == d.Action()
				if prop_test && action_test {
					if atomic.CompareAndSwapInt32(&rule.limiter, 0, 1) {
						go func() {
							time.Sleep(time.Second)
							atomic.CompareAndSwapInt32(&rule.limiter, 1, 0)
						}()
						matchchan <- rule
					}
				}
			}
		}
	}
	close(matchchan)
}

// Run the commands for matching rules
// use a small pool in case a script is slow
func commandRunners(conf *Config) chan<- rule {
	matchchan := make(chan rule, Workers*3)
	for i := 0; i < Workers; i++ {
		go func() {
			for r := range matchchan {
				time.Sleep(WorkerDelay)
				log.Println("************************ rule fired: ", r.Command)
				cmd := r.Command
				if !filepath.IsAbs(cmd) {
					cmd = filepath.Join(os.ExpandEnv(conf.ScriptPath),
						r.Command)
				}
				out, err := exec.Command(cmd, r.Args...).CombinedOutput()
				if err != nil {
					log.Println(err)
				}
				if len(out) > 0 {
					log.Printf("%s\n", out)
				}
			}
		}()
	}
	return matchchan
}

// Returns the channel of device events
func deviceChan(conf *Config) <-chan device {
	u := udev.Udev{}
	m := u.NewMonitorFromNetlink("udev")
	m.FilterAddMatchTag("seat")
	for _, sub := range conf.subsystems {
		m.FilterAddMatchSubsystem(sub)
	}

	ctx, cancel := context.WithCancel(context.Background())
	devchan := make(chan device)
	ch, err := m.DeviceChan(ctx)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		<-sighalt()
		cancel()
	}()
	// wrap udev's chan in one that I can control the type
	go func() {
		for d := range ch {
			devchan <- d
		}
		close(devchan)
	}()
	return devchan
}

// watch for signals to quit
func sighalt() <-chan os.Signal {
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)
	return interrupts
}

// display the list of devices
func displayDeviceList(conf *Config) {
	u := udev.Udev{}
	e := u.NewEnumerate()
	e.AddMatchTag("seat")
	for _, sub := range conf.subsystems {
		e.AddMatchSubsystem(sub)
	}
	e.AddMatchIsInitialized()

	udev_devices, err := e.Devices()
	if err != nil {
		log.Fatal(err)
	}
	for _, d := range udev_devices {
		log.Println(devString(d))
	}
}

// returns list of connected devices and properties
func devString(dev device) string {
	properties := dev.Properties()
	ordered_keys := make([]string, 0, len(properties))
	result := make([]string, 0, len(properties)+1)

	result = append(result, fmt.Sprintf("%s\n\n", strings.Repeat("-", 3)))
	for k := range properties {
		ordered_keys = append(ordered_keys, k)
	}
	sort.Sort(sort.StringSlice(ordered_keys))
	for _, k := range ordered_keys {
		v := properties[k]
		result = append(result,
			fmt.Sprintf("%s = \"%s\"\n", k, strings.TrimSpace(v)))
	}
	return strings.Join(result, "")
}
