package jet

import (
	"strings"
	"unicode/utf8"
)

// templateError owns its diagnostic text; catch need not invoke a native hook.
type templateError struct {
	message string
	cause   error
}

func (e *templateError) Error() string { return e.message }
func (e *templateError) Unwrap() error { return e.cause }

const catchDiagnosticBytes = 512

func (st *Runtime) catchDiagnostic(recovered interface{}) string {
	message := "template execution failed"
	switch failure := recovered.(type) {
	case *templateError:
		message = failure.message
	case string:
		message = failure
	}
	n := len(message)
	suffix := ""
	if n > catchDiagnosticBytes {
		n = catchDiagnosticBytes - 3
		for n > 0 && !utf8.RuneStart(message[n]) {
			n--
		}
		suffix = "..."
	}
	st.checkpoint(uint64(n + len(suffix)))
	st.reserveValue(int64(n + len(suffix)))
	var result strings.Builder
	result.Grow(n + len(suffix))
	result.WriteString(message[:n])
	result.WriteString(suffix)
	return result.String()
}
