package scientist

import "fmt"

const (
	controlBehavior   = "control"
	candidateBehavior = "candidate"
)

func New(name string) *Experiment {
	return &Experiment{
		Name:      name,
		behaviors: make(map[string]Behavior),
	}
}

type Behavior func() (interface{}, error)

type Experiment struct {
	Name      string
	behaviors map[string]Behavior
}

func (e *Experiment) Use(fn func() (interface{}, error)) {
	e.Behavior(controlBehavior, fn)
}

func (e *Experiment) Try(fn func() (interface{}, error)) {
	e.Behavior(candidateBehavior, fn)
}

func (e *Experiment) Behavior(name string, fn func() (interface{}, error)) {
	e.behaviors[name] = Behavior(fn)
}

func (e *Experiment) Run() (interface{}, error) {
	r := Run(e)
	return r.Control.Value, r.Control.Err
}

func Run(e *Experiment) Result {
	r := Result{Experiment: e}
	r.Control = observe(e, controlBehavior, e.behaviors[controlBehavior])
	r.Candidates = make([]Observation, len(e.behaviors)-1)
	i := 0
	for name, b := range e.behaviors {
		if name == controlBehavior {
			continue
		}
		r.Candidates[i] = observe(e, name, b)
		i += 1
	}

	return r
}

type Observation struct {
	Experiment *Experiment
	Name       string
	Value      interface{}
	Err        error
}

func observe(e *Experiment, name string, b Behavior) Observation {
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
