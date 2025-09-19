package utils

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

func InitLogger() {
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	log.SetLevel(log.InfoLevel)
}

type LogEntry struct {
	Type        string      `json:"type"`
	Timestamp   time.Time   `json:"timestamp"`
	ProductName string      `json:"product_name,omitempty"`
	URL         string      `json:"url,omitempty"`
	IP          string      `json:"ip,omitempty"`
	UserAgent   string      `json:"user_agent,omitempty"`
	Browser     string      `json:"browser,omitempty"`
	OS          string      `json:"os,omitempty"`
	Device      string      `json:"device,omitempty"`
	Referer     string      `json:"referer,omitempty"`
	QueryRaw    string      `json:"query_raw,omitempty"`
	QueryParams interface{} `json:"query_params,omitempty"`
	Headers     interface{} `json:"headers,omitempty"`
	Extra       interface{} `json:"extra,omitempty"`
}

func LogInfo(entry LogEntry) {
	entry.Timestamp = time.Now()
	log.WithFields(log.Fields{
		"type":         entry.Type,
		"product_name": entry.ProductName,
		"url":          entry.URL,
		"ip":           entry.IP,
		"user_agent":   entry.UserAgent,
		"browser":      entry.Browser,
		"os":           entry.OS,
		"device":       entry.Device,
		"referer":      entry.Referer,
		"query_params": entry.QueryParams,
		"headers":      entry.Headers,
		"extra":        entry.Extra,
	}).Info("event logged")
}

func LogFatal(entry LogEntry, exitCode int) {
	entry.Timestamp = time.Now()
	log.WithFields(log.Fields{
		"type":         entry.Type,
		"product_name": entry.ProductName,
		"url":          entry.URL,
		"ip":           entry.IP,
		"user_agent":   entry.UserAgent,
		"browser":      entry.Browser,
		"os":           entry.OS,
		"device":       entry.Device,
		"referer":      entry.Referer,
		"query_params": entry.QueryParams,
		"headers":      entry.Headers,
		"extra":        entry.Extra,
	}).Fatal("fatal event logged")
	os.Exit(exitCode)
}
