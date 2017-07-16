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
// Configuration
//
// Location of scripts (absolute paths ignore this)
const SCRIPT_PATH = "${HOME}/bin/xinput.d"

// Device rules:
// PropName is the name of the device property to match against
// PropValue is the value to match against (suffix match)
// Action the udev "action" to filter on (add, remove, change, online, offline)
// Command is the name of your script/program to run
var rules []rule = []rule{
	{
		Subsystem: "hid",
		PropName:  "HID_NAME",
		PropValue: "FiiO DigiHug USB Audio",
		Action:    "add",
		Command:   "set-default-sink",
	},
	{
		Subsystem: "hid",
		PropName:  "HID_NAME",
		PropValue: "2010 REV 1.7 Audioengine D1",
		Action:    "add",
		Command:   "set-default-sink",
	},
	{
		Subsystem: "hid",
		PropName:  "HID_NAME",
		PropValue: "Kensington Kensington Slimblade Trackball",
		Action:    "add",
		Command:   "slimblade",
	},
}

// ---------------------------------------------------------------------
//
var Workers = 3
var WorkerDelay = 200 * time.Millisecond

type rule struct {
	PropName, PropValue, Command, Action, Subsystem string
	limiter                                         int32
}

var listem bool
var subsystems map[string]struct{} = make(map[string]struct{})

func init() {
	flag.BoolVar(&listem, "list", false, "List devices connected.")
	flag.Parse()
	for _, r := range rules {
		subsystems[r.Subsystem] = struct{}{}
	}
}

// ---------------------------------------------------------------------

func main() {
	if listem {
		displayDeviceList()
	} else {
		devchan := deviceChan()
		matchchan := commandRunners()
		watchLoop(devchan, matchchan)
	}
}

// abstract the *Device type so I can create test entries
type device interface {
	Action() string
	Properties() map[string]string
	PropertyValue(string) string
}

// main loop
// monitors udev events, looks for matches and runs commands
func watchLoop(devchan <-chan device, matchchan chan<- rule) {
	watched_actions := map[string]bool{}
	for _, rule := range rules {
		watched_actions[rule.Action] = true
	}
	for d := range devchan {
		if watched_actions[d.Action()] {
			for _, rule := range rules {
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
func commandRunners() chan<- rule {
	matchchan := make(chan rule, Workers*3)
	for i := 0; i < Workers; i++ {
		go func() {
			for r := range matchchan {
				time.Sleep(WorkerDelay)
				fmt.Println("************************ rule fired: ", r.Command)
				cmd := r.Command
				if !filepath.IsAbs(cmd) {
					cmd = filepath.Join(os.ExpandEnv(SCRIPT_PATH), r.Command)
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
func deviceChan() <-chan device {
	u := udev.Udev{}
	m := u.NewMonitorFromNetlink("udev")

	for sub := range subsystems {
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
func displayDeviceList() {
	u := udev.Udev{}
	e := u.NewEnumerate()

	for sub := range subsystems {
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
