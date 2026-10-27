package logtest

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func Test_Dump_fieldsNotMarshalable(t *testing.T) {
	c := New(t)

	// a channel can not be encoded as json, so the dump names that instead of dropping the entry
	logrus.WithField("a", make(chan int)).Info("entry with a channel field")

	dump := c.Dump()
	if !strings.Contains(dump, "entry with a channel field") {
		t.Errorf("Wrong dump. expect: the message, got: %q", dump)
	}
	if !strings.Contains(dump, "(fields not marshalable)") {
		t.Errorf("Wrong dump. expect: the marker of the fields, got: %q", dump)
	}
}
