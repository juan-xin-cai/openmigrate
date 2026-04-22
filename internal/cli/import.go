package cli

import (
	"context"
	"fmt"

	"github.com/openmigrate/openmigrate/internal/cli/tui"
	"github.com/openmigrate/openmigrate/internal/core"
	"github.com/openmigrate/openmigrate/internal/core/types"
	"github.com/spf13/cobra"
)

func NewImportCommand(app *App) *cobra.Command {
	var yes bool
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
			fmt.Fprintln(app.Streams.Out, "账号检查: PASS（M1 无 Desktop 数据）")

			mapping := preview.SuggestedMapping
			if !nonInteractive {
				mapping, err = tui.RunPathMapping(app.Streams.In, app.Streams.Out, preview)
				if err != nil {
					return exitf(1, err, "已取消路径映射")
				}
			}

			conflicts, err := core.Import(context.Background(), core.ImportParams{
				PackagePath: args[0],
				Passphrase:  passphrase,
				TargetHome:  mapping.TargetHome,
				Mapping:     mapping,
				Verbose:     app.verboseWriter(),
			})
			if err != nil {
				return exportCoreError("冲突预检失败", err)
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
					PackagePath: args[0],
					Passphrase:  passphrase,
					Mapping:     mapping,
					Decisions:   decisions,
					Verbose:     app.verboseWriter(),
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
