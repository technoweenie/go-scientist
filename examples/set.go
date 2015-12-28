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
	e := scientist.New("set")
	e.Use(func() (interface{}, error) {
		for _, i := range arr {
			if i == 9999 {
				return true, nil
			}
		}

		return false, nil
	})

	e.Try(func() (interface{}, error) {
		return set[9999], nil
	})

	e.Publish(publish)
	e.Run()
}

func publish(r scientist.Result) error {
	fmt.Println("Experiment:", r.Experiment.Name)
	publishObservation(r.Control)
	for _, o := range r.Candidates {
		publishObservation(o)
	}
	return nil
}

func publishObservation(o scientist.Observation) {
	fmt.Printf(" * %s\n", o.Name)
	fmt.Printf("   value: %v\n", o.Value)
	fmt.Printf("   err: %v\n", o.Err)
	fmt.Printf("   time: %v\n", o.Runtime)
}
