// main.go
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type config struct {
	dbPath     string
	logDir     string
	timeWindow time.Duration
}

type experimentInfo struct {
	timestamp time.Time
	language  string
	scenario  string
	params    map[string]int
}

type request struct {
	timestamp    time.Time
	eventid      string
	status       int
	ttfb         float64
	total        float64
	isCold       bool
	dns          float64
	connect      float64
	tls          float64
	errorMessage string
	target       string
}

type processingStats struct {
	filesProcessed      int
	experimentsInserted int
	requestsInserted    int
}

var knownLanguages = map[string]bool{
	"go":         true,
	"python":     true,
	"rust":       true,
	"typescript": true,
	"ts":         true,
	"quarkus":    true,
	"springboot": true,
	"all":        true,
}

var (
	reRequestsPerSecond = regexp.MustCompile(`RequestsPerSecond:(\d+)`)
	reDuration          = regexp.MustCompile(`Duration:([\d\w]+)`)
	reMaxIdleConns      = regexp.MustCompile(`MaxIdleConns:(\d+)`)
	reMaxIdleConnsHost  = regexp.MustCompile(`MaxIdleConnsPerHost:(\d+)`)
	reIdleConnTimeout   = regexp.MustCompile(`IdleConnTimeout:([\d\w]+)`)
	reTimeout           = regexp.MustCompile(`Timeout:([\d\w]+)`)
)

func main() {
	cfg := parseFlags()
	db := initDB(cfg.dbPath)
	defer db.Close()

	stats, err := processLogs(db, cfg)
	if err != nil {
		log.Fatalf("Error processing logs: %v", err)
	}

	log.Printf("Processing complete. Files: %d, Experiments: %d, Requests: %d",
		stats.filesProcessed, stats.experimentsInserted, stats.requestsInserted)
}

func parseFlags() config {
	c := config{}
	flag.StringVar(&c.dbPath, "db", "benchmark.db", "SQLite database path")
	flag.StringVar(&c.logDir, "logs", "./logs", "Log directory path")
	timeWindow := flag.Int("hours", 24, "Processing time window in hours")
	flag.Parse()

	c.timeWindow = time.Duration(*timeWindow) * time.Hour
	return c
}

