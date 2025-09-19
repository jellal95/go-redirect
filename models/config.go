package models

type Config struct {
	Propeller Propeller `yaml:"propeller"`
	Galaksion Galaksion `yaml:"galaksion"`
	Popcash   Popcash   `yaml:"popcash"`
	BotFilter BotFilter `yaml:"bot_filter"`
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

type BotFilter struct {
	AllowCountries     []string `yaml:"allow_countries"`
	AllowMobileOnly    bool     `yaml:"allow_mobile_only"`
	RateLimitMax       int      `yaml:"rate_limit_max"`
	RateLimitWindowSec int      `yaml:"rate_limit_window_sec"`
	LogAllowed         bool     `yaml:"log_allowed"`
	LogBlocked         bool     `yaml:"log_blocked"`
	BlacklistUA        []string `yaml:"blacklist_ua"`
	BlacklistIPPrefix  []string `yaml:"blacklist_ip_prefix"`
	BlacklistReferrer  []string `yaml:"blacklist_referrer"`
	BlacklistRefRegex  []string `yaml:"blacklist_ref_regex"`
}
