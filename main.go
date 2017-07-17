package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jochenvg/go-udev"
)

// ---------------------------------------------------------------------
// Workers to run triggers
var Workers = 3

// Workers delay running scripts slightly to give OS time to register device
var WorkerDelay = 200 * time.Millisecond

// Flags
var list_devs bool
var monit bool
var monit_subsystems []string

const usage_text = `Usage: %s [options] [subsystem ...]

Monitor and run scripts based on udev events. Primary run with a configation
file running in the background of your session, with other options available
to help configuring it.

Options:

  subsystem - one or more udev subsystems filters (overrides configured list)

`

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, usage_text, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "  -h    Help\n")
		os.Exit(1)
	}
	flag.BoolVar(&list_devs, "list", false, "List devices connected.")
	flag.BoolVar(&monit, "monit", false, "Print device events to STDOUT.")
	flag.Parse()
	if monit {
		monit_subsystems = flag.Args()
	}
}

// abstract the *Device type so I can create test entries
type device interface {
	Action() string
	Properties() map[string]string
	PropertyValue(string) string
}

// ---------------------------------------------------------------------

func main() {
	conf := getConfig()
	if list_devs {
		displayDeviceList(conf)
	} else if monit {
		if len(monit_subsystems) > 0 {
			conf.subsystems = monit_subsystems
			fmt.Println("Monitored subsystem override:", monit_subsystems)
		}
		devchan := deviceChan(conf)
		printerChan(devchan)
	} else {
		devchan := deviceChan(conf)
		matchchan := commandRunners(conf)
		watchLoop(devchan, matchchan, conf)
	}
}

//
func printerChan(devchan <-chan device) {
	for d := range devchan {
		fmt.Println(devString(d))
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
				fmt.Println("************************ rule fired: ", r.Command)
				cmd := r.Command
				if !filepath.IsAbs(cmd) {
					cmd = filepath.Join(os.ExpandEnv(conf.ScriptPath),
						r.Command)
				}
				out, err := exec.Command(cmd).CombinedOutput()
				if err != nil {
					fmt.Println(err)
				}
				if len(out) > 0 {
					fmt.Printf("%s\n", out)
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

	for _, sub := range conf.subsystems {
		m.FilterAddMatchSubsystem(sub)
	}

	done := make(chan struct{})
	devchan := make(chan device)
	ch, err := m.DeviceChan(done)
	if err != nil {
		fatal(err)
	}
	go func() {
		<-sighalt()
		close(done)
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

func fatal(i ...interface{}) {
	fmt.Fprintln(os.Stderr, i...)
	os.Exit(1)
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

	for _, sub := range conf.subsystems {
		e.AddMatchSubsystem(sub)
	}
	e.AddMatchIsInitialized()

	udev_devices, err := e.Devices()
	if err != nil {
		fatal(err)
	}
	for _, d := range udev_devices {
		fmt.Println(devString(d))
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