func initDB(dbPath string) *sql.DB {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS experiments (
            id INTEGER PRIMARY KEY,
            timestamp DATETIME NOT NULL,
            language TEXT NOT NULL,
            scenario TEXT NOT NULL,
            concurrency INTEGER,
            rps INTEGER,
            requests_per_second INTEGER,
            duration TEXT,
            max_idle_conns INTEGER,
            max_idle_conns_per_host INTEGER,
            idle_conn_timeout TEXT,
            timeout TEXT,
            triggers INTEGER,
			workers INTEGER
        );

		CREATE TABLE IF NOT EXISTS requests (
            id INTEGER PRIMARY KEY,
            experiment_id INTEGER NOT NULL,
            timestamp DATETIME NOT NULL,
            status INTEGER NOT NULL,
            ttfb REAL NOT NULL,
            total_time REAL NOT NULL,
            is_cold BOOLEAN NOT NULL,
            dns_time REAL NOT NULL,
            connect_time REAL NOT NULL,
            tls_time REAL NOT NULL,
            error_message TEXT,
			event_id TEXT,
			target TEXT,
            FOREIGN KEY(experiment_id) REFERENCES experiments(id)
        );
	`)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func processLogs(db *sql.DB, cfg config) (*processingStats, error) {
	stats := &processingStats{}
	cutoff := time.Now().Add(-cfg.timeWindow)

	entries, err := os.ReadDir(cfg.logDir)
	if err != nil {
		return nil, fmt.Errorf("reading log directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}

		expInfo, err := parseFilename(entry.Name())
		if err != nil {
			log.Printf("Skipping invalid filename %q: %v", entry.Name(), err)
			continue
		}

		if expInfo.timestamp.Before(cutoff) {
			continue
		}

		filePath := filepath.Join(cfg.logDir, entry.Name())
		config, requests, err := processFile(filePath)
		if err != nil {
			log.Printf("Error processing %q: %v", entry.Name(), err)
			continue
		}

		expID, err := insertExperiment(db, expInfo, config)
		if err != nil {
			log.Printf("Error inserting experiment: %v", err)
			continue
		}

		if len(requests) > 0 {
			err = insertRequests(db, expID, requests)
			if err != nil {
				log.Printf("Error inserting requests: %v", err)
				continue
			}
		}

		stats.filesProcessed++
		stats.experimentsInserted++
		stats.requestsInserted += len(requests)
	}

	return stats, nil
}

func parseFilename(filename string) (*experimentInfo, error) {
	base := strings.TrimSuffix(filename, ".log")
	parts := strings.Split(base, "_")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid filename format")
	}

	// Handle timestamp which is always the last two parts
	timestamp, err := time.Parse(
		"2006-01-02 15-04-05",
		fmt.Sprintf("%s %s", parts[len(parts)-2], parts[len(parts)-1]),
	)
	if err != nil {
		return nil, fmt.Errorf("parsing timestamp: %w", err)
	}

	// Remove timestamp parts for easier parsing of the rest
	log.Printf("processing : %v", parts)
	parts = parts[:len(parts)-2]
	// Handle eventing scenarios
	if strings.HasPrefix(parts[0], "eventing") {
		scenario := parts[0]
		params := make(map[string]int)

		for _, part := range parts[1:] {
			if strings.HasSuffix(part, "rps") {
				val, err := strconv.Atoi(strings.TrimSuffix(part, "rps"))
				if err != nil {
					return nil, fmt.Errorf("parsing rps: %w", err)
				}
				params["rps"] = val
			}
			if strings.HasSuffix(part, "triggers") {
				val, err := strconv.Atoi(strings.TrimSuffix(part, "triggers"))
				if err != nil {
					return nil, fmt.Errorf("parsing triggers: %w", err)
				}
				params["triggers"] = val
			}
			if strings.HasSuffix(part, "workers") {
				val, err := strconv.Atoi(strings.TrimSuffix(part, "workers"))
				if err != nil {
					return nil, fmt.Errorf("parsing workers: %w", err)
				}
				params["workers"] = val
			}
		}

		return &experimentInfo{
			timestamp: timestamp.UTC(),
			language:  "",
			scenario:  scenario,
			params:    params,
		}, nil
	}

	// Handle serving scenarios
	if strings.HasPrefix(parts[0], "serving") {
		scenario := parts[0]
		params := make(map[string]int)
		var language string

		for _, part := range parts[1:] {
			if strings.HasSuffix(part, "rps") {
				val, err := strconv.Atoi(strings.TrimSuffix(part, "rps"))
				if err != nil {
					return nil, fmt.Errorf("parsing rps: %w", err)
				}
				params["rps"] = val
			} else if knownLanguages[part] {
				language = part
			}
		}

		return &experimentInfo{
			timestamp: timestamp.UTC(),
			language:  language,
			scenario:  scenario,
			params:    params,
		}, nil
	}

	return nil, fmt.Errorf("unrecognized filename format")
}

func processFile(path string) (map[string]interface{}, []request, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 {
		return nil, nil, fmt.Errorf("invalid log file format")
	}

	config := parseConfig(lines[1])
	var requests []request

	for _, line := range lines[2:] {
		if strings.Contains(line, `msg=Success`) || strings.Contains(line, `msg=Failed`) {
			req, err := parseRequest(line)
			if err == nil {
				requests = append(requests, req)
			}
		}
	}

	return config, requests, nil
}

func parseConfig(line string) map[string]interface{} {
	config := make(map[string]interface{})
	matches := reRequestsPerSecond.FindStringSubmatch(line)
	if len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			config["requests_per_second"] = val
		}
	}

	addConfigValue(line, reDuration, "duration", config)
	addConfigValue(line, reMaxIdleConns, "max_idle_conns", config)
	addConfigValue(line, reMaxIdleConnsHost, "max_idle_conns_per_host", config)
	addConfigValue(line, reIdleConnTimeout, "idle_conn_timeout", config)
	addConfigValue(line, reTimeout, "timeout", config)

	return config
}

func addConfigValue(line string, re *regexp.Regexp, key string, config map[string]interface{}) {
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		config[key] = matches[1]
	}
}

func parseRequest(line string) (request, error) {
	var req request
	pairs := parseKeyValuePairs(line)

	if t, ok := pairs["time"]; ok {
		ts, err := time.Parse(time.RFC3339Nano, t)
		if err == nil {
			req.timestamp = ts.UTC()
		}
	}

	if s, ok := pairs["status"]; ok {
		status, _ := strconv.Atoi(s)
		req.status = status
	}

	if t, ok := pairs["target"]; ok {
		req.target = t
	}

	if cold, ok := pairs["isCold"]; ok {
		// First remove escaped quotes if they exist
		cold = strings.ReplaceAll(cold, `\"`, "")
		// Then remove any remaining quotes
		cold = strings.Trim(cold, `"`)
		// Convert to lowercase
		cold = strings.ToLower(cold)
		// Log the value for debugging
		//log.Printf("Processing isCold - original: %q, cleaned: %q", pairs["isCold"], cold)
		req.isCold = cold == "true"
	}

	req.ttfb = parseDuration(pairs["TTFB"])
	req.total = parseDuration(pairs["Total"])
	req.dns = parseDuration(pairs["DNS"])
	req.connect = parseDuration(pairs["Connect"])
	req.tls = parseDuration(pairs["TLS"])

	if pairs["msg"] == "Failed" {
		req.errorMessage = pairs["error"]
	}

	if s, ok := pairs["id"]; ok {
		req.eventid = s
	}

	return req, nil
}

