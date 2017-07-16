package main

import (
	"testing"
	"time"
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

func testConfig() *Config {
	return &Config{
		ScriptPath: "",
		Rules: []rule{
			{PropName: "HID_NAME", PropValue: "foo",
				Command: "foo", Action: "add"},
		},
	}
}

func TestWatchLoop(t *testing.T) {
	conf := testConfig()
	devchan := make(chan device)
	matchchan := make(chan rule)
	go watchLoop(devchan, matchchan, conf)
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

// more an intergration test, but
func TestCommandRunner(t *testing.T) {
	conf := testConfig()
	Workers = 1
	WorkerDelay = time.Nanosecond
	matchchan := commandRunners(conf)
	matchchan <- rule{Command: "/bin/true"}
	matchchan <- rule{Command: "/bin/false"}
	time.Sleep(time.Millisecond)
}

var asstrings = []string{`---

HID_NAME = "foo"
SUBSYSTEM = "hid"
`, `---

HID_NAME = "bar"
SUBSYSTEM = "hid"
`,
}

func TestDeviceList(t *testing.T) {
	for i := range fakedevices {
		str := devString(fakedevices[i])
		if str != asstrings[i] {
			t.Error("list format is wrong, got:", str, "want:", asstrings[i])
		}
	}
}
