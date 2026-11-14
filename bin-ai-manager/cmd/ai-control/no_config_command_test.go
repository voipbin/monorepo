package main

import "testing"

// the stored platform settings are gone, so the root command has no config command
func Test_initCommand_noConfigCommand(t *testing.T) {
	root := initCommand()

	for _, c := range root.Commands() {
		if c.Name() == "config" {
			t.Errorf("Wrong match. expect: no config command, got: %s", c.CommandPath())
		}
	}
}
