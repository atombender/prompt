package main

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"time"
)

type SubprocessExporter struct {
	cmd       *exec.Cmd
	buf       *bytes.Buffer
	stderrBuf *bytes.Buffer
	err       error
}

func NewSubprocessExporter(command string) *SubprocessExporter {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdin = nil
	return &SubprocessExporter{
		cmd:       cmd,
		buf:       new(bytes.Buffer),
		stderrBuf: new(bytes.Buffer),
	}
}

func (exp *SubprocessExporter) Exec(w io.Writer) error {
	stdout, err := exp.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	go func() {
		_, err := io.Copy(exp.buf, stdout)
		if err != nil {
			exp.err = err
		}
	}()

	stderr, err := exp.cmd.StderrPipe()
	if err != nil {
		return err
	}

	go func() {
		_, err := io.Copy(exp.stderrBuf, stderr)
		if err != nil && exp.err == nil {
			exp.err = err
		} else if exp.stderrBuf.Len() > 0 {
			logger.Errorf("[stderr from %q] %s", exp.cmd.Path, exp.stderrBuf.String())
		}
	}()

	startTime := time.Now()

	if err := exp.cmd.Start(); err != nil {
		return err
	}

	_ = exp.cmd.Wait()

	elapsedTime := time.Since(startTime)

	if exp.cmd.ProcessState.Success() {
		logger.Debugf("Process finished successfully in %s", elapsedTime.String())
		if _, err := io.Copy(w, exp.buf); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Process finished with non-zero exit code: %s",
			exp.cmd.ProcessState.String())
	}

	if exp.err != nil {
		return exp.err
	}

	_, err = w.Write([]byte{'\n'})
	return err
}
