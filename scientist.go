package scientist

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	controlBehavior   = "control"
	candidateBehavior = "candidate"
)

type Observation struct {
	Experiment *Experiment
	Name       string
	Started    time.Time
	Runtime    time.Duration
	Value      interface{}
	Err        error
}

func (o *Observation) CleanedValue() (interface{}, error) {
	return o.Experiment.cleaner(o.Value)
}

type Result struct {
	Experiment   *Experiment
	Control      *Observation
	Observations []*Observation
	Candidates   []*Observation
	Ignored      []*Observation
	Mismatched   []*Observation
	Errors       []ResultError
}

func (r Result) IsMatched() bool {
	if r.IsMismatched() || r.IsIgnored() {
		return false
	}
	return true
}

func (r Result) IsMismatched() bool {
	return len(r.Mismatched) > 0
}

func (r Result) IsIgnored() bool {
	return len(r.Ignored) > 0
}

func RunAsync(ctx context.Context, e *Experiment, controlBehavior string) Result {
	var errors []ResultError
	if err := e.beforeRun(); err != nil {
		errors = append(errors, e.resultErr("before_run", err))
	}

	behaviors := e.Shuffle(controlBehavior, false)
	control, candidates := runAsync(ctx, e, behaviors, controlBehavior)
	r := gatherResult(e, control, candidates, controlBehavior)
	r.Errors = append(r.Errors, errors...)
	if err := e.publisher(r); err != nil {
		r.Errors = append(r.Errors, e.resultErr("publish", err))
	}

	if len(r.Errors) > 0 {
		e.errorReporter(r.Errors...)
	}

	return r
}
func RunAsyncCandidatesOnly(ctx context.Context, e *Experiment, controlBehavior string) Result {
	r := Result{Experiment: e}
	var errors []ResultError
	if err := e.beforeRun(); err != nil {
		errors = append(r.Errors, e.resultErr("before_run", err))
	}
	r.Control = observe(ctx, e, controlBehavior, e.behaviors[controlBehavior])
	r.Errors = errors

	go func(ctx context.Context, e *Experiment) {
		behaviors := e.Shuffle(controlBehavior, true)
		_, candidates := runAsync(ctx, e, behaviors, controlBehavior)
		r := gatherResult(e, r.Control, candidates, controlBehavior)
		r.Errors = append(r.Errors, errors...)
		if err := e.publisher(r); err != nil {
			r.Errors = append(r.Errors, e.resultErr("publish", err))
		}

		if len(r.Errors) > 0 {
			e.errorReporter(r.Errors...)
		}
	}(ctx, e)

	return r
}

func Run(ctx context.Context, e *Experiment, controlBehavior string) Result {
	r := Result{Experiment: e}
	if err := e.beforeRun(); err != nil {
		r.Errors = append(r.Errors, e.resultErr("before_run", err))
	}

	numCandidates := len(e.behaviors) - 1
	r.Control = observe(ctx, e, controlBehavior, e.behaviors[controlBehavior])
	r.Candidates = make([]*Observation, numCandidates)
	r.Ignored = make([]*Observation, 0, numCandidates)
	r.Mismatched = make([]*Observation, 0, numCandidates)
	r.Observations = make([]*Observation, numCandidates+1)
	r.Observations[0] = r.Control

	i := 0
	for bname, b := range e.behaviors {
		if bname == controlBehavior {
			continue
		}

		c := observe(ctx, e, bname, b)
		r.Candidates[i] = c
		i += 1
		r.Observations[i] = c

		ok, err := matching(e, r.Control, c)
		if err != nil {
			ok = false
			r.Errors = append(r.Errors, e.resultErr("compare", err))
		}

		if ok {
			continue
		}

		ignored, err := ignoring(e, r.Control, c)
		if err != nil {
			ignored = false
			r.Errors = append(r.Errors, e.resultErr("ignore", err))
		}

		if ignored {
			r.Ignored = append(r.Ignored, c)
		} else {
			r.Mismatched = append(r.Mismatched, c)
		}
	}

	if err := e.publisher(r); err != nil {
		r.Errors = append(r.Errors, e.resultErr("publish", err))
	}

	if len(r.Errors) > 0 {
		e.errorReporter(r.Errors...)
	}

	return r
}

