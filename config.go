package main

import (
	"io/ioutil"
	"log"

	"github.com/BurntSushi/toml"
	"github.com/BurntSushi/xdg"
)

type Config struct {
	// Location of scripts (absolute paths ignore this)
	ScriptPath string
	// List of property match/script trigger rules
	Rules []rule
	// Populated from Rules
	subsystems []string
}

// PropName is the name of the device property to match against
// PropValue is the value to match against (suffix match)
// Action the udev "action" to filter on (add, remove, change, online, offline)
// Command is the name of your script/program to run
// Args is a list of arguments to pass to the script/program
// Subsystem is used to filter udev events monitored
type rule struct {
	PropName, PropValue, Command, Action, Subsystem string
	Args                                            []string
	limiter                                         int32
}

// Find config file in XDG directory with optional override using
// environment variable "UdevNotifyConfig".
func configPath() string {
	paths := xdg.Paths{
		XDGSuffix: "udev-notify",
	}
	path, err := paths.ConfigFile("config.toml")
	if err != nil {
		log.Fatal(err)
	}
	return path
}

// Loads toml into data struct
func loadConfig(path string) *Config {
	var conf *Config
	var bs []byte
	var err error
	if bs, err = ioutil.ReadFile(path); err != nil {
		log.Fatal(err)
	}
	if _, err = toml.Decode(string(bs), &conf); err != nil {
		log.Fatal(err)
	}
	log.Printf("Config file successfully loaded with %d rules.\n",
		len(conf.Rules))
	set := make(map[string]struct{})
	for _, r := range conf.Rules {
		set[r.Subsystem] = struct{}{}
	}
	conf.subsystems = make([]string, 0, len(set))
	for k := range set {
		conf.subsystems = append(conf.subsystems, k)
	}
	return conf
}

// Optional overriding of values
func (conf *Config) overrideSubsystems(subs []string) {
	if len(subs) > 0 {
		if subs[0] == "all" {
			conf.subsystems = conf.subsystems[:0]
		} else {
			conf.subsystems = subs
		}
		log.Println("Subsystem override:", subs)
	}
}

// Combine above
func getConfig(cpath string, subs []string) *Config {
	if cpath == "" {
		cpath = configPath()
	}
	conf := loadConfig(cpath)
	conf.overrideSubsystems(subs)
	return conf
}
