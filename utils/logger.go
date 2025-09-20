package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

var wibLocation *time.Location

func init() {
	// Load WIB timezone (UTC+7)
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		// Fallback to fixed UTC+7 if timezone data not available
		loc = time.FixedZone("WIB", 7*60*60)
	}
	wibLocation = loc
}

var Logs []LogEntry
var logMu sync.Mutex

type LogEntry struct {
	Type        string                 `json:"type"`
	Timestamp   time.Time              `json:"timestamp"`
	ProductName string                 `json:"product_name,omitempty"`
	URL         string                 `json:"url,omitempty"`
	IP          string                 `json:"ip,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	Browser     string                 `json:"browser,omitempty"`
	OS          string                 `json:"os,omitempty"`
	Device      string                 `json:"device,omitempty"`
	Referer     string                 `json:"referer,omitempty"`
	QueryParams map[string]string      `json:"query_params,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

func LogInfo(entry LogEntry) error {
	logMu.Lock()
	defer logMu.Unlock()

	// Set WIB timestamp if not already set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().In(wibLocation)
	}

	// --- append memory ---
	Logs = append(Logs, entry)

	// --- append file ---
	dateStr := time.Now().In(wibLocation).Format("2006-01-02")
	// Use /logs for production (Fly.io volume), ./logs for development
	folder := os.Getenv("LOG_PATH")
	if folder == "" {
		folder = "logs" // default for development
	}
	os.MkdirAll(folder, 0755)
	filename := fmt.Sprintf("%s/log-%s.jsonl", folder, dateStr)

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	b, _ := json.Marshal(entry)
	_, err = f.Write(append(b, '\n'))
	return err
}

func LogsSummary() map[string]interface{} {
	logMu.Lock()
	defer logMu.Unlock()

	typeCount := map[string]int{}
	deviceCount := map[string]int{}
	productCount := map[string]int{}

	// dari memory
	for _, entry := range Logs {
		typeCount[entry.Type]++
		if entry.Device != "" {
			deviceCount[entry.Device]++
		}
		if entry.ProductName != "" {
			productCount[entry.ProductName]++
		}
	}

	// --- optional baca file log hari ini ---
	dateStr := time.Now().Format("2006-01-02")
	folder := os.Getenv("LOG_PATH")
	if folder == "" {
		folder = "logs" // default for development
	}
	filename := fmt.Sprintf("%s/log-%s.jsonl", folder, dateStr)
	file, err := os.Open(filename)
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var entry LogEntry
			json.Unmarshal(scanner.Bytes(), &entry)
			typeCount[entry.Type]++
			if entry.Device != "" {
				deviceCount[entry.Device]++
			}
			if entry.ProductName != "" {
				productCount[entry.ProductName]++
			}
		}
	}

	return map[string]interface{}{
		"total_logs":    len(Logs),
		"type_count":    typeCount,
		"device_count":  deviceCount,
		"product_count": productCount,
	}
}

func LogFatal(entry LogEntry, code int) {
	_ = LogInfo(entry)
	os.Exit(code)
}
