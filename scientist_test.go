package scientist

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func basicExperiment() *Experiment {
	e := New("basic")
	e.Use(func(ctx context.Context) (interface{}, error) {
		return 1, nil
	})

	e.Try(func(ctx context.Context) (interface{}, error) {
		return 2, nil
	})

	e.Behavior("three", func(ctx context.Context) (interface{}, error) {
		return 3, nil
	})

	e.Behavior("correct", func(ctx context.Context) (interface{}, error) {
		return 1, nil
	})
	return e
}

func TestRun(t *testing.T) {
	e := basicExperiment()
	r := Run(context.Background(), e, "control")
	if len(r.Errors) != 0 {
		t.Errorf("Unexpected experiment errors: %v", r.Errors)
	}

	if r.Control.Name != "control" {
		t.Errorf("Unexpected control observation name: %q", r.Control.Name)
	}

	if r.Control.Err != nil {
		t.Errorf("Expected no error, got: %v", r.Control.Err)
	}

	if r.Control.Value != 1 {
		t.Errorf("Bad value for 'control': %v", r.Control.Value)
	}

	assertObservationNames(t, "candidate", r.Candidates, []string{"candidate", "correct", "three"})
	assertObservationNames(t, "ignored", r.Ignored, []string{})
	assertObservationNames(t, "mismatched", r.Mismatched, []string{"candidate", "three"})

	candidatesMap := make(map[string]*Observation, len(r.Candidates))
	for _, o := range r.Candidates {
		candidatesMap[o.Name] = o
	}

	two, ok := candidatesMap["candidate"]
	if !ok {
		t.Errorf("No behavior 'candidate'")
	} else {
		if two.Err != nil {
			t.Errorf("Error for 'candidate': %v", two.Err)
		}

		if two.Value != 2 {
			t.Errorf("Bad value for 'candidate': %v", two.Value)
		}
	}

	three, ok := candidatesMap["three"]
	if !ok {
		t.Errorf("No behavior 'three'")
	} else {
		if three.Err != nil {
			t.Errorf("Error for 'three': %v", three.Err)
		}

		if three.Value != 3 {
			t.Errorf("Bad value for 'three': %v", three.Value)
		}
	}

	correct, ok := candidatesMap["correct"]
	if !ok {
		t.Errorf("No behavior 'correct'")
	} else {
		if correct.Err != nil {
			t.Errorf("Error for 'correct': %v", correct.Err)
		}

		if correct.Value != 1 {
			t.Errorf("Bad value for 'correct': %v", correct.Value)
		}
	}
}

func TestIgnore(t *testing.T) {
	e := basicExperiment()
	e.Ignore(func(control, candidate interface{}) (bool, error) {
		return candidate == 3, nil
	})
	r := Run(context.Background(), e, "control")
	if len(r.Errors) != 0 {
		t.Errorf("Unexpected experiment errors: %v", r.Errors)
	}

	assertObservationNames(t, "candidate", r.Candidates, []string{"candidate", "correct", "three"})
	assertObservationNames(t, "ignored", r.Ignored, []string{"three"})
	assertObservationNames(t, "mismatched", r.Mismatched, []string{"candidate"})
}

func TestCompare(t *testing.T) {
	e := basicExperiment()
	e.Compare(func(control, candidate interface{}) (bool, error) {
		return control == 1 && candidate == 3, nil
	})

	r := Run(context.Background(), e, "control")
	if len(r.Errors) != 0 {
		t.Errorf("Unexpected experiment errors: %v", r.Errors)
	}

	assertObservationNames(t, "candidate", r.Candidates, []string{"candidate", "correct", "three"})
	assertObservationNames(t, "ignored", r.Ignored, []string{})
	assertObservationNames(t, "mismatched", r.Mismatched, []string{"candidate", "correct"})
}

func TestCompareAndIgnore(t *testing.T) {
	e := basicExperiment()
	e.Compare(func(control, candidate interface{}) (bool, error) {
		return control == 1 && candidate == 3, nil
	})
	e.Ignore(func(control, candidate interface{}) (bool, error) {
		return candidate == 1, nil
	})
	r := Run(context.Background(), e, "control")
	if len(r.Errors) != 0 {
		t.Errorf("Unexpected experiment errors: %v", r.Errors)
	}

	assertObservationNames(t, "candidate", r.Candidates, []string{"candidate", "correct", "three"})
	assertObservationNames(t, "ignored", r.Ignored, []string{"correct"})
	assertObservationNames(t, "mismatched", r.Mismatched, []string{"candidate"})
}

func TestDefaultCleaner(t *testing.T) {
	e := New("cleaner")
	e.Use(func(ctx context.Context) (interface{}, error) {
		return "booya", nil
	})
	r := Run(context.Background(), e, "control")

	cleaned, err := r.Control.CleanedValue()
	if err != nil {
		t.Errorf("Unexpected cleaning error: %v", err)
	}

	if cleaned != "booya" {
		t.Errorf("bad cleaned value: %v", cleaned)
	}
}

func TestCustomCleaner(t *testing.T) {
	e := New("cleaner")
	e.Use(func(ctx context.Context) (interface{}, error) {
		return "booya", nil
	})
	e.Clean(func(v interface{}) (interface{}, error) {
		return strings.ToUpper(v.(string)), nil
	})
	r := Run(context.Background(), e, "control")

	cleaned, err := r.Control.CleanedValue()
	if err != nil {
		t.Errorf("Unexpected cleaning error: %v", err)
	}

	if cleaned != "BOOYA" {
		t.Errorf("bad cleaned value: %v", cleaned)
	}
}

func assertObservationNames(t *testing.T, key string, obs []*Observation, expected []string) {
	actual := observationNames(obs)
	if reflect.DeepEqual(expected, actual) {
		return
	}

	t.Errorf("Expected %s observations: %v, got: %v", key, expected, actual)
}

func observationNames(obs []*Observation) []string {
	names := make([]string, len(obs))
	for i, o := range obs {
		names[i] = o.Name
	}
	sort.Strings(names)
	return names
}
