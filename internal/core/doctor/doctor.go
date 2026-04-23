package doctor

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	omlog "github.com/openmigrate/openmigrate/internal/core/log"
	"github.com/openmigrate/openmigrate/internal/core/types"
)

var (
	execCommand    = exec.Command
	lookPath       = exec.LookPath
	desktopAppPath = "/Applications/Claude.app"
)

type Params struct {
	Mode                types.DoctorMode
	ExpectedPackageSize int64
	AbortOnSkew         bool
	PackageAgentVersion string
}

func Run(params Params, logger *omlog.Logger) (types.DoctorReport, error) {
	report := types.DoctorReport{}
	add := func(name string, status types.DoctorStatus, message string) {
		report.Items = append(report.Items, types.DoctorItem{Name: name, Status: status, Message: message})
	}

	if output, err := execCommand("sw_vers", "-productVersion").Output(); err == nil {
		add("macOS", types.DoctorPass, strings.TrimSpace(string(output)))
	} else {
		add("macOS", types.DoctorWarn, "无法读取 macOS 版本")
	}

	claudePath, err := lookPath("claude")
	if err != nil {
		add("claude", types.DoctorBlock, "未找到 Claude Code 可执行文件")
	} else {
		versionOutput, versionErr := execCommand(claudePath, "--version").Output()
		if versionErr != nil {
			add("claude", types.DoctorBlock, "Claude Code 无法返回版本")
		} else {
			version := strings.TrimSpace(string(versionOutput))
			add("claude", types.DoctorPass, version)
			if params.Mode == types.DoctorModeImport && params.PackageAgentVersion != "" {
				status := types.DoctorWarn
				message := "版本跨度较大"
				if sameMajor(version, params.PackageAgentVersion) {
					status = types.DoctorPass
					message = "版本兼容"
				} else if params.AbortOnSkew {
					status = types.DoctorBlock
					message = "版本跨度较大，已按参数阻止继续"
				}
				add("version-skew", status, message)
			}
		}
	}

	if _, err := os.Stat(desktopAppPath); err != nil {
		add("claude-desktop", types.DoctorWarn, "未找到 Claude Desktop")
	} else {
		add("claude-desktop", types.DoctorPass, "已安装 Claude Desktop")
		version, err := readDesktopVersion()
		if err != nil || version == "" {
			add("claude-desktop-version", types.DoctorWarn, "Claude Desktop version unreadable")
		} else {
			add("claude-desktop-version", types.DoctorPass, version)
		}
	}

	home, err := os.UserHomeDir()
	if err == nil {
		if params.Mode == types.DoctorModeImport {
			desktopDataPath := filepath.Join(home, "Library", "Application Support", "Claude")
			if _, err := os.ReadDir(desktopDataPath); err != nil {
				if os.IsPermission(err) {
					add("claude-desktop-full-disk-access", types.DoctorBlock, "Full Disk Access for Claude Desktop 未授权")
				} else {
					add("claude-desktop-full-disk-access", types.DoctorBlock, "Full Disk Access for Claude Desktop 未授权或数据目录不可读")
				}
			} else {
				add("claude-desktop-full-disk-access", types.DoctorPass, "Claude Desktop 数据目录可读")
			}
		}

		if params.ExpectedPackageSize > 0 {
			var stat syscall.Statfs_t
			if err := syscall.Statfs(home, &stat); err == nil {
				free := int64(stat.Bavail) * int64(stat.Bsize)
				if free < params.ExpectedPackageSize*2 {
					add("disk-space", types.DoctorBlock, "磁盘空间不足，无法完成快照与导入")
				} else {
					add("disk-space", types.DoctorPass, "磁盘空间充足")
				}
			}
		}
	}

	if logger != nil {
		logger.Info("doctor finished", map[string]interface{}{"items": len(report.Items)})
	}
	return report, nil
}

func readDesktopVersion() (string, error) {
	output, err := execCommand("plutil", "-extract", "CFBundleShortVersionString", "xml1", filepath.Join(desktopAppPath, "Contents", "Info.plist"), "-o", "-").Output()
	if err != nil {
		return "", err
	}
	re := regexp.MustCompile(`<string>([^<]+)</string>`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) == 2 {
		return strings.TrimSpace(matches[1]), nil
	}
	return strings.TrimSpace(string(output)), nil
}

func sameMajor(left, right string) bool {
	re := regexp.MustCompile(`\d+`)
	l := re.FindString(left)
	r := re.FindString(right)
	return l != "" && l == r
}
