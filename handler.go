package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
)

func NewHandler(configs []ExporterConfig) http.HandlerFunc {
	exportersByName := map[string]Exporter{}
	exporters := []Exporter{}
	for _, config := range configs {
		if _, ok := exportersByName[config.Name]; ok {
			logger.Warningf("Exporter %q was added more than once; ignoring", config.Name)
			continue
		}

		logger.Infof("Adding exporter %q", config.Name)
		exporter := NewBaseExporter(config)

		exportersByName[config.Name] = exporter
		exporters = append(exporters, exporter)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var exps []Exporter
		if match := metricsSubPath.FindStringSubmatch(req.URL.Path); len(match) > 0 {
			name := match[1]
			if exporter, ok := exportersByName[name]; ok {
				exps = []Exporter{exporter}
			} else {
				logger.Errorf("No such exporter %q", name)
				w.WriteHeader(404)
				w.Write([]byte(fmt.Sprintf("No such exporter %q", name)))
				return
			}
		} else {
			exps = exporters
		}
		if len(exps) == 0 {
			w.WriteHeader(204)
			return
		}

		var wg sync.WaitGroup
		buffers := make([]*bytes.Buffer, len(exps))
		for idx, exporter := range exporters {
			wg.Add(1)
			buffers[idx] = new(bytes.Buffer)
			go func(exporter Exporter, buf *bytes.Buffer) {
				defer wg.Done()
				runExporter(exporter, buf)
			}(exporter, buffers[idx])
		}
		wg.Wait()

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		for _, buf := range buffers {
			_, err := io.Copy(w, buf)
			if err != nil {
				logger.Errorf("Error emitting data: %s", err)
			}
		}
	})
}

func runExporter(exporter Exporter, w io.Writer) {
	logger.Debugf("Running exporter %#v", exporter)

	err := exporter.Exec(w)
	if err != nil {
		logger.Errorf("Exporter failed to run: %s", err)
	}
}

var metricsSubPath = regexp.MustCompile("^/metrics/(.+)$")
