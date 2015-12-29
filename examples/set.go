package main

import (
	"fmt"

	scientist ".."
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
	includes(9999)
}

func includes(n int) (bool, error) {
	e := scientist.New("set")
	e.Use(func() (interface{}, error) {
		for _, i := range arr {
			if i == n {
				return true, nil
			}
		}

		return false, nil
	})

	e.Try(func() (interface{}, error) {
		return set[n], nil
	})

	e.Context["control"] = "array"
	e.Context["candidate"] = "map"

	e.Publish(publish)

	ok, err := e.Run()
	if err != nil {
		return false, err
	}

	switch t := ok.(type) {
	case bool:
		return t, nil
	default:
		return false, fmt.Errorf("bad type: %v", ok)
	}
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
	fmt.Printf(" * %s\n", o.Name)
	fmt.Printf("   value: %v\n", o.Value)
	fmt.Printf("   err: %v\n", o.Err)
	fmt.Printf("   time: %v\n", o.Runtime)
}
