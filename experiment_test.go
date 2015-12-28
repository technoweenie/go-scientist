package scientist

import (
	"fmt"
	"testing"
)

func TestExperimentRunBefore(t *testing.T) {
	runIf := false
	before := false

	e := New("run")
	e.Use(func() (interface{}, error) {
		return 1, nil
	})
	e.Try(func() (interface{}, error) {
		return 1, nil
	})

	e.RunIf(func() (bool, error) {
		runIf = true
		return true, nil
	})

	e.BeforeRun(func() error {
		before = true
		return nil
	})

	v, err := e.Run()
	if v != 1 {
		t.Errorf("Unexpected control value: %d", v)
	}

	if err != nil {
		t.Errorf("Unexpected control error: %v", err)
	}

	if !runIf {
		t.Errorf("expected RunIf callback to run")
	}

	if !before {
		t.Errorf("expected BeforeRun callback to run")
	}
}

func TestExperimentDisabledRunBefore(t *testing.T) {
	runIf := false

	e := New("run")
	e.Use(func() (interface{}, error) {
		return 1, nil
	})
	e.Try(func() (interface{}, error) {
		return 1, nil
	})

	e.RunIf(func() (bool, error) {
		runIf = true
		return false, nil
	})

	e.BeforeRun(func() error {
		t.Errorf("did not expect BeforeRun callback to run")
		return nil
	})

	v, err := e.Run()
	if v != 1 {
		t.Errorf("Unexpected control value: %d", v)
	}

	if err != nil {
		t.Errorf("Unexpected control error: %v", err)
	}

	if !runIf {
		t.Errorf("expected RunIf callback to run")
	}
}

func TestExperimentEmptyRunBefore(t *testing.T) {
	runIf := false

	e := New("run")
	e.Use(func() (interface{}, error) {
		return 1, nil
	})

	e.RunIf(func() (bool, error) {
		runIf = true
		return true, nil
	})

	e.BeforeRun(func() error {
		t.Errorf("did not expect BeforeRun callback to run")
		return nil
	})

	v, err := e.Run()
	if v != 1 {
		t.Errorf("Unexpected control value: %d", v)
	}

	if err != nil {
		t.Errorf("Unexpected control error: %v", err)
	}

	if !runIf {
		t.Errorf("expected RunIf callback to run")
	}
}

func TestExperimentRunIfError(t *testing.T) {
	reported := false
	e := New("run")
	e.Use(func() (interface{}, error) {
		return 1, nil
	})

	e.Try(func() (interface{}, error) {
		t.Errorf("did not expect to run experiment if RunIf() returns error")
		return 1, nil
	})

	e.Publish(func(r Result) error {
		t.Errorf("did not expect to publish")
		return nil
	})

	e.ReportErrors(func(errors ...ResultError) {
		for _, err := range errors {
			switch err.Operation {
			case "run_if":
				reported = true
				if err.BehaviorName != "experiment" {
					t.Errorf("Bad behavior name for run_if operation: %q", err.BehaviorName)
				}
				if err.Index != -1 {
					t.Errorf("Bad index for run_if operation: %d", err.Index)
				}
				if actual := err.Error(); actual != "run_if" {
					t.Errorf("Bad error message for run_if operation: %q", actual)
				}
			default:
				t.Errorf("Bad operation: %q", err.Operation)
			}
		}
	})

	e.RunIf(func() (bool, error) {
		return true, fmt.Errorf("run_if")
	})

	v, err := e.Run()
	if v != nil {
		t.Errorf("unexpected result: %v", v)
	}

	if err == nil {
		t.Errorf("expected a run_if error!")
	} else if err.Error() != "run_if" {
		t.Errorf("unexpected error: %v", err.Error())
	}

	if !reported {
		t.Errorf("result errors never reported!")
	}
}
