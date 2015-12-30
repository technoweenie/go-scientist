package scientist

import "fmt"

func Bool(ok interface{}, err error) (bool, error) {
	if err != nil {
		return false, err
	}

	switch t := ok.(type) {
	case bool:
		return t, nil
	default:
		return false, fmt.Errorf("[scientist] bad result type: %v (%T)", ok, ok)
	}
}
