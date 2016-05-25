package main

import (
	"io/ioutil"

	"github.com/pkg/errors"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Exporters []ExporterConfig
}

func (c *Config) ReadFromFile(fileName string) error {
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return errors.Wrapf(err, "Unable to read config file %q", fileName)
	}

	var o struct {
		Exporters []ExporterConfig `yaml:"exporters"`
	}
	err = yaml.Unmarshal(b, &o)
	if err != nil {
		return errors.Wrapf(err, "Unable to parse config file %q", fileName)
	}

	c.Exporters = append(c.Exporters, o.Exporters...)
	return nil
}

type ExporterConfig struct {
	Name        string   `yaml:"name"`
	Command     string   `yaml:"command"`
	MinInterval *float64 `yaml:"minInterval"`

	// TODO: For later:
	// Namespace string `yaml:"namespace"`
	// Subsystem string `yaml:"subsystem"`
}
