package sentryhook

import (
	"fmt"
	"sync"
	"time"

	"github.com/ainiaa/bytesconv"
	sentrygo "github.com/getsentry/sentry-go"
	"github.com/sirupsen/logrus"
)

// SentryHook is a hook to handle writing to kafka log files.
// SentryHook delivers logs to a sentry server.
type SentryHook struct {
	Timeout                 time.Duration
	StacktraceConfiguration StackTraceConfiguration
	flushTimeout            time.Duration
	client                  *sentrygo.Client
	levels                  []logrus.Level
	hub                     *sentrygo.Hub
	tags                    map[string]string
	disableStacktrace       bool
	level                   logrus.Level
	asynchronous            bool
	formatter               logrus.Formatter
	mu                      sync.RWMutex
	wg                      sync.WaitGroup
}

type Option func(hook *SentryHook)

func WithTimeout(timeout time.Duration) Option {
	return func(hook *SentryHook) {
		hook.Timeout = timeout
	}
}

func WithLevels(levels []logrus.Level) Option {
	return func(hook *SentryHook) {
		hook.levels = levels
	}
}

func WithLevel(level logrus.Level) Option {
	return func(hook *SentryHook) {
		hook.level = level
	}
}

func WithFormatter(formatter logrus.Formatter) Option {
	return func(hook *SentryHook) {
		hook.formatter = formatter
	}
}

func WithTags(tags map[string]string) Option {
	return func(hook *SentryHook) {
		hook.tags = tags
	}
}

// NewSentryHook creates a hook to be added to an instance of logger
// and initializes the raven client.
// This method sets the timeout to 100 milliseconds.
func NewSentryHook(DSN string, opts ...Option) (*SentryHook, error) {
	client, err := sentrygo.NewClient(sentrygo.ClientOptions{
		Dsn: DSN,
	})
	if err != nil {
		return nil, err
	}
	return NewWithClientSentryHook(client, opts...)
}

// NewWithClientSentryHook creates a hook using an initialized sentrygo client.
// This method sets the timeout to 100 milliseconds.
func NewWithClientSentryHook(client *sentrygo.Client, opts ...Option) (*SentryHook, error) {
	hook := &SentryHook{
		Timeout: 100 * time.Millisecond,
		StacktraceConfiguration: StackTraceConfiguration{
			Enable:            false,
			Level:             logrus.WarnLevel,
			Skip:              6,
			Context:           0,
			InAppPrefixes:     nil,
			SendExceptionType: true,
		},
		flushTimeout: 3 * time.Second,
		client:       client,
	}
	levels := make([]logrus.Level, 4)
	levels[0] = logrus.WarnLevel
	levels[1] = logrus.FatalLevel
	levels[2] = logrus.ErrorLevel
	levels[3] = logrus.PanicLevel
	hook.levels = levels
	hook.formatter = &logrus.JSONFormatter{}
	for _, o := range opts {
		o(hook)
	}
	return hook, nil
}

// NewAsyncSentryHook creates a hook same as NewSentryHook, but in asynchronous
// mode.
func NewAsyncSentryHook(DSN string) (*SentryHook, error) {
	hook, err := NewSentryHook(DSN)
	return setAsync(hook), err
}

// Fire writes the log file to defined path or using the defined writer.
// User who run this function needs write permissions to the file or directory if the file does not yet exist.
func (hook *SentryHook) Fire(entry *logrus.Entry) error {
	fmt.Printf("start entry:%+v", entry)
	// We may be crashing the program, so should flush any buffered events.
	content := hook.createContent(entry)

	event := sentrygo.NewEvent()
	event.Message = bytesconv.BytesToString(content)
	event.Timestamp = entry.Time
	event.Level = severityMap[entry.Level]
	event.Platform = "Golang"
	event.Extra = entry.Data
	event.Tags = hook.tags

	if !hook.disableStacktrace {
		trace := sentrygo.NewStacktrace()
		if trace != nil {
			value := ""
			if entry.Caller != nil {
				value = entry.Caller.File
			}
			event.Exception = []sentrygo.Exception{{
				Type:       entry.Message,
				Value:      value,
				Stacktrace: trace,
			}}
		}
	}

	hub := hook.hub
	if hub == nil {
		hub = sentrygo.CurrentHub()
	}
	_ = hook.client.CaptureEvent(event, nil, hub.Scope())
	//if entry.Level > logrus.ErrorLevel {
		hook.client.Flush(hook.flushTimeout)
	//}

	return nil
}

func (hook *SentryHook) createContent(entry *logrus.Entry) []byte {
	msg, err := hook.formatter.Format(entry)
	if err != nil {
		return []byte("")
	}
	return msg
}

// Levels returns configured log levels.
func (hook *SentryHook) Levels() []logrus.Level {
	return hook.levels
}
