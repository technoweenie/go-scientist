```go
expr := scientist.New("widget-permissions")

expr.Use(func() (interface{}, error) {
  return false, nil
})

expr.Try(func() (interface{}, error) {
  return true, nil
})

expr.Behavior("candidate-2", func() (interface{}, error) {
  return true, nil
})

expr.Compare(func(control, candidate interface{}) (bool, error) {
  return control == candidate, nil
})

expr.Clean(func(value interface{}) (interface{}, error) {
  return nil, nil
})

expr.Ignore(func(control, candidate interface{}) (bool, error) {
  return false, nil
})

expr.RunIf(func() (bool, error) {
  return true, nil
})

expr.Publish(func(r scientist.Result) error {
  // post to graphite/librato/etc
  return nil
})

expr.ReportErrors(func(errors ...scientist.ResultError) {
  // post to error tracking service (sentry)
})

expr.Context["key"] = "value"

value, err := expr.Run()
```
