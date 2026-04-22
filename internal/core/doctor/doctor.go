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

type Params struct {
	Mode               types.DoctorMode
	ExpectedPackageSize int64
	AbortOnSkew        bool
	PackageAgentVersion string
}

func Run(params Params, logger *omlog.Logger) (types.DoctorReport, error) {
	report := types.DoctorReport{}
	add := func(name string, status types.DoctorStatus, message string) {
		report.Items = append(report.Items, types.DoctorItem{Name: name, Status: status, Message: message})
	}

	if output, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
		add("macOS", types.DoctorPass, strings.TrimSpace(string(output)))
	} else {
		add("macOS", types.DoctorWarn, "无法读取 macOS 版本")
	}

	claudePath, err := exec.LookPath("claude")
	if err != nil {
		add("claude", types.DoctorBlock, "未找到 Claude Code 可执行文件")
	} else {
		versionOutput, versionErr := exec.Command(claudePath, "--version").Output()
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

	home, err := os.UserHomeDir()
	if err == nil {
		fdaPath := filepath.Join(home, "Library", "Application Support", "Claude")
		if _, err := os.ReadDir(fdaPath); err != nil {
			if os.IsPermission(err) {
				add("full-disk-access", types.DoctorBlock, "Full Disk Access 未授权")
			} else {
				add("full-disk-access", types.DoctorWarn, "Claude 数据目录不存在或不可读")
			}
		} else {
			add("full-disk-access", types.DoctorPass, "Claude 数据目录可读")
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

func sameMajor(left, right string) bool {
	re := regexp.MustCompile(`\d+`)
	l := re.FindString(left)
	r := re.FindString(right)
	return l != "" && l == r
}
