package models

type Config struct {
	PropAds   PropAds   `yaml:"propellerAds"`
	Galaksion Galaksion `yaml:"galaksion"`
	Products  []Product `yaml:"products"`
}

type PropAds struct {
	Aid         string `yaml:"aid"`
	Tid         string `yaml:"tid"`
	PostbackURL string `yaml:"postback_url"`
}

type Galaksion struct {
	Cid         string `yaml:"cid"`
	PostbackURL string `yaml:"postback_url"`
}
