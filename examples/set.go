package main

import (
	"context"
	"fmt"
	"time"

	"github.com/technoweenie/go-scientist"
)

var (
	arr = make([]int, 10000)
	set = make(map[int]bool, 10000)
)

func init() {
	for i := 0; i < 10000; i++ {
		arr[i] = i
		set[i] = true
	}
}

func main() {
	n := 9999
	controlFn := func(ctx context.Context) (interface{}, error) {
		for _, i := range arr {
			if i == n {
				return true, nil
			}
		}

		return false, nil
	}
	candidateFn := func(ctx context.Context) (interface{}, error) {
		return set[n], nil
	}

	Run(controlFn, candidateFn)
	RunAsync(controlFn, candidateFn)
	RunAsyncCandidatesOnly(controlFn, candidateFn)
	// Wait for sometime
	time.Sleep(time.Second * 10)
}

func Run(controlFn func(ctx context.Context) (interface{}, error), candidateFn func(ctx context.Context) (interface{}, error)) {
	start := time.Now()
	defer fmt.Printf("Run experiment time elapsed: %s\n", time.Since(start))

	e := scientist.New("synchronous")
	e.Use(controlFn)
	e.Try(candidateFn)

	e.Context["control"] = "array"
	e.Context["candidate"] = "map"
	e.Context["run_type"] = "sync"

	e.Publish(publish)
	result, err := e.Run(context.Background())
	if err != nil {
		fmt.Printf("experiment error: %q\n", err)
		return
	}
	fmt.Printf("The arbitrary example returned: %v\n", result)
}

func RunAsync(controlFn func(ctx context.Context) (interface{}, error), candidateFn func(ctx context.Context) (interface{}, error)) {
	start := time.Now()
	defer fmt.Printf("RunAsync experiment time elapsed: %s\n", time.Since(start))

	e1 := scientist.New("asynchronous")
	e1.Use(controlFn)
	e1.Try(candidateFn)

	e1.Context["control"] = "array"
	e1.Context["candidate"] = "map"
	e1.Context["run_type"] = "async"

	e1.Publish(publish)
	result, err := e1.RunAsync(context.Background())
	if err != nil {
		fmt.Printf("experiment error: %q\n", err)
		return
	}
	fmt.Printf("The arbitrary example returned: %v\n", result)
}

func RunAsyncCandidatesOnly(controlFn func(ctx context.Context) (interface{}, error), candidateFn func(ctx context.Context) (interface{}, error)) {
	start := time.Now()
	defer fmt.Printf("RunAsyncCandidatesOnly experiment time elapsed: %s\n", time.Since(start))

	e2 := scientist.New("asynchronousCandidatesOnly")
	e2.Use(controlFn)
	e2.Try(candidateFn)

	e2.Context["control"] = "array"
	e2.Context["candidate"] = "map"
	e2.Context["run_type"] = "asyncCandidatesOnly"

	e2.Publish(publish)
	result, err := e2.RunAsyncCandidatesOnly(context.Background())
	if err != nil {
		fmt.Printf("experiment error: %q\n", err)
		return
	}
	fmt.Printf("The arbitrary example returned: %v\n", result)
}

func publish(r scientist.Result) error {
	fmt.Println("Experiment:", r.Experiment.Name)
	publishObservation(r.Control)
	for _, o := range r.Candidates {
		publishObservation(o)
	}
	fmt.Println(" * Context:")
	for key, value := range r.Experiment.Context {
		fmt.Printf("   %q: %q\n", key, value)
	}
	return nil
}

func publishObservation(o *scientist.Observation) {
	fmt.Println("publishObservation", o)
	fmt.Printf(" * %s\n", o.Name)
	fmt.Printf("   value: %v\n", o.Value)
	fmt.Printf("   err: %v\n", o.Err)
	fmt.Printf("   time: %v\n", o.Runtime)
}
