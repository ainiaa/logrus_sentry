package sentryhook

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestNewSyncHook(t *testing.T) {
	log := logrus.New()

	hook, err := NewSentryHook("")
	if err != nil {
		fmt.Printf("error:%s", err.Error())
	}
	log.Hooks.Add(hook)

	log.Info("test log info")

	log.Warn("test log warn adbdd")

	log.Error("test log error efdd")
}
