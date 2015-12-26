```go
expr := scientist.New("widget-permissions")

expr.Use(func() (interface{}, error) {
  return false, nil
})

expr.Try(func() (interface{}, error) {
  return true, nil
})

expr.Compare(func(control, candidate interface{}) bool {
  return control == candidate
})

expr.Clean(func(value interface{}) (interface{}, error) {
  return nil, nil
})

expr.Ignore(func(control, candidate interface{}) bool {
  return false
})

expr.RunIf(func() bool {

})

expr.SetContext("key", interface{})

value, err := expr.Run()
```
