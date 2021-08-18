package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Project string  `yaml:"project"`
	Gauge   []Query `yaml:"gauge-queries"`
	Counter []Query `yaml:"counter-queries"`
}

type Query struct {
	Query string `yaml:"query"`
	File  string `yaml:"file"`
}

func (cfg *Config) GetGaugeFiles() []string {

	var paths []string
	for _, path := range cfg.Gauge {

		paths = append(paths, path.File)
	}
	return paths
}

func (cfg *Config) GetCounterFiles() []string {

	var paths []string
	for _, path := range cfg.Counter {

		paths = append(paths, path.File)
	}
	return paths
}

func ReadConfigFile(path string) (*Config, error) {

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("something wrong during file opening: %s", err.Error())
	}

	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("something wrong during configuration unmarshalling: %s", err.Error())
	}

	err = validate(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validate(cfg *Config) error {

	if len(cfg.Counter) == 0 {
		return fmt.Errorf("no Counter parameters available")
	}

	if len(cfg.Gauge) == 0 {
		return fmt.Errorf("no Gauge parameters available")
	}
	if strings.TrimSpace(cfg.Project) == "" {
		return fmt.Errorf("no Project parameter available")
	}
	return nil
}