func matching(e *Experiment, control, candidate *Observation) (bool, error) {
	// neither returned errors
	if control.Err == nil && candidate.Err == nil {
		return e.comparator(control.Value, candidate.Value)
	}

	// both returned errors
	if control.Err != nil && candidate.Err != nil {
		return control.Err.Error() == candidate.Err.Error(), nil
	}

	// returned different errors
	return false, nil
}

func ignoring(e *Experiment, control, candidate *Observation) (bool, error) {
	for _, i := range e.ignores {
		ok, err := i(control.Value, candidate.Value)
		if err != nil {
			return false, err
		}

		if ok {
			return true, nil
		}
	}

	return false, nil
}

func behaviorNotFound(e *Experiment, name string) error {
	return fmt.Errorf("Behavior %q not found for experiment %q", name, e.Name)
}

type ResultError struct {
	Operation  string
	Experiment string
	Err        error
}

func (e ResultError) Error() string {
	return e.Err.Error()
}

type MismatchError struct {
	Result Result
}

func (e MismatchError) Error() string {
	return fmt.Sprintf("[scientist] experiment %q observations mismatched", e.Result.Experiment.Name)
}

func runAsync(ctx context.Context, e *Experiment, behaviors []string, controlBehavior string) (*Observation, []*Observation) {
	var (
		control    *Observation
		candidates []*Observation
		wg         sync.WaitGroup
	)

	finished := make(chan *Observation, len(behaviors))

	for _, name := range behaviors {
		wg.Add(1)
		go func(ctx context.Context, name string) {
			defer wg.Done()
			finished <- observe(ctx, e, name, e.behaviors[name])
		}(ctx, name)
	}
	wg.Wait()
	close(finished)

	for o := range finished {
		if o.Name == controlBehavior {
			control = o
		} else {
			candidates = append(candidates, o)
		}
	}

	return control, candidates
}

func observe(ctx context.Context, e *Experiment, name string, b behaviorFunc) *Observation {
	o := &Observation{
		Experiment: e,
		Name:       name,
		Started:    time.Now(),
	}

	defer func() {
		if r := recover(); r != nil {
			o.Err = errors.New(fmt.Sprintf("recover from bad behavior %s: %v", name, r))
		}
	}()

	if b == nil {
		b = e.behaviors[name]
	}

	if b == nil {
		o.Runtime = time.Since(o.Started)
		o.Err = behaviorNotFound(e, name)
	} else {
		v, err := b(ctx)
		o.Runtime = time.Since(o.Started)
		o.Value = v
		o.Err = err
	}

	return o
}

func gatherResult(e *Experiment, control *Observation, candidates []*Observation, controlName string) Result {
	r := Result{
		Experiment:   e,
		Control:      control,
		Candidates:   candidates,
		Observations: make([]*Observation, len(candidates)+1),
		Errors:       nil,
		Ignored:      nil,
		Mismatched:   nil,
	}
	r.Observations = append(r.Observations, control)
	r.Observations = append(r.Observations, candidates...)

	for _, candidate := range candidates {
		ok, err := matching(e, r.Control, candidate)
		if err != nil {
			ok = false
			r.Errors = append(r.Errors, e.resultErr("compare", err))
		}
		if ok {
			continue
		}

		ignored, err := ignoring(e, r.Control, candidate)
		if err != nil {
			ignored = false
			r.Errors = append(r.Errors, e.resultErr("ignore", err))
		}

		if ignored {
			r.Ignored = append(r.Ignored, candidate)
		} else {
			r.Mismatched = append(r.Mismatched, candidate)
		}
	}

	return r
}

// Shuffle randomizes the behavior access.
func (e *Experiment) Shuffle(behaviourName string, skip bool) []string {
	var behaviors []string
	for name := range e.behaviors {
		if skip && (behaviourName == name) {
			continue
		}
		behaviors = append(behaviors, name)
	}

	t := time.Now()
	rand.Seed(int64(t.Nanosecond()))

	arr := behaviors
	for i := len(arr) - 1; i > 0; i-- {
		j := rand.Intn(i)
		arr[i], arr[j] = arr[j], arr[i]
	}
	return arr
}
