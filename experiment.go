package scientist

import "reflect"

func New(name string) *Experiment {
	return &Experiment{
		Name:          name,
		behaviors:     make(map[string]behaviorFunc),
		comparator:    defaultComparator,
		runcheck:      defaultRunCheck,
		publisher:     defaultPublisher,
		errorReporter: defaultErrorReporter,
		beforeRun:     defaultBeforeRun,
		cleaner:       defaultCleaner,
	}
}

type behaviorFunc func() (value interface{}, err error)

type Experiment struct {
	Name          string
	Context       map[string]string
	behaviors     map[string]behaviorFunc
	ignores       []func(control, candidate interface{}) (bool, error)
	comparator    func(control, candidate interface{}) (bool, error)
	runcheck      func() (bool, error)
	publisher     func(Result) error
	errorReporter func(...ResultError)
	beforeRun     func() error
	cleaner       func(interface{}) (interface{}, error)
}

func (e *Experiment) Use(fn func() (interface{}, error)) {
	e.Behavior(controlBehavior, fn)
}

func (e *Experiment) Try(fn func() (interface{}, error)) {
	e.Behavior(candidateBehavior, fn)
}

func (e *Experiment) Behavior(name string, fn func() (interface{}, error)) {
	e.behaviors[name] = fn
}

func (e *Experiment) Compare(fn func(control, candidate interface{}) (bool, error)) {
	e.comparator = fn
}

func (e *Experiment) Clean(fn func(v interface{}) (interface{}, error)) {
	e.cleaner = fn
}

func (e *Experiment) Ignore(fn func(control, candidate interface{}) (bool, error)) {
	e.ignores = append(e.ignores, fn)
}

func (e *Experiment) RunIf(fn func() (bool, error)) {
	e.runcheck = fn
}

func (e *Experiment) BeforeRun(fn func() error) {
	e.beforeRun = fn
}

func (e *Experiment) Publish(fn func(Result) error) {
	e.publisher = fn
}

func (e *Experiment) ReportErrors(fn func(...ResultError)) {
	e.errorReporter = fn
}

func (e *Experiment) Run() (interface{}, error) {
	enabled, err := e.runcheck()
	if err != nil {
		enabled = true
		e.errorReporter(ResultError{"run_if", "experiment", -1, err})
		return nil, err
	}

	if enabled && len(e.behaviors) > 1 {
		r := Run(e)
		return r.Control.Value, r.Control.Err
	}

	behavior, ok := e.behaviors[controlBehavior]
	if !ok {
		return nil, behaviorNotFound(e, controlBehavior)
	}

	return runBehavior(e, controlBehavior, behavior)
}

func defaultComparator(candidate, control interface{}) (bool, error) {
	return reflect.DeepEqual(candidate, control), nil
}

func defaultRunCheck() (bool, error) {
	return true, nil
}

func defaultCleaner(v interface{}) (interface{}, error) {
	return v, nil
}

func defaultPublisher(r Result) error {
	return nil
}

func defaultErrorReporter(errors ...ResultError) {
}

func defaultBeforeRun() error {
	return nil
}
