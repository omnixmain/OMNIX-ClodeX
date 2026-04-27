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

func BeautifyFileName(name string) (string, string, string) {
	// Remove extension
	ext := filepath.Ext(name)
	name = strings.TrimSuffix(name, ext)

	// Quality extraction (e.g., 480p, 720p, 1080p, 2160p, 4k)
	qualityRegex := regexp.MustCompile(`(?i)(480p|720p|1080p|2160p|4k)`)
	quality := qualityRegex.FindString(name)
	if quality != "" {
		quality = strings.ToUpper(quality)
	} else {
		quality = "Unknown"
	}

	// Season and Episode extraction
	seasonEpisodeRegex := regexp.MustCompile(`(?i)S(\d+)E(\d+)`)
	seMatch := seasonEpisodeRegex.FindStringSubmatch(name)
	seasonEpisode := ""
	if len(seMatch) == 3 {
		seasonEpisode = fmt.Sprintf("Season %s • Episode %s", seMatch[1], seMatch[2])
	}

	// Replace underscores and dots with spaces
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, ".", " ")

	// Remove quality and season/episode from name if present to avoid duplication
	name = qualityRegex.ReplaceAllString(name, "")
	name = seasonEpisodeRegex.ReplaceAllString(name, "")

	// Clean up extra spaces
	name = strings.Join(strings.Fields(name), " ")

	// Final trim and clean up
	name = strings.TrimSpace(name)
	name = strings.Trim(name, "- ")

	return name, quality, seasonEpisode
}
