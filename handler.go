package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"sync/atomic"
)

// TODO: Make configurable.
const maxConcurrency = 8

type Job struct {
	Name       string
	Exporter   Exporter
	ErrorCount int64
}

func NewHandler(configs []ExporterConfig) http.HandlerFunc {
	jobsByName := map[string]*Job{}
	jobs := []*Job{}
	for _, config := range configs {
		if _, ok := jobsByName[config.Name]; ok {
			logger.Warningf("Exporter %q was added more than once; ignoring", config.Name)
			continue
		}

		logger.Infof("Adding exporter %q", config.Name)
		exporter := NewBaseExporter(config)

		job := &Job{
			Name:     config.Name,
			Exporter: exporter,
		}
		jobsByName[config.Name] = job
		jobs = append(jobs, job)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var exps []*Job
		if match := metricsSubPath.FindStringSubmatch(req.URL.Path); len(match) > 0 {
			name := match[1]
			if job, ok := jobsByName[name]; ok {
				exps = []*Job{job}
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

		type entry struct {
			job *Job
			buf *bytes.Buffer
			err error
		}

		var wg sync.WaitGroup
		resultCh := make(chan entry, len(exps))
		entryCh := make(chan entry, len(exps))

		concurrency := len(exps)
		if concurrency > maxConcurrency {
			concurrency = maxConcurrency
		}

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for entry := range entryCh {
					entry.err = runJob(entry.job, entry.buf)
					resultCh <- entry
				}
			}()
		}
		for _, job := range exps {
			entryCh <- entry{
				job: job,
				buf: new(bytes.Buffer),
			}
		}
		close(entryCh)
		wg.Wait()

		close(resultCh)
		for result := range resultCh {
			if result.err != nil {
				atomic.AddInt64(&result.job.ErrorCount, 1)
			} else {
				_, err := io.Copy(w, result.buf)
				if err != nil {
					logger.Warningf("Error emitting data: %s", err)
				}
			}
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		bw := bufio.NewWriter(w)
		bw.WriteString("# TYPE prompt_errors_total counter\n")
		for _, job := range jobs {
			bw.WriteString(fmt.Sprintf(`prompt_errors_total{exporter="%s"} %d`, job.Name, job.ErrorCount))
			bw.WriteRune('\n')
		}
		bw.Flush()
	})
}

func runJob(job *Job, w io.Writer) error {
	logger.Debugf("Running exporter %q", job.Name)

	err := job.Exporter.Exec(w)
	if err != nil {
		logger.Errorf("Exporter %q failed to run: %s", job.Name, err)
	}
	return err
}

var metricsSubPath = regexp.MustCompile("^/metrics/(.+)$")
