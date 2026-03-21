package wtr

import "strings"

// filterFiles returns files whose paths contain the query (case-insensitive).
func filterFiles(files []string, query string) []string {
	if query == "" {
		return files
	}
	q := strings.ToLower(query)
	var result []string
	for _, f := range files {
		if strings.Contains(strings.ToLower(f), q) {
			result = append(result, f)
		}
	}
	return result
}