func parseKeyValuePairs(line string) map[string]string {
	result := make(map[string]string)
	// Modified regex to better handle escaped quotes
	re := regexp.MustCompile(`(\w+)=((?:"\\+"[^"]*\\+""|"[^"]*"|[^"\s]+))`)
	matches := re.FindAllStringSubmatch(line, -1)

	for _, m := range matches {
		key := m[1]
		value := m[2]
		// Don't trim quotes here, let the individual parsers handle it
		result[key] = value
	}

	return result
}

// parseDuration converts a duration string to milliseconds by parsing it into a time.Duration
// and converting nanoseconds to milliseconds (dividing by 1e6)
func parseDuration(s string) float64 {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return float64(d.Nanoseconds()) / 1e6
}

func insertExperiment(db *sql.DB, exp *experimentInfo, config map[string]interface{}) (int64, error) {
	stmt := `
        INSERT INTO experiments (
            timestamp, language, scenario, concurrency, rps,
            requests_per_second, duration, max_idle_conns,
            max_idle_conns_per_host, idle_conn_timeout, timeout,
            triggers, workers
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	args := []interface{}{
		exp.timestamp.Format(time.RFC3339),
		exp.language,
		exp.scenario,
		exp.params["concurrency"],
		exp.params["rps"],
		config["requests_per_second"],
		config["duration"],
		config["max_idle_conns"],
		config["max_idle_conns_per_host"],
		config["idle_conn_timeout"],
		config["timeout"],
		exp.params["triggers"],
		exp.params["workers"],
	}

	res, err := db.Exec(stmt, args...)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func insertRequests(db *sql.DB, expID int64, requests []request) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
        INSERT INTO requests (
            experiment_id, timestamp, status, ttfb, total_time,
            is_cold, dns_time, connect_time, tls_time, error_message, event_id, target
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, req := range requests {
		_, err := stmt.Exec(
			expID,
			req.timestamp.Format(time.RFC3339Nano),
			req.status,
			req.ttfb,
			req.total,
			req.isCold,
			req.dns,
			req.connect,
			req.tls,
			req.errorMessage,
			req.eventid,
			req.target,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
