package jet

import "strconv"

func (st *Runtime) reserveNumericInput(size int) {
	if st.execution.ctx != nil {
		st.reserveSlots(size, 2)
	}
}

type numericExecutionError struct{ cause *strconv.NumError }

func (e *numericExecutionError) Error() string {
	return "strconv." + e.cause.Func + ": " + e.cause.Err.Error()
}

func (e *numericExecutionError) Unwrap() error { return e.cause }

func (st *Runtime) numericError(err error) error {
	if st.execution.ctx != nil {
		if cause, ok := err.(*strconv.NumError); ok {
			return &numericExecutionError{cause: cause}
		}
	}
	return err
}
