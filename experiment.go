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
	}
}

type behaviorFunc func() (value interface{}, err error)
type valueFunc func(control, candidate interface{}) (bool, error)
type checkFunc func() (bool, error)
type resultFunc func(Result) error
type resultErrorFunc func(...ResultError)
type callbackFunc func() error

type Experiment struct {
	Name          string
	Context       map[string]string
	behaviors     map[string]behaviorFunc
	ignores       []valueFunc
	comparator    valueFunc
	runcheck      checkFunc
	publisher     resultFunc
	errorReporter resultErrorFunc
	beforeRun     callbackFunc
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

func (e *Experiment) BeforeRun(fn func() error) {
	e.beforeRun = callbackFunc(fn)
}

func (e *Experiment) Enabled() (bool, error) {
	return e.runcheck()
}

func (e *Experiment) Run() (interface{}, error) {
	enabled, err := e.Enabled()
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

	return behavior()
}

func (e *Experiment) Publish(fn func(Result) error) {
	e.publisher = resultFunc(fn)
}

func (e *Experiment) ReportErrors(fn func(...ResultError)) {
	e.errorReporter = resultErrorFunc(fn)
}

func defaultComparator(candidate, control interface{}) (bool, error) {
	return reflect.DeepEqual(candidate, control), nil
}

func defaultRunCheck() (bool, error) {
	return true, nil
}

func defaultPublisher(r Result) error {
	return nil
}

func defaultErrorReporter(errors ...ResultError) {
}

func defaultBeforeRun() error {
	return nil
}
