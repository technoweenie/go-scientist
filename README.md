## Scientist

A Go port of a great Ruby library for carefully refactoring critical paths.
Check out the original: https://github.com/github/scientist

For a detailed look at actually using this thing, check out this blog post:
[Move fast and fix things](http://githubengineering.com/move-fast/).

NOTE: This port is an experiment in porting a small Ruby lib to Go. While I think
the differences in the languages result in interesting comparisons and contrasts
between the approaches, the Go version is _not_ used in production anywhere.
Consider this alpha, unsupported software. The Ruby version, however, is very
stable.

## How do I science?

Let's pretend you're changing the way you handle permissions in a large web app. Tests can help guide your refactoring, but you really want to compare the current and refactored behaviors under load.

```go
package permissions

import "scientist"

type Widget struct {
  ...
}

func (w *Widget) Allows(u *User) (bool, error) {
  experiment := scientist.New("widget-permissions")
  // old way
  experiment.Use(func() (interface{}, error) {
    return w.IsValid(u), nil
  })
  // new way
  experiment.Try(func() (interface{}, error) {
    return u.Can("read", w), nil
  })

  return scientist.Bool(experiment.Run())
}
```

Write a `Use` callback around the code's original behavior, and a `Try` around the new behavior. `experiment.Run()` will always return whatever the `Use` callback returns, but it does a bunch of stuff behind the scenes:

* It decides whether or not to run the `Try` callback,
* Randomizes the order in which `Use` and `Try` callbacks are run,
* Measures the durations of all behaviors,
* Compares the result of `Try` to the result of `Use`,
* Swallows (but records) any errors in the `Try` callback, and
* Publishes all this information.

The `Use` callback is called the **control**. The `Try` callback is called the **candidate**.

TODO: mention helpers like scientist.Bool()

If you don't declare any `Try` callbacks, none of the Scientist machinery is invoked and the control value is always returned.

Experiments do not attempt to recover from any runtime panics, and are not
goroutine safe. Any `*scientist.Experiment` objects should be Run and discarded
immediately after being initialized. Ideally, your application should already
handle any runtime panics somehow.

## Making science useful

The examples above will run, but they're not really *doing* anything. The `Try` callbacks run every time and none of the results get published. Replace the default experiment implementation to control execution and reporting:

```go
package permissions

import "scientist"

type Widget struct {
  ...
}

func (w *Widget) Allows(u *User) (bool, error) {
  experiment := Experiment("widget-permissions")
  experiment.Use(func() (interface{}, error) {
    return w.IsValid(u), nil
  })
  experiment.Try(func() (interface{}, error) {
    u.Can("read", w)
  })

  return scientist.Bool(experiment.Run())
}

// experiment constructor for all uses in the "permissions" package
func Experiment(name string) *scientist.Experiment {
  experiment := scientist.New("widget-permissions")
  experiment.RunIf(func() (bool, error) {
    // see "Ramping up experiments" below
    return true, nil
  })

  experiment.Publish(func(r scientist.Result) error {
    // see "Publishing results" below
    // post to graphite/redis/librato/etc
    return nil
  })

  experiment.ReportErrors(func(errs ...scientist.ResultError) {
    // post to sentry or other error reporting tool
  })
  return experiment
}
```

Now calls to the `Experiment()` function return a `*scientist.Experiment` with
common callbacks for ramping up experiments, publishing results, and reporting
errors.

### Controlling comparison

Scientist compares control and candidate values using `reflect.DeepEqual()`. To override this behavior, set a `Compare` callback to define how to compare observed values instead:

```go
func (w *Widget) Allows(u *User) (bool, error) {
  experiment := Experiment("widget-permissions")
  experiment.Use(func() (interface{}, error) {
    return w.IsValid(u), nil
  })
  experiment.Try(func() (interface{}, error) {
    u.Can("read", w)
  })

  experiment.Compare(func(control, candidate interface{}) (bool, error) {
    // cast as user, return login, or convert to string
    getLogin = func(value interface{}) string {
      if user, ok := value.(*User); ok {
        return user.Login
      }
      return fmt.Sprintf("%v", value)
    }

    return getLogin(control) == getLogin(candidate), nil
  })

  return scientist.Bool(experiment.Run())
}
```

### Adding context

Results aren't very useful without some way to identify them. Use the `context` method to add to or retrieve the context for an experiment:

```go
experiment := Experiment("widget-permissions")
experiment.Use(func() (interface{}, error) {
  return w.IsValid(u), nil
})
experiment.Try(func() (interface{}, error) {
  u.Can("read", w)
})
experiment.Context["user"] = fmt.Sprintf("%d", user.Id)
```

`Context` is a string-keyed map of string values. The data is available in the `Publish` callback.

### Expensive setup

If an experiment requires expensive setup that should only occur when the experiment is going to be run, define it with the `before_run` method:

```go
experiment := Experiment("widget-permissions")
experiment.Use(func() (interface{}, error) {
  return w.IsValid(u), nil
})
experiment.BeforeRun(func() error {
  // something expensive...
  return nil
})
experiment.Try(func() (interface{}, error) {
  u.Can("read", w)
})
```

### Keeping it clean

Sometimes you don't want to store the full value for later analysis. For example, an experiment may return `User` instances, but when researching a mismatch, all you care about is the logins. You can define how to clean these values in an experiment:

```go
experiment := Experiment("widget-permissions")
experiment.Use(func() (interface{}, error) {
  return w.IsValid(u), nil
})
experiment.Try(func() (interface{}, error) {
  u.Can("read", w)
})

experiment.Clean(func(value interface{}) (interface{}, error) {
  switch arr := value.(type) {
  case []*User:
    logins := make([]string, len(arr))
    for i, u := range arr {
      logins[i] = u.Login
    }
    sort.Strings(logins)
    return logins, nil
  default:
    return value, nil
  }
})
```

And this cleaned value is available in observations in the final published result:

```go
experiment.Publish(func(result scientist.Result) {
  result.Control.Value          // [*User, *User, *User]
  result.Control.CleanedValue() // ["alice", "bob", "carol"]
})
```

### Ignoring mismatches

During the early stages of an experiment, it's possible that some of your code will always generate a mismatch for reasons you know and understand but haven't yet fixed. Instead of these known cases always showing up as mismatches in your metrics or analysis, you can tell an experiment whether or not to ignore a mismatch using an `Ignore` callback. You may include more than one callback if needed:

```go
func (w *Widget) IsAdmin(u *User) (bool, error) {
  experiment := Experiment("widget-permissions")
  experiment.Use(func() (interface{}, error) {
    return w.IsAdmin(u), nil
  })
  experiment.Try(func() (interface{}, error) {
    u.Can("admin", w)
  })

  experiment.Ignore(func(control, candidate interface{}) (bool, error) {
    return u.IsStaff, nil
  })

  experiment.Ignore(func(control, candidate interface{}) (bool, error) {
    return control != nil && candidate == nil && !u.HasConfirmedEmail, nil
  })
  return scientist.Bool(experiment.Run())
}
```

The ignore callbacks are only called if the *values* don't match. If one observation returns an error and the other doesn't, it's always considered a mismatch. If both observations return different errors, that is also considered a mismatch.

### Ramping up experiments

Sometimes you don't want an experiment to run. Say, disabling a new codepath for anyone who isn't staff. You can disable an experiment by setting a `RunIf` callback. If this returns `false`, the experiment will merely return the control value.

```go
experiment := Experiment("widget-permissions")
experiment.RunIf(func() (bool, error) {
  return currentUser.IsStaff, nil
})
```

As a scientist, you know it's always important to be able to turn your experiment off, lest it run amok and result in villagers with pitchforks on your doorstep.

```go
experiment := Experiment("widget-permissions")
experiment.RunIf(func() (bool, error) {
  // track this in a databae, env var, etc
  // flipper isn't ported to Go... YET
  percentEnabled, err := flipper.PercentEnabled()
  if err != nil {
    return false, err
  }

  return percentEnabled > 0 && rand.Intn(100) < percentEnabled, nil
})
```

This code will be invoked for every method with an experiment every time, so be sensitive about its performance. For example, you can store an experiment in the database but wrap it in various levels of caching such as memcache or a per-request context.

### Publishing results

What good is science if you can't publish your results?

You must implement the `Publish` callback, and can publish data however you like. For example, timing data can be sent to graphite, and mismatches can be placed in a capped collection in redis for debugging later.

The `Publish` callback is given a `scientist.Result` instance with its associated `*scientist.Observation`s:

```go
// Globally setup somewhere...
// Example uses https://github.com/peterbourgon/g2s
statsd, _ := g2s.Dial("udp", "statsd-server:8125")

// The actual experiment
experiment := Experiment("widget-permissions")
experiment.Publish(func(r scientist.Result) error {
  statsd.Timing(1.0, fmt.Sprintf("science.%s.control", r.Experiment.Name), r.Control.Runtime)
  statsd.Timing(1.0, fmt.Sprintf("science.%s.candidate", r.Experiment.Name), r.Candidates[0].Runtime)
})
```

### Testing

When running your test suite, it's helpful to know that the experimental results always match. To help with testing, Scientist has a ErrorOnMismatches bool value
to set either on the `scientist` package, or on a `*scientist.Experiment`:

To raise on mismatches:

```go
// do this in a *_test.go file so it's set on tests only
import "scientist"

func init() {
  scientist.ErrorOnMismatches = true
}

// or enable it for a specific experiment only
experiment := scientist.New("something")
experiment.ErrorOnMismatches = true
// ... implementation
```

Scientist will raise a `scientist.MismatchError` error if any observations don't
match.

### Handling errors

If an exception is raised within any of scientist's internal callbacks, like `Publish`, `Compare`, or `Clean`, the `ReportErrors` method is called with a slice of errors, each containing the string name of the internal operation that failed and the error that was returned. The default behavior is to dump the errors to STDERR.

```go
experiment := Experiment("widget-permissions")
experiment.ReportErrors(func(errs ...scientist.ResultError) {
  for _, resErr := range errs {
    errortracker.Track(resErr.Err, "science failure in %s: %s", resErr.Experiment, resErr.Operation)
  }
})
```

The operations that may be handled here are:

* `before_run` - an error returned in a `BeforeRun` callback
* `clean` - an exception is raised in a `Clean` callback
* `compare` - an exception is raised in a `Compare` callback
* `ignore` - an exception is raised in an `Ignore` callback
* `publish` - an exception is raised in the `Publish` callback
* `run_if` - an exception is raised in a `RunIf` callback

### Designing an experiment

Because the `RunIf` callback determines when a candidate runs, it's impossible to guarantee that it will run every time. For this reason, Scientist is only safe for wrapping methods that aren't changing data.

When using Scientist, we've found it most useful to modify both the existing and new systems simultaneously anywhere writes happen, and verify the results at read time with `science`. `raise_on_mismatches` has also been useful to ensure that the correct data was written during tests, and reviewing published mismatches has helped us find any situations we overlooked with our production data at runtime. When writing to and reading from two systems, it's also useful to write some data reconciliation scripts to verify and clean up production data alongside any running experiments.

### Finishing an experiment

As your candidate behavior converges on the controls, you'll start thinking about removing an experiment and using the new behavior.

* If there are any `ignore` callbacks, the candidate behavior is *guaranteed* to be different. If this is unacceptable, you'll need to remove the `ignore` callbacks
and resolve any ongoing mismatches in behavior until the observations match
perfectly every time.
* When removing a read-behavior experiment, it's a good idea to keep any write-side duplication between an old and new system in place until well after the new behavior has been in production, in case you need to roll back.

## Breaking the rules

Sometimes scientists just gotta do weird stuff. We understand.

### Ignoring results entirely

Science is useful even when all you care about is the timing data or even whether or not a new code path blew up. If you have the ability to incrementally control how often an experiment runs via your `RunIf` callback, you can use it to silently and carefully test new code paths and ignore the results altogether. You can do this
by:

```go
experiment.Compare(func(control, candidate interface{}) (bool, error) {
  return true, nil
})
```

This will still log mismatches if any errors are returned, but will disregard the values entirely.

TODO: Confirm with unit test!

### Trying more than one thing

It's not usually a good idea to try more than one alternative simultaneously. Behavior isn't guaranteed to be isolated and reporting + visualization get quite a bit harder. Still, it's sometimes useful.

To try more than one alternative at once, add names to some `Behavior` callbacks:

```go
experiment := scientist.New("widget-permissions")
experiment.Use(func() (interface{}, error) {
  return w.IsValid(u), nil
})

// new service API
experiment.Behavior("api", func() (interface{}, error) {
  return u.Can("read", w), nil
})

// raw query
experiment.Behavior("raw-sql", func() (interface{}, error) {
  return u.CanSql("read", w), nil
})
```

When the experiment runs, all candidate behaviors are tested and each candidate observation is compared with the control in turn.

### No control, just candidates

Define the candidates with named `Behavior` callbacks, omit a `Use`, and pass a candidate name to `run`:

```go
experiment := scientist.New("widget-permissions")
experiment.Use(func() (interface{}, error) {
  return w.IsValid(u), nil
})

// new service API
experiment.Behavior("api", func() (interface{}, error) {
  return u.Can("read", w), nil
})

// raw query
experiment.Behavior("raw-sql", func() (interface{}, error) {
  return u.CanSql("read", w), nil
})

experiment.RunBehavior("second-way")
```

## Hacking

Run `go fmt` before committing. `go test` runs the unit tests. The scientist
package was written on Go 1.5+, but may work on older Go 1.x versions.

## Maintainers

nope.
