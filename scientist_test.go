package scientist

import (
	"testing"
)

func TestRun(t *testing.T) {
	e := New("basic")
	e.Use(func() (interface{}, error) {
		return 1, nil
	})

	e.Try(func() (interface{}, error) {
		return 2, nil
	})

	e.Behavior("three", func() (interface{}, error) {
		return 3, nil
	})

	r := Run(e)

	if r.Control.Name != "control" {
		t.Errorf("Unexpected control observation name: %q", r.Control.Name)
	}

	if r.Control.Err != nil {
		t.Errorf("Expected no error, got: %v", r.Control.Err)
	}

	if r.Control.Value != 1 {
		t.Errorf("Bad value for 'control': %v", r.Control.Value)
	}

	if candidates := len(r.Candidates); candidates != 2 {
		t.Errorf("Wrong number of candidates: %d", candidates)
	}

	candidatesMap := make(map[string]Observation, len(r.Candidates))
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
}
