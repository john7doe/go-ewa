# E.W.A - Errors With Attributes

![ewa.png](ewa.png)

## What is E.W.A?

E.W.A is a package that provides a way to add attributes to errors. So when they are logged they 
reassemble the structured logging your application is already using.

## Why E.W.A?

So you are using structured logging in you application, but its error messages are not structured 
and the logged errors are not as easy searchable as you would like them to be. 

instead of this:

```go
...
return nil, fmt.Errorf("error getting response from service (%s): %w", "some service", errorFromYourCode)
...
```

that produces log lines like this:

`level=INFO msg="error getting response from service (some service): timeout while calling /bar"`

or trying to retrofit structured logging in while trying to be DRY:

```go
...
err := fmt.Errorf("error getting response from service (%s): %w", "some service", errorFromYourCode)
slog.Error(err.Error(),  "serviceName", "some service")
return nil, err
...
```

you can use E.W.A:

```go
...
return nil, ewa.Wrap(ewaFromYourCode, "error getting response from service", "serviceName", "some service")
...
```

that produces log lines like this:

```log
level=INFO msg="error getting response from service: timeout while calling" serviceName="some service" stacktrace="github.com/john7doe/go-ewa_test.simTimeoutEwa\ngithub.com/john7doe/go-ewa_test.ExampleReadme\ntesting.runExample\ntesting.runExamples\ntesting.(*M).Run\nmain.main\nruntime.main\nruntime.goexit\n" url=/bar
```

## Inspiration

* https://dave.cheney.net/2016/04/27/dont-just-check-errors-handle-them-gracefully (in particular the "Only handle errors once" section)
* https://dave.cheney.net/2016/06/12/stack-traces-and-the-errors-package
