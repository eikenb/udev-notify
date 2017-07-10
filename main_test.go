package main

import (
	"reflect"
	"testing"
)

type ruletest struct {
	props map[string]string
	ok    bool
}

type dev map[string]string

func (f dev) Syspath() string { return f["HID_NAME"] }
func (f dev) Action() string  { return "add" }
func (f dev) Properties() map[string]string {
	return f
}
func (f dev) PropertyValue(k string) string {
	return f[k]
}

var fakedevices = []device{
	dev{"HID_NAME": "foo", "SUBSYSTEM": "hid"},
	dev{"HID_NAME": "bar", "SUBSYSTEM": "hid"},
}

func TestWatchLoop(t *testing.T) {
	rules = []rule{
		{PropName: "HID_NAME", PropValue: "foo", Command: "foo"},
	}
	devchan := make(chan device)
	matchchan := make(chan rule)
	go watchLoop(devchan, matchchan)
	go func() {
		for _, d := range fakedevices {
			devchan <- d
		}
		close(devchan)
	}()
	out := <-matchchan
	if out.Command != "foo" {
		t.Error("match failure, got:", out.Command, "want: foo")
	}
	out = <-matchchan
	if out.Command != "" {
		t.Error("Rule mis-match, should have got nil rule, got: ", out)
	}
}

var asstrings = []string{
	"\nfoo\n---\n", "PropertyName = PropertyValue\n",
	"- HID_NAME = \"foo\"\n", "- SUBSYSTEM = \"hid\"\n",
	"\nbar\n---\n", "PropertyName = PropertyValue\n",
	"- HID_NAME = \"bar\"\n", "- SUBSYSTEM = \"hid\"\n",
}

func TestDeviceList(t *testing.T) {
	res := deviceList(fakedevices)
	if !reflect.DeepEqual(res, asstrings) {
		t.Error("list format is wrong, got:", res, "want:", asstrings)
	}
}
