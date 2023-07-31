package ewa

import (
	"errors"
	"fmt"
	"runtime"
	"sort"
	"strings"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slog"
)

type Attrser interface {
	Attrs() []slog.Attr
}

type errorWithAttrs struct {
	msg   string
	attrs []slog.Attr
}

func (e *errorWithAttrs) Error() string {
	return e.msg
}

func (e *errorWithAttrs) Attrs() []slog.Attr {
	return e.attrs
}

type Stacktracer interface {
	StackTrace() string
}

func isStacktracer(err error) bool {
	var st Stacktracer
	return errors.As(err, &st)
}

type errorWithAttrsAndStackTrace struct {
	errorWithAttrs
	stack []uintptr
}

func (e *errorWithAttrsAndStackTrace) StackTrace() string {
	return stackAsString(e.stack)
}

func stackAsString(stack []uintptr) string {

	frames := runtime.CallersFrames(stack)
	var result = strings.Builder{}
	for {
		frame, more := frames.Next()
		_, _ = fmt.Fprintf(&result, "%s\n", frame.Function)
		if !more {
			break
		}
	}
	return result.String()
}

type errorWithAttrsAndParent struct {
	errorWithAttrs
	err error
}

type errorWithAttrsAndParentStackTrace struct {
	errorWithAttrsAndParent
	stack []uintptr
}

func (e *errorWithAttrsAndParentStackTrace) StackTrace() string {
	return stackAsString(e.stack)
}

func (e *errorWithAttrsAndParent) Error() string {
	return e.msg + ": " + e.err.Error()
}

func (e *errorWithAttrsAndParent) Unwrap() error {
	return e.err
}

func WrapAttrsS(err error, text string, attrs ...slog.Attr) error {
	return wrapAttrs(err, text, true, attrs...)

}

func WrapAttrs(err error, text string, attrs ...slog.Attr) error {
	return wrapAttrs(err, text, false, attrs...)

}

func wrapAttrs(err error, text string, stacktrace bool, attrs ...slog.Attr) error {
	attrParent := errorWithAttrsAndParent{
		errorWithAttrs: errorWithAttrs{
			msg:   text,
			attrs: attrs,
		},
		err: err,
	}

	// don't add stacktrace if not requested or if the error chain already has a stacktrace
	if !stacktrace || isStacktracer(err) {
		return &attrParent
	}

	return &errorWithAttrsAndParentStackTrace{
		errorWithAttrsAndParent: attrParent,
		stack:                   callers(),
	}
}

func Wrap(err error, text string, args ...any) error {
	attrs := argsToAttrs(args)
	return WrapAttrs(err, text, attrs...)
}

func WrapS(err error, text string, args ...any) error {
	attrs := argsToAttrs(args)
	return WrapAttrsS(err, text, attrs...)
}

func newAttrsS(text string, callers []uintptr, attrs ...slog.Attr) error {
	return &errorWithAttrsAndStackTrace{
		errorWithAttrs: errorWithAttrs{
			msg:   text,
			attrs: attrs,
		},
		stack: callers,
	}
}

func NewAttrsS(text string, attrs ...slog.Attr) error {
	return newAttrsS(text, callers(), attrs...)
}

func NewS(text string, args ...any) error {
	attrs := argsToAttrs(args)
	return newAttrsS(text, callers(), attrs...)
}

func NewAttrs(text string, attrs ...slog.Attr) error {
	return &errorWithAttrs{
		msg:   text,
		attrs: attrs,
	}
}

func New(text string, args ...any) error {
	attrs := argsToAttrs(args)
	return NewAttrs(text, attrs...)
}

func callers() []uintptr {
	stack := make([]uintptr, 42)
	_ = runtime.Callers(3, stack)
	return stack
}

// I'm lazy, for this poc reuse slog's Attr impl
func argsToAttrs(args []any) []slog.Attr {
	r := &slog.Record{}
	r.Add(args...)
	result := make([]slog.Attr, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		result = append(result, a)
		return true
	})
	return result
}

func LogInfo(err error, logger *slog.Logger) {
	message, attrs, ok := getAttrs(err)
	if ok {
		logger.LogAttrs(nil, slog.LevelInfo, message, attrs...)
	}
}

func getAttrs(err error) (string, []slog.Attr, bool) {
	// build list of attrs from err, if there are duplicate keys, then the one deepest in the chain wins
	// TODO: alternative key.# to make sure all are logged?
	if err == nil {
		return "", nil, false
	}

	message := err.Error()

	keyToAttr := make(map[string]slog.Attr)
	var deepestStackstace Stacktracer
	for {
		if errWithAttrs, ok := err.(Attrser); ok {
			for _, attr := range errWithAttrs.Attrs() {
				keyToAttr[attr.Key] = attr
			}
		}

		// Keep the deepest stacktrace
		if stacktracer, ok := err.(Stacktracer); ok {
			deepestStackstace = stacktracer
		}

		err = errors.Unwrap(err)
		if err == nil {
			break
		}
	}

	// Add stacktrace if there is one
	if deepestStackstace != nil {
		a := slog.String("stacktrace", deepestStackstace.StackTrace())
		keyToAttr[a.Key] = a
	}

	if len(keyToAttr) == 0 {
		return message, nil, true
	}

	// maps.Values(keyToAttr) is not stable, so we cant use it :-(
	keys := maps.Keys(keyToAttr)
	sort.Strings(keys)

	attrs := make([]slog.Attr, len(keys))
	for _, key := range keys {
		attrs = append(attrs, keyToAttr[key])
	}

	return message, attrs, true
}
