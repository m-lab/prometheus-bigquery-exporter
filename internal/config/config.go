package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/m-lab/go/logx"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

var fs = afero.NewOsFs()

type Config struct {
	Name    string      `yaml:"-"`
	stat    os.FileInfo `yaml:"-"`
	Project string      `yaml:"project"`
	Gauge   []Query     `yaml:"gauge-queries"`
	Counter []Query     `yaml:"counter-queries"`
	mu      sync.Mutex  `yaml:"-"`
}

func (cfg *Config) CheckModified() bool {
	defer cfg.mu.Unlock()
	cfg.mu.Lock()
	res, err := cfg.IsModified()
	if err != nil {
		fmt.Printf("Something wrong: %s", err.Error())
	}
	return res
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

func (cfg *Config) IsModified() (bool, error) {

	var err error
	if cfg.stat == nil {
		cfg.stat, err = fs.Stat(cfg.Name)
		logx.Debug.Println("IsModified:stat1:", cfg.Name, err)
		// Return true on the first successful Stat(), or the error otherwise.
		return err == nil, err
	}
	curr, err := fs.Stat(cfg.Name)
	if err != nil {
		log.Printf("Failed to stat %q: %v", cfg.Name, err)
		return false, err
	}
	logx.Debug.Println("IsModified:stat2:", cfg.Name, curr.ModTime(), cfg.stat.ModTime(),
		curr.ModTime().After(cfg.stat.ModTime()))
	modified := curr.ModTime().After(cfg.stat.ModTime())
	if modified {
		// Update the stat cache to the latest version.
		cfg.stat = curr
	}
	return modified, nil
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

	cfg.Name = path
	cfg.stat, err = fs.Stat(cfg.Name)
	if err != nil {
		return nil, fmt.Errorf("something wrong during file stat extraction: %s", err.Error())
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
