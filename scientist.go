package scientist

import (
	"fmt"
	"reflect"
	"time"
)

const (
	controlBehavior   = "control"
	candidateBehavior = "candidate"
)

var (
	defaultComparator = func(candidate, control interface{}) (bool, error) {
		return reflect.DeepEqual(candidate, control), nil
	}
)

func New(name string) *Experiment {
	return &Experiment{
		Name:       name,
		behaviors:  make(map[string]behaviorFunc),
		comparator: defaultComparator,
	}
}

type behaviorFunc func() (value interface{}, err error)
type valueFunc func(control, candidate interface{}) (bool, error)

type Experiment struct {
	Name       string
	Context    map[string]string
	behaviors  map[string]behaviorFunc
	ignores    []valueFunc
	comparator valueFunc
}

func (e *Experiment) Use(fn func() (interface{}, error)) {
	e.Behavior(controlBehavior, fn)
}

func (e *Experiment) Try(fn func() (interface{}, error)) {
	e.Behavior(candidateBehavior, fn)
}

func (e *Experiment) Behavior(name string, fn func() (interface{}, error)) {
	e.behaviors[name] = behaviorFunc(fn)
}

func (e *Experiment) Compare(fn func(control, candidate interface{}) (bool, error)) {
	e.comparator = valueFunc(fn)
}

func (e *Experiment) Ignore(fn func(control, candidate interface{}) (bool, error)) {
	e.ignores = append(e.ignores, valueFunc(fn))
}

func (e *Experiment) Run() (interface{}, error) {
	r := Run(e)
	return r.Control.Value, r.Control.Err
}

func Run(e *Experiment) Result {
	r := Result{Experiment: e}
	numCandidates := len(e.behaviors) - 1
	r.Control = observe(e, controlBehavior, e.behaviors[controlBehavior])
	r.Candidates = make([]Observation, numCandidates)
	r.Ignored = make([]Observation, 0, numCandidates)
	r.Mismatched = make([]Observation, 0, numCandidates)

	i := 0
	for name, b := range e.behaviors {
		if name == controlBehavior {
			continue
		}
		c := observe(e, name, b)
		r.Candidates[i] = c
		i += 1

		mismatched, err := mismatching(e, r.Control, c)
		if err != nil {
			mismatched = true
			r.Errors = append(r.Errors, &resultError{"compare", name, -1, err})
		}

		if !mismatched {
			continue
		}

		ignored, idx, err := ignoring(e, r.Control, c)
		if err != nil {
			ignored = false
			r.Errors = append(r.Errors, &resultError{"ignore", name, idx, err})
		}

		if ignored {
			r.Ignored = append(r.Ignored, c)
		} else {
			r.Mismatched = append(r.Mismatched, c)
		}
	}

	return r
}

func mismatching(e *Experiment, control, candidate Observation) (bool, error) {
	matching, err := e.comparator(control.Value, candidate.Value)
	return !matching, err
}

func ignoring(e *Experiment, control, candidate Observation) (bool, int, error) {
	for idx, i := range e.ignores {
		ok, err := i(control.Value, candidate.Value)
		if err != nil {
			return false, idx, err
		}

		if ok {
			return true, idx, nil
		}
	}

	return false, -1, nil
}

type Observation struct {
	Experiment *Experiment
	Name       string
	Started    time.Time
	Runtime    time.Duration
	Value      interface{}
	Err        error
}

func observe(e *Experiment, name string, b behaviorFunc) Observation {
	o := Observation{
		Experiment: e,
		Name:       name,
		Started:    time.Now(),
	}

	if b == nil {
		b = e.behaviors[name]
	}

	if b == nil {
		o.Err = fmt.Errorf("Behavior %q not found for experiment %q", name, e.Name)
		o.Runtime = time.Since(o.Started)
	} else {
		v, err := b()
		o.Runtime = time.Since(o.Started)
		o.Value = v
		o.Err = err
	}

	return o
}

type resultError struct {
	Operation    string
	BehaviorName string
	Index        int
	Err          error
}

func (e *resultError) Error() string {
	return e.Err.Error()
}

type Result struct {
	Experiment *Experiment
	Control    Observation
	Candidates []Observation
	Ignored    []Observation
	Mismatched []Observation
	Errors     []*resultError
}
