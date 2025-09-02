package utils

import (
	"encoding/json"
	"io/ioutil"

	"go-redirect/models"
)

func LoadProducts(path string) ([]models.Product, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var products []models.Product
	if err := json.Unmarshal(data, &products); err != nil {
		return nil, err
	}
	return products, nil
}
