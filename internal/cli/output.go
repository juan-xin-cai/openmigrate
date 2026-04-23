package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/openmigrate/openmigrate/internal/core"
	"github.com/openmigrate/openmigrate/internal/core/types"
)

func PrintDoctorReport(out io.Writer, report types.DoctorReport) {
	writer := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
	fmt.Fprintln(writer, "状态\t检查项\t说明")
	for _, item := range report.Items {
		fmt.Fprintf(writer, "%s\t%s\t%s\n", doctorStatusLabel(item.Status), item.Name, item.Message)
		if item.Status == types.DoctorBlock {
			fmt.Fprintf(writer, "\t建议\t%s\n", doctorSuggestion(item))
		}
	}
	_ = writer.Flush()
}

func PrintExportSummary(out io.Writer, result core.ExportResult) {
	fmt.Fprintf(out, "导出完成\n包文件: %s\n元信息: %s\n日志: %s\n", result.PackagePath, result.MetaPath, result.LogPath)
}

func PrintInspectResult(out io.Writer, meta types.PackageMeta) {
	agentTypes := meta.AgentTypes
	if len(agentTypes) == 0 && meta.Agent != "" {
		agentTypes = []string{meta.Agent}
	}
	fmt.Fprintf(out, "主机名:     %s\n", meta.Hostname)
	fmt.Fprintf(out, "创建时间:   %s\n", meta.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	fmt.Fprintf(out, "Agent 类型: %s\n", strings.Join(agentTypes, ", "))
	fmt.Fprintf(out, "Agent 版本: %s\n", meta.AgentVersion)
	fmt.Fprintf(out, "文件数:     %d\n", meta.FileCount)
	fmt.Fprintf(out, "总大小:     %s\n", formatBytes(meta.TotalSize))
	if meta.OwnerAccountID != "" {
		fmt.Fprintf(out, "Desktop 账号 ID: %s\n", meta.OwnerAccountID)
	}
}

func PrintImportSummary(out io.Writer, result types.ImportResult) {
	fmt.Fprintf(out, "导入完成\n新增: %d\n更新: %d\n跳过: %d\n日志: %s\n", len(result.Written), len(result.Updated), len(result.Skipped), result.LogPath)
}

func PrintPostInstallChecklist(out io.Writer, items []types.CheckItem) {
	if len(items) == 0 {
		fmt.Fprintln(out, "安装后检查: 无异常")
		return
	}
	fmt.Fprintln(out, "安装后检查:")
	for _, item := range items {
		fmt.Fprintf(out, "- [%s] %s: %s\n", item.Category, item.Name, item.Message)
	}
	fmt.Fprintln(out, "不会自动修改任何配置")
}

func PrintRollbackSummary(out io.Writer, snapshotID string) {
	fmt.Fprintf(out, "回滚完成\n快照: %s\n", snapshotID)
}

func doctorStatusLabel(status types.DoctorStatus) string {
	switch status {
	case types.DoctorPass:
		return "PASS"
	case types.DoctorWarn:
		return "WARN"
	default:
		return "BLOCK"
	}
}

func doctorSuggestion(item types.DoctorItem) string {
	if item.Name == "claude-desktop-full-disk-access" {
		return "在系统设置 > 隐私与安全 > 完全磁盘访问权限中添加 Claude Desktop，然后重试"
	}
	lower := strings.ToLower(item.Name + " " + item.Message)
	switch {
	case strings.Contains(lower, "full-disk"):
		return "给终端授予 Full Disk Access 后重试"
	case strings.Contains(lower, "claude"):
		return "确认 Claude Code 已安装且可执行"
	case strings.Contains(lower, "disk"):
		return "释放磁盘空间后重试"
	default:
		return "按提示修复后重试"
	}
}

func formatBytes(size int64) string {
	const unit = 1024
	switch {
	case size >= unit*unit:
		return fmt.Sprintf("%.1f MB", float64(size)/float64(unit*unit))
	case size >= unit:
		return fmt.Sprintf("%.1f KB", float64(size)/float64(unit))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
