package sentryhook

import (
	"runtime"

	sentrygo "github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	severityMap = map[logrus.Level]sentrygo.Level{
		logrus.TraceLevel: sentrygo.LevelDebug,
		logrus.DebugLevel: sentrygo.LevelDebug,
		logrus.InfoLevel:  sentrygo.LevelInfo,
		logrus.WarnLevel:  sentrygo.LevelWarning,
		logrus.ErrorLevel: sentrygo.LevelError,
		logrus.FatalLevel: sentrygo.LevelFatal,
		logrus.PanicLevel: sentrygo.LevelFatal,
	}
)


// The Stacktracer interface allows an error type to return a raven.Stacktrace.
type Stacktracer interface {
	GetStacktrace() *sentrygo.Stacktrace
}

type causer interface {
	Cause() error
}

type pkgErrorStackTracer interface {
	StackTrace() errors.StackTrace
}

// StackTraceConfiguration allows for configuring stacktraces
type StackTraceConfiguration struct {
	// whether stacktraces should be enabled
	Enable bool
	// the level at which to start capturing stacktraces
	Level logrus.Level
	// how many stack frames to skip before stacktrace starts recording
	Skip int
	// the number of lines to include around a stack frame for context
	Context int
	// the prefixes that will be matched against the stack frame.
	// if the stack frame's package matches one of these prefixes
	// sentry will identify the stack frame as "in_app"
	InAppPrefixes []string
	// whether sending exception type should be enabled.
	SendExceptionType bool
	// whether the exception type and message should be switched.
	SwitchExceptionTypeAndMessage bool
	// whether to include a breadcrumb with the full error stack
	IncludeErrorBreadcrumb bool
}

func setAsync(hook *SentryHook) *SentryHook {
	if hook == nil {
		return nil
	}
	hook.asynchronous = true
	return hook
}

// Flush waits for the log queue to empty. This function only does anything in
// asynchronous mode.
func (hook *SentryHook) Flush() {
	if !hook.asynchronous {
		return
	}
	hook.mu.Lock() // Claim exclusive access; any logging goroutines will block until the flush completes
	defer hook.mu.Unlock()

	hook.wg.Wait()
}

func (hook *SentryHook) findStacktrace(err error) *sentrygo.Stacktrace {
	var stacktrace *sentrygo.Stacktrace
	var stackErr errors.StackTrace
	for err != nil {
		// Find the earliest *raven.Stacktrace, or error.StackTrace
		if tracer, ok := err.(Stacktracer); ok {
			stacktrace = tracer.GetStacktrace()
			stackErr = nil
		} else if tracer, ok := err.(pkgErrorStackTracer); ok {
			stacktrace = nil
			stackErr = tracer.StackTrace()
		}
		if cause, ok := err.(causer); ok {
			err = cause.Cause()
		} else {
			break
		}
	}
	if stackErr != nil {
		stacktrace = hook.convertStackTrace(stackErr)
	}
	return stacktrace
}

// convertStackTrace converts an errors.StackTrace into a natively consumable
// *raven.Stacktrace
func (hook *SentryHook) convertStackTrace(st errors.StackTrace) *sentrygo.Stacktrace {
	stFrames := []errors.Frame(st)
	frames := make([]sentrygo.Frame, 0, len(stFrames))
	for i := range stFrames {
		pc := uintptr(stFrames[i])
		fn := runtime.FuncForPC(pc)
		file, line := fn.FileLine(pc)
		rFrame := runtime.Frame{
			PC:       pc,
			Func:     fn,
			Function: fn.Name(),
			File:     file,
			Line:     line,
			Entry:    fn.Entry(),
		}
		frame := sentrygo.NewFrame(rFrame)
		frames = append(frames, frame)
	}

	// Sentry wants the frames with the oldest first, so reverse them
	for i, j := 0, len(frames)-1; i < j; i, j = i+1, j-1 {
		frames[i], frames[j] = frames[j], frames[i]
	}
	return &sentrygo.Stacktrace{Frames: frames}
}

// utility classes for breadcrumb support
type Breadcrumbs struct {
	Values []Value `json:"values"`
}

type Value struct {
	Timestamp int64       `json:"timestamp"`
	Type      string      `json:"type"`
	Message   string      `json:"message"`
	Category  string      `json:"category"`
	Level     string      `json:"string"`
	Data      interface{} `json:"data"`
}

func (b *Breadcrumbs) Class() string {
	return "breadcrumbs"
}