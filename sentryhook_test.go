package sentryhook

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestNewSyncHook(t *testing.T) {
	log := logrus.New()

	hook, err := NewSentryHook("https://ffbf2d8b20714fb2a569f4e146146b36:06afa2ee512e423f84a87fed1fd08802@sentry.ddxq.mobi:443/36")
	if err != nil {
		fmt.Printf("error:%s", err.Error())
	}
	log.Hooks.Add(hook)

	log.Info("test log info")

	log.Warn("test log warn adbdd")

	log.Error("test log error efdd")
}
