package config

import (
	"bufio"
	"log"
	"os"
	"strings"
)

// LoadEnv loads key-value pairs from a .env file into environment variables
func LoadEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		// File does not exist, ignore
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`) // Strip quotes if any

			// Only set if not already set in environment
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, val)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Warning: failed to read .env: %v", err)
	}
}
