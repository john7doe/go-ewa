package ewa

import (
	"errors"
	"runtime"
	"sort"

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

type errorWithAttrsAndStackTrace struct {
	errorWithAttrs
	stack []uintptr
}

func (e *errorWithAttrsAndStackTrace) StackTrace() string {
	return stackAsString(e.stack)
}

func stackAsString(stack []uintptr) string {
	var result string

	frames := runtime.CallersFrames(stack)
	for {
		frame, more := frames.Next()
		result += frame.Func.Name() + "\n"
		if !more {
			break
		}
	}
	return result
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

func WrapAttrs(err error, text string, attrs ...slog.Attr) error {
	attrParent := errorWithAttrsAndParent{
		errorWithAttrs: errorWithAttrs{
			msg:   text,
			attrs: attrs,
		},
		err: err,
	}

	hasStacktrace := isStacktracer(err)
	if hasStacktrace {
		return &attrParent
	}

	return &errorWithAttrsAndParentStackTrace{
		errorWithAttrsAndParent: attrParent,
		stack:                   callers(),
	}
}

func isStacktracer(err error) bool {
	if _, ok := err.(Stacktracer); ok {
		return true
	}
	parent := errors.Unwrap(err)
	if parent != nil {
		return isStacktracer(parent)
	}
	return false
}

func Wrap(err error, text string, args ...any) error {
	attrs := argsToAttrs(args)
	return WrapAttrs(err, text, attrs...)
}

func newAttrs(text string, callers []uintptr, attrs ...slog.Attr) error {
	return &errorWithAttrsAndStackTrace{
		errorWithAttrs: errorWithAttrs{
			msg:   text,
			attrs: attrs,
		},
		stack: callers,
	}
}

func NewAttrs(text string, attrs ...slog.Attr) error {
	return newAttrs(text, callers(), attrs...)
}

func New(text string, args ...any) error {
	attrs := argsToAttrs(args)
	return newAttrs(text, callers(), attrs...)
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
	for {
		if errWithAttrs, ok := err.(Attrser); ok {
			for _, attr := range errWithAttrs.Attrs() {
				keyToAttr[attr.Key] = attr
			}
		}

		if stacktracer, ok := err.(Stacktracer); ok {
			a := slog.String("stacktrace", stacktracer.StackTrace())
			keyToAttr[a.Key] = a
		}

		err = errors.Unwrap(err)
		if err == nil {
			break
		}
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
