package scientist

import (
  "testing"
)

func TestRun(t *testing.T) {
  e := New("run")
  e.Use(func () (interface{}, error) {
    return 1, nil
  })

  e.Try(func () (interface{}, error) {
    return 2, nil
  })

  value, err := e.Run()
  if err != nil {
    t.Errorf("Expected no error, got: %v", err)
  }

  if value != 1 {
    t.Errorf("Bad value: %v", value)
  }
}

func TestResult(t *testing.T) {
  e := New("basic")
  e.Use(func () (interface{}, error) {
    return 1, nil
  })

  e.Try(func () (interface{}, error) {
    return 2, nil
  })

  r := e.Result()

  if r.Control.Name != "control" {
    t.Errorf("Unexpected control observation name: %q", r.Control.Name)
  }

  if r.Control.Err != nil {
    t.Errorf("Expected no error, got: %v", r.Control.Err)
  }

  if r.Control.Value != 1 {
    t.Errorf("Bad value: %v", r.Control.Value)
  }
}
