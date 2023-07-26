package ewa

import (
	"errors"
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

type errorWithAttrsAndParent struct {
	errorWithAttrs
	err error
}

func (e *errorWithAttrsAndParent) Error() string {
	return e.msg + ": " + e.err.Error()
}

func (e *errorWithAttrsAndParent) Unwrap() error {
	return e.err
}

func WrapAttrs(err error, text string, attrs ...slog.Attr) error {
	return &errorWithAttrsAndParent{
		errorWithAttrs: errorWithAttrs{
			msg:   text,
			attrs: attrs,
		},
		err: err,
	}
}

func Wrap(err error, text string, args ...any) error {
	attrs := argsToAttrs(args)
	return WrapAttrs(err, text, attrs...)
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
	// TODO: stacktrace as attrs?
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
