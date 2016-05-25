package main

import (
	"bytes"
	"io"
	"sync"
	"time"
)

type Exporter interface {
	Exec(w io.Writer) error
}

type BaseExporter struct {
	ExporterConfig

	lastRun *time.Time
	cached  []byte

	sync.RWMutex
}

func NewBaseExporter(config ExporterConfig) *BaseExporter {
	return &BaseExporter{
		ExporterConfig: config,
	}
}

func (exp *BaseExporter) Exec(w io.Writer) error {
	exp.RLock()
	if exp.MinInterval != nil && exp.lastRun != nil && time.Since(*exp.lastRun).Seconds() < *exp.MinInterval {
		logger.Debugf("Min interval not yet met; returning cached copy")
		c, err := io.Copy(w, bytes.NewBuffer(exp.cached))
		logger.Debugf("c=%#v", c)
		exp.RUnlock()
		return err
	}
	exp.RUnlock()

	buf := new(bytes.Buffer)
	subprocess := NewSubprocessExporter(exp.Command)
	if err := subprocess.Exec(buf); err != nil {
		return err
	}

	exp.Lock()
	exp.cached = buf.Bytes()
	_, err := io.Copy(w, buf)
	t := time.Now()
	exp.lastRun = &t
	exp.Unlock()
	return err
}
