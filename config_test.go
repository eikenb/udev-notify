package main

import (
	"testing"
)

func TestConfigLoad(t *testing.T) {
	conf := loadConfig("./example-config.toml")
	if conf.ScriptPath != "${HOME}/bin/udev.d" {
		t.Error("Bad ScriptPath value: ", conf.ScriptPath)
	}
	if len(conf.Rules) != 3 {
		t.Error("Wrong number of rules: ", len(conf.Rules))
	}
	if conf.Rules[1].PropName != "HID_NAME" {
		t.Error("Bad Rules field, got: ", conf.Rules[1].PropName,
			"want: HID_NAME")
	}
}
