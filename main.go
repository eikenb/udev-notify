package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	udev "github.com/jochenvg/go-udev"
)

// ---------------------------------------------------------------------
// Workers to run triggers
var Workers = 3

// Workers delay running scripts slightly to give OS time to register device
var WorkerDelay = 200 * time.Millisecond

// abstract the *Device type so I can create test entries
type device interface {
	Action() string
	Properties() map[string]string
	PropertyValue(string) string
}

// ---------------------------------------------------------------------
func main() {
	flagParse()

	conf := getConfig(configfile, override_subsystems)
	switch {
	case list_devs:
		displayDeviceList(conf)
	case monit:
		devchan := deviceChan(conf)
		printerChan(devchan)
	default:
		devchan := deviceChan(conf)
		matchchan := commandRunners(conf)
		watchLoop(devchan, matchchan, conf)
	}
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

// Print device stream
func printerChan(devchan <-chan device) {
	for d := range devchan {
		log.Println(devString(d))
	}
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

// main loop
// monitors udev events, looks for matches and runs commands
func watchLoop(devchan <-chan device, matchchan chan<- rule, conf *Config) {
	rate_limiter := map[string]struct{}{}
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
					if _, ok := rate_limiter[rule.Command]; !ok {
						rate_limiter[rule.Command] = struct{}{}
						go func() {
							time.Sleep(time.Second)
							delete(rate_limiter, rule.Command)
						}()
						matchchan <- rule
					}
				}
			}
		}
	}
	close(matchchan)
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
