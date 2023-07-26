package ewa_test

import (
	"fmt"
	"log"
	"os"

	"github.com/john7doe/go-ewa"
	"golang.org/x/exp/slog"
)

func removeTime(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey && len(groups) == 0 {
		return slog.Attr{}
	}
	return a
}

/*
Common patterns that we expect folks to convert from, so should we mimic them in the api we provide

e1 := errors.New("foo")
e2 := fmt.Errorf("foo: %w", e1)
e3 := errors.Wrap(e2, "bar")

e1 and e3: yes
e2: we want to stop embedding values into the error message, so im on a no for this one
*/
func ExampleUsage() {
	// do not include timestamp
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: removeTime}))
	slog.SetDefault(logger)

	ewa1 := ewa.New("test error", "key", "value", "key2", 2)
	ewa2 := ewa.NewAttrs("test error 2", slog.String("key", "value"), slog.Int("key2", 2))

	ewa3 := ewa.Wrap(ewa1, "wrap test error", "key3", "value3")
	ewa4 := ewa.WrapAttrs(ewa2, "wrap test error 2", slog.String("key3", "value3"))

	ewa.LogInfo(ewa3, slog.Default())
	ewa.LogInfo(ewa4, slog.Default())

	// Output:
	// level=INFO msg="wrap test error: test error" key=value key2=2 key3=value3
	// level=INFO msg="wrap test error 2: test error 2" key=value key2=2 key3=value3
}

func ExampleReadme() {
	// do not include timestamp
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: removeTime}))
	slog.SetDefault(logger)

	errorFromYourCode := fmt.Errorf("timeout while calling %s", "/bar")
	wrappedError := fmt.Errorf("error getting response from service (%s): %w", "some service", errorFromYourCode)
	log.Print(wrappedError)

	ewaFromYourCode := ewa.New("timeout while calling", "url", "/bar")
	ewaWrappedError := ewa.Wrap(ewaFromYourCode, "error getting response from service", "serviceName", "some service")

	ewa.LogInfo(ewaWrappedError, slog.Default())
	// Output:
	// level=INFO msg="error getting response from service (some service): timeout while calling /bar"
	// level=INFO msg="error getting response from service: timeout while calling" serviceName="some service" url=/bar

}
