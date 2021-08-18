package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	GaugeQueries   GaugeQueries   `yaml:"gauge_queries"`
	CounterQueries CounterQueries `yaml:"counter_queries"`
}

type GaugeQueries struct {
	Queries []map[string]string `yaml:"gauge_queries,flow"`
}

type CounterQueries struct {
	Queries []map[string]string `yaml:"counter_queries,flow"`
}

func ReadConfigFile(path string) (*Config, error) {

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, err
	}

	//TODO aggiungere un metodo di validazione semantica del contenuto della configuration

	return &cfg, nil
}
