package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
)

type Job struct {
	Name       string
	Exporter   Exporter
	ErrorCount int64
}

func NewHandler(configs []ExporterConfig) http.HandlerFunc {
	jobsByName := map[string]Job{}
	jobs := []Job{}
	for _, config := range configs {
		if _, ok := jobsByName[config.Name]; ok {
			logger.Warningf("Exporter %q was added more than once; ignoring", config.Name)
			continue
		}

		logger.Infof("Adding exporter %q", config.Name)
		exporter := NewBaseExporter(config)

		job := Job{
			Name:     config.Name,
			Exporter: exporter,
		}
		jobsByName[config.Name] = job
		jobs = append(jobs, job)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var exps []Job
		if match := metricsSubPath.FindStringSubmatch(req.URL.Path); len(match) > 0 {
			name := match[1]
			if job, ok := jobsByName[name]; ok {
				exps = []Job{job}
			} else {
				logger.Errorf("No such exporter %q", name)
				w.WriteHeader(404)
				w.Write([]byte(fmt.Sprintf("No such exporter %q", name)))
				return
			}
		} else {
			exps = jobs
		}
		if len(exps) == 0 {
			w.WriteHeader(204)
			return
		}

		var wg sync.WaitGroup
		buffers := make([]*bytes.Buffer, len(exps))
		for idx, job := range jobs {
			wg.Add(1)
			buffers[idx] = new(bytes.Buffer)
			go func(job Job, buf *bytes.Buffer) {
				defer wg.Done()
				runJob(job, buf)
			}(job, buffers[idx])
		}
		wg.Wait()

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		for _, buf := range buffers {
			_, err := io.Copy(w, buf)
			if err != nil {
				logger.Errorf("Error emitting data: %s", err)
			}
		}

		bw := bufio.NewWriter(w)
		bw.WriteString("# TYPE prompt_errors_total counter\n")
		for _, job := range jobs {
			bw.WriteString(fmt.Sprintf(`prompt_errors_total{exporter="%s"} %d`, job.Name, job.ErrorCount))
			bw.WriteRune('\n')
		}
		bw.Flush()
	})
}

func runJob(job Job, w io.Writer) {
	logger.Debugf("Running exporter %q", job.Name)

	err := job.Exporter.Exec(w)
	if err != nil {
		logger.Errorf("Exporter %q failed to run: %s", job.Name, err)
	}
}

var metricsSubPath = regexp.MustCompile("^/metrics/(.+)$")
