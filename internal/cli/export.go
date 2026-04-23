package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/openmigrate/openmigrate/internal/cli/tui"
	"github.com/openmigrate/openmigrate/internal/core"
	"github.com/openmigrate/openmigrate/internal/core/types"
	"github.com/spf13/cobra"
)

func NewExportCommand(app *App) *cobra.Command {
	var out string
	var only []string
	var exclude []string
	var noHistory bool
	cmd := &cobra.Command{
		Use:   "export",
		Short: "导出 Claude Code 数据",
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := core.Doctor(context.Background(), core.DoctorParams{Mode: types.DoctorModeExport, Verbose: app.verboseWriter()})
			if err != nil {
				return exitf(2, err, "doctor 失败: %v", err)
			}
			for _, item := range report.Items {
				if item.Status == types.DoctorBlock {
					PrintDoctorReport(app.Streams.Out, report)
					return exitf(2, nil, "")
				}
			}
			passphrase, err := ReadPassphrase("请输入导出密码: ", app.Streams)
			if err != nil {
				return passphraseError(err)
			}
			outputDir, finalPackage := resolveExportTarget(out)
			var result core.ExportResult
			err = tui.RunWithProgress(app.Streams.In, app.Streams.Out, app.Streams.ErrOut, !IsNonInteractive(false, app.Streams.In), "正在导出…", app.Verbose, func(update func(string)) error {
				update("正在调用 core 导出…")
				exportResult, exportErr := core.Export(context.Background(), core.ExportParams{
					Agent:         "claude-code",
					Version:       "v2",
					OutputDir:     outputDir,
					Passphrase:    passphrase,
					OnlyScopes:    only,
					ExcludeScopes: exclude,
					NoHistory:     noHistory,
					Verbose:       app.verboseWriter(),
				})
				result = exportResult
				return exportErr
			})
			if err != nil {
				return exportCoreError("导出失败", err)
			}
			if err := moveExportArtifacts(result, finalPackage); err != nil {
				return exitf(2, err, "移动导出文件失败: %v", err)
			}
			result.PackagePath = finalPackage
			result.MetaPath = strings.TrimSuffix(finalPackage, ".ommigrate") + ".meta.json"
			PrintExportSummary(app.Streams.Out, result)
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "输出文件或目录")
	cmd.Flags().StringSliceVar(&only, "only", nil, "只迁移指定 scope（逗号分隔）")
	cmd.Flags().StringSliceVar(&exclude, "exclude", nil, "排除指定 scope（逗号分隔）")
	cmd.Flags().BoolVar(&noHistory, "no-history", false, "排除历史记录（projects/*.jsonl）")
	return cmd
}

func resolveExportTarget(out string) (string, string) {
	if out == "" {
		cwd, _ := os.Getwd()
		stamp := time.Now().Format("20060102-150405")
		return cwd, filepath.Join(cwd, "openmigrate-"+stamp+".ommigrate")
	}
	if strings.HasSuffix(out, ".ommigrate") {
		return filepath.Dir(out), out
	}
	return out, filepath.Join(out, "openmigrate.ommigrate")
}

func moveExportArtifacts(result core.ExportResult, finalPackage string) error {
	finalMeta := strings.TrimSuffix(finalPackage, ".ommigrate") + ".meta.json"
	if err := os.MkdirAll(filepath.Dir(finalPackage), 0o755); err != nil {
		return err
	}
	if result.PackagePath != finalPackage {
		_ = os.Remove(finalPackage)
		if err := os.Rename(result.PackagePath, finalPackage); err != nil {
			return err
		}
	}
	if result.MetaPath != finalMeta {
		_ = os.Remove(finalMeta)
		if err := os.Rename(result.MetaPath, finalMeta); err != nil {
			return err
		}
	}
	return nil
}

func passphraseError(err error) error {
	if errors.Is(err, ErrNonInteractiveNoPassphrase) {
		return exitf(2, err, "无法读取密码，请在 tty 中运行或设置 OPENMIGRATE_PASSPHRASE")
	}
	return exitf(2, err, "读取密码失败: %v", err)
}

func exportCoreError(prefix string, err error) error {
	if errors.Is(err, types.ErrConflictingScopeFilter) {
		return exitf(2, err, "--only 与 --exclude 不能同时指定")
	}
	if errors.Is(err, types.ErrDecryptFailed) {
		return exitf(2, err, "密码错误")
	}
	return exitf(2, err, "%s: %v", prefix, err)
}
