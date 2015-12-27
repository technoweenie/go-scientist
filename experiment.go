package scientist

import "reflect"

func New(name string) *Experiment {
	return &Experiment{
		Name:       name,
		behaviors:  make(map[string]behaviorFunc),
		comparator: defaultComparator,
		runcheck:   defaultRunCheck,
	}
}

type behaviorFunc func() (value interface{}, err error)
type valueFunc func(control, candidate interface{}) (bool, error)
type checkFunc func() (bool, error)

type Experiment struct {
	Name       string
	Context    map[string]string
	behaviors  map[string]behaviorFunc
	ignores    []valueFunc
	comparator valueFunc
	runcheck   checkFunc
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

func (e *Experiment) RunIf(fn func() (bool, error)) {
	e.runcheck = checkFunc(fn)
}

func (e *Experiment) Enabled() (bool, error) {
	return e.runcheck()
}

func (e *Experiment) Run() (interface{}, error) {
	enabled, err := e.Enabled()
	if err != nil {
		return nil, err
	}

	if enabled {
		r := Run(e)
		return r.Control.Value, r.Control.Err
	}

	behavior, ok := e.behaviors[controlBehavior]
	if !ok {
		return nil, behaviorNotFound(e, controlBehavior)
	}

	return behavior()
}

func defaultComparator(candidate, control interface{}) (bool, error) {
	return reflect.DeepEqual(candidate, control), nil
}

func defaultRunCheck() (bool, error) {
	return true, nil
}
