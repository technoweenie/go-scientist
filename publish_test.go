package scientist

import (
	"fmt"
	"testing"
)

func TestPublish(t *testing.T) {
	e := New("publish")
	e.Use(func() (interface{}, error) {
		return 1, nil
	})
	e.Try(func() (interface{}, error) {
		return 2, nil
	})

	published := false
	reported := false
	e.Publish(func(r Result) error {
		published = true

		if r.Experiment.Name != "publish" {
			t.Errorf("Bad experiment name: %q", r.Experiment.Name)
		}

		return nil
	})

	e.ReportErrors(func(errors ...ResultError) {
		reported = true
		t.Errorf("result errors reported :(")
	})

	v, err := e.Run()
	if v != 1 {
		t.Errorf("Unexpected control value: %d", v)
	}

	if err != nil {
		t.Errorf("Unexpected control error: %v", err)
	}

	if !published {
		t.Errorf("results never published")
	}

	if reported {
		t.Errorf("result errors reported :(")
	}
}

func TestPublishWithErrors(t *testing.T) {
	e := New("publish")
	e.Use(func() (interface{}, error) {
		return 1, nil
	})
	e.Try(func() (interface{}, error) {
		return 2, nil
	})
	e.BeforeRun(func() error {
		return fmt.Errorf("(before)")
	})
	e.Compare(func(control, candidate interface{}) (bool, error) {
		return true, fmt.Errorf("(compare) candidate: %d", candidate)
	})
	// ignore callback 0, no error
	e.Ignore(func(control, candidate interface{}) (bool, error) {
		return false, nil
	})
	// ignore callback 1, returns an error
	e.Ignore(func(control, candidate interface{}) (bool, error) {
		return true, fmt.Errorf("(ignore) candidate: %d", candidate)
	})

	published := false
	reported := make(map[string]int)
	e.Publish(func(r Result) error {
		published = true
		return fmt.Errorf("(publish) result: %s", r.Experiment.Name)
	})

	e.ReportErrors(func(errors ...ResultError) {
		for _, err := range errors {
			reported[err.Operation] = reported[err.Operation] + 1
			switch err.Operation {
			case "compare":
				if err.BehaviorName != "candidate" {
					t.Errorf("Bad behavior name for compare operation: %q", err.BehaviorName)
				}
				if err.Index != -1 {
					t.Errorf("Bad index for compare operation: %d", err.Index)
				}
				if actual := err.Error(); actual != "(compare) candidate: 2" {
					t.Errorf("Bad error message for compare operation: %q", actual)
				}
			case "ignore":
				if err.BehaviorName != "candidate" {
					t.Errorf("Bad behavior name for ignore operation: %q", err.BehaviorName)
				}
				if err.Index != 1 {
					t.Errorf("Bad index for ignore operation: %d", err.Index)
				}
				if actual := err.Error(); actual != "(ignore) candidate: 2" {
					t.Errorf("Bad error message for ignore operation: %q", actual)
				}
			case "publish":
				if err.BehaviorName != "experiment" {
					t.Errorf("Bad behavior name for publish operation: %q", err.BehaviorName)
				}
				if err.Index != -1 {
					t.Errorf("Bad index for publish operation: %d", err.Index)
				}
				if actual := err.Error(); actual != "(publish) result: publish" {
					t.Errorf("Bad error message for publish operation: %q", actual)
				}
			case "before_run":
				if err.BehaviorName != "experiment" {
					t.Errorf("Bad behavior name for before_run operation: %q", err.BehaviorName)
				}
				if err.Index != -1 {
					t.Errorf("Bad index for before_run operation: %d", err.Index)
				}
				if actual := err.Error(); actual != "(before)" {
					t.Errorf("Bad error message for before_run operation: %q", actual)
				}
			default:
				t.Errorf("Bad operation: %q", err.Operation)
			}
		}
	})

	v, err := e.Run()
	if v != 1 {
		t.Errorf("Unexpected control value: %d", v)
	}

	if err != nil {
		t.Errorf("Unexpected control error: %v", err)
	}

	if !published {
		t.Errorf("results never published")
	}

	if len(reported) != 4 {
		t.Errorf("all result errors not reported: %v", reported)
	}

	for key, times := range reported {
		if times != 1 {
			t.Errorf("%q errors reported %d times", key, times)
		}
	}
}
