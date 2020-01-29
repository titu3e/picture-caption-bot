package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	Token     string   `yaml:"token"`
	Debug     bool     `yaml:"debug"`
	Workers   int      `yaml:"workers"`
	Whitelist []int64  `yaml:"whitelist"`
	Blacklist []int64  `yaml:"blacklist"`
	Font      string   `yam:"font"`
	Phrases   []string `yaml:"phrases"`
	Group     struct {
		Enabled               bool    `yaml:"enabled"`
		ActivationPhrase      string  `yaml:"activation_phrase"`
		ActivationProbability float64 `yaml:"activation_probability"`
	} `yaml:"group"`
}

func LoadConfig(configPath string) (*Config, error) {
	config := Config{}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
