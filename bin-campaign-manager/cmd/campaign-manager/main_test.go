package main

import (
	"testing"

	joonix "github.com/joonix/log"
	"github.com/sirupsen/logrus"
)

func Test_initLog(t *testing.T) {
	std := logrus.StandardLogger()
	savedFormatter := std.Formatter
	savedLevel := std.GetLevel()
	t.Cleanup(func() {
		std.SetFormatter(savedFormatter)
		std.SetLevel(savedLevel)
	})

	std.SetLevel(logrus.InfoLevel)
	std.SetFormatter(&logrus.TextFormatter{})

	initLog()

	if std.GetLevel() != logrus.DebugLevel {
		t.Errorf("Wrong level. expect: %v, got: %v", logrus.DebugLevel, std.GetLevel())
	}
	if _, ok := std.Formatter.(*joonix.Formatter); !ok {
		t.Errorf("Wrong formatter. expect: *joonix.Formatter, got: %T", std.Formatter)
	}
}
