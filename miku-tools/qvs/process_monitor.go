package qvs

import (
	"mikutool/config"
	"os"
	"time"
)

// ProcMonitor monitors the process specified in config.Process and sends a notification if it restarts.
func ProcMonitor(config *config.Config) {
	pid := config.Process
	if pid == 0 {
		return
	}

	// Check process status periodically
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var lastSeen time.Time
	for range ticker.C {
		// Check if process is still running
		_, err := os.FindProcess(pid)
		if err != nil {
			// Process not found, assume it restarted
			if !lastSeen.IsZero() {
				SendMailWithConfig(config, "Process with PID "+string(rune(pid))+" has restarted.")
			}
			lastSeen = time.Now()
		}
	}
}
