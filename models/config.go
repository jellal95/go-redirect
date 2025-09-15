package models

type Config struct {
	Propeller Propeller `yaml:"propeller"`
	Galaksion Galaksion `yaml:"galaksion"`
	Popcash   Popcash   `yaml:"popcash"`
	Products  []Product `yaml:"products"`
}

type Propeller struct {
	Aid         string `yaml:"aid"`
	Tid         string `yaml:"tid"`
	PostbackURL string `yaml:"postback_url"`
}

type Galaksion struct {
	Cid         string `yaml:"cid"`
	PostbackURL string `yaml:"postback_url"`
}

type Popcash struct {
	Aid         string `yaml:"aid"`
	Type        string `yaml:"type"`
	PostbackURL string `yaml:"postback_url"`
}

type ClickAdilla struct {
	Token       string `yaml:"token"`
	PostbackURL string `yaml:"postback_url"`
}
