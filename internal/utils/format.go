package utils

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

func HumanReadableSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func BeautifyFileName(name string) (string, string) {
	// Remove extension
	ext := filepath.Ext(name)
	name = strings.TrimSuffix(name, ext)

	// Quality extraction (e.g., 720p, 1080p, 4k)
	qualityRegex := regexp.MustCompile(`(?i)(480p|720p|1080p|2160p|4k)`)
	quality := qualityRegex.FindString(name)
	if quality != "" {
		quality = strings.ToUpper(quality)
	} else {
		quality = "Unknown"
	}

	// Replace underscores and dots with spaces
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, ".", " ")

	// Remove quality from name if present to avoid duplication
	name = qualityRegex.ReplaceAllString(name, "")

	// Clean up extra spaces
	name = strings.Join(strings.Fields(name), " ")

	// Add hyphens for common patterns (S01E01, etc.)
	seasonRegex := regexp.MustCompile(`(?i)(S\d+E\d+)`)
	name = seasonRegex.ReplaceAllStringFunc(name, func(s string) string {
		return " - " + strings.ToUpper(s) + " - "
	})

	// Final trim and clean up
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " - - ", " - ")
	name = strings.Trim(name, "- ")

	return name, quality
}
