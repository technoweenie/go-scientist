package scientist

import (
	"fmt"
	"reflect"
)

const (
	controlBehavior   = "control"
	candidateBehavior = "candidate"
)

func New(name string) *Experiment {
	return &Experiment{
		Name:       name,
		behaviors:  make(map[string]behaviorFunc),
		comparator: reflect.DeepEqual,
	}
}

type behaviorFunc func() (value interface{}, err error)
type valueFunc func(candidate, control interface{}) bool

type Experiment struct {
	Name       string
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

func (e *Experiment) Compare(fn func(candidate, control interface{}) bool) {
	e.comparator = valueFunc(fn)
}

func (e *Experiment) Ignore(fn func(candidate, control interface{}) bool) {
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

		if mismatching(e, r.Control, c) {
			if ignoring(e, r.Control, c) {
				r.Ignored = append(r.Ignored, c)
			} else {
				r.Mismatched = append(r.Mismatched, c)
			}
		}

		i += 1
	}

	return r
}

func mismatching(e *Experiment, control, candidate Observation) bool {
	return !e.comparator(control.Value, candidate.Value)
}

func ignoring(e *Experiment, control, candidate Observation) bool {
	for _, i := range e.ignores {
		if i(control.Value, candidate.Value) {
			return true
		}
	}

	return false
}

type Observation struct {
	Experiment *Experiment
	Name       string
	Value      interface{}
	Err        error
}

func observe(e *Experiment, name string, b behaviorFunc) Observation {
	o := Observation{Experiment: e, Name: name}
	if b == nil {
		b = e.behaviors[name]
	}

	if b == nil {
		o.Err = fmt.Errorf("Behavior %q not found for experiment %q", name, e.Name)
	} else {
		v, err := b()
		o.Value = v
		o.Err = err
	}

	return o
}

type Result struct {
	Experiment *Experiment
	Control    Observation
	Candidates []Observation
	Ignored    []Observation
	Mismatched []Observation
}
