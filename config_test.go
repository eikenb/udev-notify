package main

import (
	"io/ioutil"
	"log"
	"reflect"
	"testing"
)

func init() {
	log.SetOutput(ioutil.Discard)
}

func TestConfigLoad(t *testing.T) {
	conf := loadConfig("./example-config.toml")
	if conf.ScriptPath != "${HOME}/bin/udev.d" {
		t.Error("Bad ScriptPath value: ", conf.ScriptPath)
	}
	if len(conf.Rules) != 3 {
		t.Error("Wrong number of rules: ", len(conf.Rules))
	}
	if conf.Rules[1].PropName != "ID_MODEL" {
		t.Error("Bad Rules field, got: ", conf.Rules[1].PropName,
			"want: HID_NAME")
	}
	if !reflect.DeepEqual(conf.Rules[1].Args, []string{"set-default-sink", "Audioengine_D1"}) {
		t.Error("Bad Args field, got: ", conf.Rules[1].Args,
			"want: [set-default-sink, Audioengine_D1]")
	}
}

func TestOverrides(t *testing.T) {
	conf := &Config{subsystems: []string{"foo", "bar"}}
	conf.overrideSubsystems([]string{})
	if conf.subsystems[0] != "foo" {
		t.Error("conf.subsystems replaced when it shouldn't have been.")
	}
	conf.overrideSubsystems([]string{"zed"})
	if conf.subsystems[0] != "zed" {
		t.Error("conf.subsystems not replaced when it should have been.")
	}
	if len(conf.subsystems) != 1 {
		t.Error("conf.subsystems too long", len(conf.subsystems))
	}
}

func TestAllSubsystems(t *testing.T) {
	conf := &Config{subsystems: []string{"foo", "bar"}}
	conf.overrideSubsystems([]string{"all"})
	if len(conf.subsystems) != 0 {
		t.Error("conf.subsystems 'all' didn't work")
	}
}
