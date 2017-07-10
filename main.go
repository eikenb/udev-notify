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
	"syscall"

	"github.com/jochenvg/go-udev"
)

// ---------------------------------------------------------------------
// Configuration
//
// Location of scripts
const SCRIPT_PATH = "${HOME}/bin/xinput.d"
const WORKERS = 2

type rule struct {
	PropName, PropValue, Command, Action string
}

// which udev subsystems to monitor
var subsystems = []string{
	"hid", // USB Devices
	"drm", // External Display
}

// Return property name to use as header for this type of subsystem
var subSysHeaderProp = map[string]string{
	"hid": "HID_NAME",
	"drm": "DEVPATH",
}

// Device rules
var rules []rule = []rule{
	{
		PropName:  "HID_NAME",
		PropValue: "2010 REV 1.7 Audioengine D1",
		Command:   "set-default-sink",
		Action:    "add",
	},
	{
		PropName:  "HID_NAME",
		PropValue: "Kensington Kensington Slimblade Trackball",
		Command:   "slimblade",
		Action:    "add",
	},
}

// ---------------------------------------------------------------------

var listem bool

func init() {
	flag.BoolVar(&listem, "list", false, "List devices connected.")
	flag.Parse()
}

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
	Syspath() string
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
				if pval == rule.PropValue && rule.Action == d.Action() {
					matchchan <- rule
				}
			}
		}
	}
	close(matchchan)
}

// Run the commands for matching rules
// use a small pool in case a script is slow
func commandRunners() chan<- rule {
	matchchan := make(chan rule)
	for i := 0; i < WORKERS; i++ {
		go func() {
			for r := range matchchan {
				cmd := exec.Command(
					filepath.Join(os.ExpandEnv(SCRIPT_PATH), r.Command))
				out, err := cmd.CombinedOutput()
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

	for _, sub := range subsystems {
		m.FilterAddMatchSubsystem(sub)
	}

	done := make(chan struct{})
	devchan := make(chan device)
	ch, err := m.DeviceChan(done)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	go func() {
		<-sighalt()
		close(done)
	}()
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
func displayDeviceList() {
	u := udev.Udev{}
	e := u.NewEnumerate()

	for _, sub := range subsystems {
		e.AddMatchSubsystem(sub)
	}
	e.AddMatchIsInitialized()

	udev_devices, err := e.Devices()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	for _, d := range udev_devices {
		fmt.Println(devString(device(d)))
	}
}

// returns list of connected devices and properties
func devString(dev device) string {
	name := "Mising subsystem name field"
	name_prop, ok := subSysHeaderProp[dev.PropertyValue("SUBSYSTEM")]
	if ok {
		name = strings.TrimSpace(dev.PropertyValue(name_prop))
	}
	properties := dev.Properties()
	ordered_keys := make([]string, 0, len(properties))
	result := make([]string, 0, len(properties)+2)

	result = append(result,
		fmt.Sprintf("\n%s\n%s\n", name, strings.Repeat("-", len(name))))
	result = append(result,
		fmt.Sprintf("%s = %s\n", "PropertyName", "PropertyValue"))
	for k := range properties {
		ordered_keys = append(ordered_keys, k)
	}
	sort.Sort(sort.StringSlice(ordered_keys))
	for _, k := range ordered_keys {
		v := properties[k]
		result = append(result,
			fmt.Sprintf("- %s = \"%s\"\n", k, strings.TrimSpace(v)))
	}
	return strings.Join(result, "")
}
