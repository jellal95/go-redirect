package utils

import (
	"io/ioutil"

	"go-redirect/models"

	"gopkg.in/yaml.v3"
)

func LoadConfig(path string) (*models.Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg models.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
