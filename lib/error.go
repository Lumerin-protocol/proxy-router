package lib

import "fmt"

type customError struct {
	err      error
	childErr error
}

func (e *customError) Error() string {
	return fmt.Sprintf("%s:%s", e.err.Error(), e.childErr.Error())
}

func (e *customError) Unwrap() error {
	return e.childErr
}

func (e *customError) Is(target error) bool {
	return target == e.err
}

func WrapError(parent error, child error) error {
	return &customError{
		err:      parent,
		childErr: child,
	}
}
