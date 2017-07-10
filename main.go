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
	PropName, PropValue, Command string
}

// Device rules
var rules []rule = []rule{
	{
		PropName:  "HID_NAME",
		PropValue: "2010 REV 1.7 Audioengine D1",
		Command:   "set-default-sink",
	},
	{
		PropName:  "HID_NAME",
		PropValue: "Kensington Kensington Slimblade Trackball",
		Command:   "slimblade",
	},
}

// ---------------------------------------------------------------------

var listem bool

func init() {
	flag.BoolVar(&listem, "list", false, "List HID devices connected.")
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
	for d := range devchan {
		if d.Action() == "add" {
			props := d.Properties()
			for _, rule := range rules {
				val, ok := props[rule.PropName]
				if ok && strings.TrimSpace(val) == rule.PropValue {
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
	m.FilterAddMatchSubsystem("hid")

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

// display the list of HID devices
func displayDeviceList() {
	u := udev.Udev{}
	e := u.NewEnumerate()

	e.AddMatchSubsystem("hid")
	e.AddMatchIsInitialized()

	udev_devices, err := e.Devices()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	devices := make([]device, len(udev_devices))
	for i, d := range udev_devices {
		devices[i] = device(d)
	}

	fmt.Println(deviceList(devices))
}

// returns list of connected HID devices and properties
func deviceList(devices []device) []string {
	result := []string{}
	for i := range devices {
		device := devices[i]
		hid_name := strings.TrimSpace(device.PropertyValue("HID_NAME"))
		result = append(result,
			fmt.Sprintf("\n%s\n%s\n", hid_name,
				strings.Repeat("-", len(hid_name))))
		result = append(result,
			fmt.Sprintf("%s = %s\n", "PropertyName", "PropertyValue"))
		properties := device.Properties()
		ordered_keys := make([]string, 0, len(properties))
		for k := range properties {
			ordered_keys = append(ordered_keys, k)
		}
		sort.Sort(sort.StringSlice(ordered_keys))
		for _, k := range ordered_keys {
			v := properties[k]
			result = append(result,
				fmt.Sprintf("- %s = \"%s\"\n", k, strings.TrimSpace(v)))
		}
	}
	return result
}
