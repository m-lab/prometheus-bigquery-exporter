package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Gauge   []Query `yaml:"gauge_queries"`
	Counter []Query `yaml:"counter_queries"`
}

type Query struct {
	Query string `yaml:"query"`
	File  string `yaml:"file"`
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
