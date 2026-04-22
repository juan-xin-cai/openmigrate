package postcheck

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/openmigrate/openmigrate/internal/core/types"
)

func Check(targetHome string, rewriteReport types.RewriteReport) (types.CheckReport, error) {
	report := types.CheckReport{}
	settingsPath := filepath.Join(targetHome, ".claude", "settings.json")
	if data, err := ioutil.ReadFile(settingsPath); err == nil {
		var parsed interface{}
		if err := json.Unmarshal(data, &parsed); err == nil {
			commands := make([]commandRef, 0, 8)
			scanCommands(parsed, nil, &commands)
			for _, command := range commands {
				if err := checkCommand(command.Command); err != nil {
					category := "hooks"
					if strings.Contains(strings.Join(command.Path, "."), "mcp") {
						category = "mcp"
					}
					report.Items = append(report.Items, types.CheckItem{
						Category: category,
						Name:     strings.Join(command.Path, "."),
						Status:   types.CheckWarn,
						Message:  err.Error(),
					})
				}
			}
		}
	}

	for _, projectRoot := range rewriteReport.ProjectRoots {
		if _, err := os.Stat(projectRoot); err != nil {
			report.Items = append(report.Items, types.CheckItem{
				Category: "project-root",
				Name:     projectRoot,
				Status:   types.CheckWarn,
				Message:  "项目根不存在",
			})
		}
	}

	for _, path := range rewriteReport.ExternalPaths {
		if err := checkCommand(path); err != nil {
			report.Items = append(report.Items, types.CheckItem{
				Category: "external-tool",
				Name:     path,
				Status:   types.CheckWarn,
				Message:  err.Error(),
			})
		}
	}
	return report, nil
}

type commandRef struct {
	Path    []string
	Command string
}

func scanCommands(node interface{}, current []string, out *[]commandRef) {
	switch typed := node.(type) {
	case map[string]interface{}:
		for key, value := range typed {
			next := append(append([]string(nil), current...), key)
			if key == "command" {
				if cmd, ok := value.(string); ok {
					*out = append(*out, commandRef{Path: next, Command: cmd})
				}
			}
			scanCommands(value, next, out)
		}
	case []interface{}:
		for _, value := range typed {
			scanCommands(value, current, out)
		}
	}
}

func checkCommand(command string) error {
	if command == "" {
		return nil
	}
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return nil
	}
	execName := fields[0]
	if filepath.IsAbs(execName) {
		info, err := os.Stat(execName)
		if err != nil {
			return err
		}
		if info.Mode()&0o111 == 0 {
			return os.ErrPermission
		}
		return nil
	}
	_, err := exec.LookPath(execName)
	return err
}
