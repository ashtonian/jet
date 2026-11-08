package jet

import (
	"bytes"
	"io"
)

// A transaction owns only its temporary output. The render keeps the same
// execution state through nested buffering, rollback, and commit.
type outputTransaction struct {
	parent io.Writer
	buffer bytes.Buffer
}

type executionWriter struct {
	runtime  *Runtime
	writer   io.Writer
	retained bool
}

func (st *Runtime) outputWriter(w io.Writer, retained bool) io.Writer {
	if st.execution.ctx == nil {
		return &checkedWriter{writer: w, failure: &st.writeErr}
	}
	return &executionWriter{runtime: st, writer: w, retained: retained}
}

func (w *executionWriter) Write(b []byte) (int, error) {
	return w.write(b, true)
}

func (w *executionWriter) write(b []byte, generated bool) (int, error) {
	st := w.runtime
	st.checkpoint(uint64(len(b)))
	if generated {
		st.reserve(&st.execution.output, uint64(st.execution.options.MaxOutputBytes), uint64(len(b)), "output", ErrOutputLimit)
	}
	if w.retained {
		buffer := w.writer.(*bytes.Buffer)
		size, ok := addValueSizes(int64(buffer.Len()), int64(len(b)))
		if !ok || uint64(size) > uint64(^uint(0)>>1) {
			st.failValueSize()
		}
		st.reserveValue(int64(len(b)))
	}
	sink := checkedWriter{writer: w.writer, failure: &st.writeErr}
	return sink.Write(b)
}

func (st *Runtime) beginOutputTransaction() *outputTransaction {
	st.checkpoint(0)
	tx := &outputTransaction{parent: st.Writer}
	st.Writer = st.outputWriter(&tx.buffer, true)
	return tx
}

func (st *Runtime) commitOutputTransaction(tx *outputTransaction) {
	st.checkpoint(0)
	st.Writer = tx.parent
	var err error
	if parent, ok := tx.parent.(*executionWriter); ok && parent.runtime == st {
		_, err = parent.write(tx.buffer.Bytes(), false)
	} else {
		_, err = tx.parent.Write(tx.buffer.Bytes())
	}
	if err != nil {
		st.terminate(err)
	}
	st.checkpoint(0)
}
