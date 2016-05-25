package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jessevdk/go-flags"
)

type options struct {
	Listen      string   `short:"l" long:"listen" required:"true" description:"Listen address." value-name:"HOST[:PORT]"`
	ConfigFiles []string `short:"c" long:"config" required:"true" description:"Configuration file name. If a directory, reads all files ending with .conf. May be specified multiple times." value-name:"FILE|DIR"`
}

func (o options) GetConfig() (*Config, error) {
	var config Config
	for _, fileName := range o.ConfigFiles {
		info, err := os.Stat(fileName)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			dirFiles, err := filepath.Glob(filepath.Join(fileName, "*.conf"))
			if err != nil {
				return nil, err
			}
			for _, dirFileName := range dirFiles {
				logger.Debugf("Loading config from %s", dirFileName)
				if err := config.ReadFromFile(dirFileName); err != nil {
					return nil, err
				}
			}
		} else {
			logger.Debugf("Loading config from %s", fileName)
			if err := config.ReadFromFile(fileName); err != nil {
				return nil, err
			}
		}
	}
	return &config, nil
}

func main() {
	var opts options
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}

	config, err := opts.GetConfig()
	if err != nil {
		fmt.Fprintln(os.Stdout, err)
		os.Exit(1)
	}

	logger.Infof("Listening on %s", opts.Listen)

	handler := NewHandler(config.Exporters)
	http.Handle("/metrics", handler)
	http.Handle("/metrics/", handler)
	http.ListenAndServe(opts.Listen, nil)
}
