package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/openmigrate/openmigrate/internal/cli/tui"
	"github.com/openmigrate/openmigrate/internal/core"
	"github.com/openmigrate/openmigrate/internal/core/types"
	"github.com/spf13/cobra"
)

func NewImportCommand(app *App) *cobra.Command {
	var yes bool
	var skipDesktopCheck bool
	cmd := &cobra.Command{
		Use:   "import <package.ommigrate>",
		Short: "导入 OpenMigrate 包",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nonInteractive := IsNonInteractive(yes, app.Streams.In)
			passphrase, err := ReadPassphrase("请输入导入密码: ", app.Streams)
			if err != nil {
				return passphraseError(err)
			}
			preview, err := core.PreviewImport(context.Background(), core.ImportPreviewParams{
				PackagePath: args[0],
				Passphrase:  passphrase,
				Verbose:     app.verboseWriter(),
			})
			if err != nil {
				return exportCoreError("导入预览失败", err)
			}
			printAccountCheckStatus(app.Streams.Out, preview.Meta)

			mapping := preview.SuggestedMapping
			if !nonInteractive {
				mapping, err = tui.RunPathMapping(app.Streams.In, app.Streams.Out, preview)
				if err != nil {
					return exitf(1, err, "已取消路径映射")
				}
			}

			skipFlag := skipDesktopCheck
			conflicts, err := core.Import(context.Background(), core.ImportParams{
				PackagePath:             args[0],
				Passphrase:              passphrase,
				TargetHome:              mapping.TargetHome,
				Mapping:                 mapping,
				SkipDesktopSessionCheck: skipFlag,
				Verbose:                 app.verboseWriter(),
			})
			if err != nil {
				if errors.Is(err, types.ErrAccountMismatch) {
					skipFlag, err = confirmSkipDesktopAccountCheck(app.Streams.In, app.Streams.Out, nonInteractive, err)
					if err != nil {
						return err
					}
					conflicts, err = core.Import(context.Background(), core.ImportParams{
						PackagePath:             args[0],
						Passphrase:              passphrase,
						TargetHome:              mapping.TargetHome,
						Mapping:                 mapping,
						SkipDesktopSessionCheck: skipFlag,
						Verbose:                 app.verboseWriter(),
					})
				}
				if err != nil {
					return exportCoreError("冲突预检失败", err)
				}
			}

			decisions := defaultConflictDecisions(conflicts)
			if !nonInteractive && hasConflicts(conflicts) {
				decisions, err = tui.RunConflictMerge(app.Streams.In, app.Streams.Out, conflicts, decisions)
				if err != nil {
					return exitf(1, err, "已取消冲突处理")
				}
			}

			var result types.ImportResult
			err = tui.RunWithProgress(app.Streams.In, app.Streams.Out, app.Streams.ErrOut, !nonInteractive, "正在导入…", app.Verbose, func(update func(string)) error {
				update("正在调用 core 写入…")
				importResult, importErr := core.ApplyImport(context.Background(), core.ImportApplyParams{
					PackagePath:             args[0],
					Passphrase:              passphrase,
					Mapping:                 mapping,
					Decisions:               decisions,
					SkipDesktopSessionCheck: skipFlag,
					Verbose:                 app.verboseWriter(),
				})
				result = importResult
				return importErr
			})
			if err != nil {
				return exportCoreError("导入失败", err)
			}
			PrintImportSummary(app.Streams.Out, result)
			PrintPostInstallChecklist(app.Streams.Out, result.Checks)
			fmt.Fprintf(app.Streams.Out, "回滚命令: openmigrate rollback --snapshot %s\n", result.Snapshot.ID)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "非交互式执行并采用默认决策")
	cmd.Flags().BoolVar(&skipDesktopCheck, "skip-desktop-session-check", false, "跳过 Desktop sessions 账号校验")
	return cmd
}

func defaultConflictDecisions(report types.ConflictReport) types.ConflictDecision {
	actions := make(map[string]types.DecisionAction)
	for _, bucket := range report.Buckets {
		for _, item := range bucket.Conflicts {
			actions[item.Key] = types.DecisionKeepTarget
		}
	}
	return types.ConflictDecision{Actions: actions}
}

func hasConflicts(report types.ConflictReport) bool {
	for _, bucket := range report.Buckets {
		if len(bucket.Conflicts) > 0 {
			return true
		}
	}
	return false
}

func printAccountCheckStatus(out io.Writer, meta types.PackageMeta) {
	if meta.OwnerAccountID != "" {
		fmt.Fprintln(out, "账号检查: 待验证（包含 Desktop sessions）")
		return
	}
	fmt.Fprintln(out, "账号检查: 无 Desktop sessions，跳过")
}

func confirmSkipDesktopAccountCheck(in io.Reader, out io.Writer, nonInteractive bool, err error) (bool, error) {
	fmt.Fprintln(out, "账号不一致：请先打开 Claude Desktop 并登录同一 Anthropic 账号。")
	if nonInteractive {
		return false, exitf(1, err, "非交互式模式下账号不一致，中止")
	}
	fmt.Fprint(out, "是否忽略并继续导入？（Desktop sessions 将产生 orphan）[y/N] ")
	reader := bufio.NewReader(in)
	reply, readErr := reader.ReadString('\n')
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return false, exitf(1, readErr, "读取确认输入失败: %v", readErr)
	}
	if strings.EqualFold(strings.TrimSpace(reply), "y") {
		return true, nil
	}
	return false, exitf(1, nil, "已取消")
}
