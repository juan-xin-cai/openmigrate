package pathscan

import (
	"io/ioutil"
	"regexp"
	"sort"
	"strings"

	"github.com/openmigrate/openmigrate/internal/core/rewrite"
	"github.com/openmigrate/openmigrate/internal/core/types"
)

var pathExpr = regexp.MustCompile(`/(?:Users/[^/\s"']+|[A-Za-z0-9._-]+)(?:/[^\s"'<>]+)+`)

func Scan(manifest types.Manifest) (types.PathScanResult, error) {
	homeCount := make(map[string]int)
	projectCount := make(map[string]int)
	externalSet := make(map[string]struct{})

	for _, entry := range manifest.Entries {
		if entry.IsDir || entry.IsSymlink {
			continue
		}
		isText, err := rewrite.IsTextFile(entry.SourcePath)
		if err != nil || !isText {
			continue
		}
		data, err := ioutil.ReadFile(entry.SourcePath)
		if err != nil {
			return types.PathScanResult{}, err
		}
		matches := pathExpr.FindAllString(string(data), -1)
		for _, match := range matches {
			match = strings.Trim(match, `"'.,)`)
			home := inferHome(match)
			if home == "" {
				externalSet[match] = struct{}{}
				continue
			}
			homeCount[home]++
		}
	}

	homePrefix := topByCount(homeCount)
	if homePrefix == "" {
		return types.PathScanResult{}, nil
	}

	for _, entry := range manifest.Entries {
		if entry.IsDir || entry.IsSymlink {
			continue
		}
		isText, err := rewrite.IsTextFile(entry.SourcePath)
		if err != nil || !isText {
			continue
		}
		data, err := ioutil.ReadFile(entry.SourcePath)
		if err != nil {
			return types.PathScanResult{}, err
		}
		matches := pathExpr.FindAllString(string(data), -1)
		for _, match := range matches {
			match = strings.Trim(match, `"'.,)`)
			if strings.HasPrefix(match, homePrefix+"/") {
				if projectRoot := inferProjectRoot(match, homePrefix); projectRoot != "" {
					projectCount[projectRoot]++
				}
				continue
			}
			externalSet[match] = struct{}{}
		}
	}

	projectRoots := orderCounts(projectCount)
	external := make([]string, 0, len(externalSet))
	for item := range externalSet {
		external = append(external, item)
	}
	sort.Strings(external)

	return types.PathScanResult{
		HomePrefix:    homePrefix,
		ProjectRoots:  projectRoots,
		ExternalPaths: external,
	}, nil
}

func inferHome(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) < 4 || parts[1] != "Users" || parts[2] == "" {
		return ""
	}
	return "/Users/" + parts[2]
}

func inferProjectRoot(pathValue, home string) string {
	rest := strings.TrimPrefix(pathValue, home+"/")
	parts := strings.Split(rest, "/")
	if len(parts) < 2 || strings.HasPrefix(parts[0], ".") {
		return ""
	}
	return home + "/" + parts[0] + "/" + parts[1]
}

func topByCount(items map[string]int) string {
	best := ""
	bestCount := -1
	for key, count := range items {
		if count > bestCount || (count == bestCount && key < best) {
			best = key
			bestCount = count
		}
	}
	return best
}

func orderCounts(items map[string]int) []string {
	type pair struct {
		Key   string
		Count int
	}
	ordered := make([]pair, 0, len(items))
	for key, count := range items {
		ordered = append(ordered, pair{Key: key, Count: count})
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Count == ordered[j].Count {
			return ordered[i].Key < ordered[j].Key
		}
		return ordered[i].Count > ordered[j].Count
	})
	out := make([]string, 0, len(ordered))
	for _, item := range ordered {
		out = append(out, item.Key)
	}
	return out
}
