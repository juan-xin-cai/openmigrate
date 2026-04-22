package cli

import (
	"context"
	"errors"

	"github.com/openmigrate/openmigrate/internal/cli/tui"
	"github.com/openmigrate/openmigrate/internal/core"
	"github.com/openmigrate/openmigrate/internal/core/types"
	"github.com/spf13/cobra"
)

func NewRollbackCommand(app *App) *cobra.Command {
	var snapshotID string
	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "回滚最近一次导入",
		RunE: func(cmd *cobra.Command, args []string) error {
			passphrase, err := ReadPassphrase("请输入回滚密码: ", app.Streams)
			if err != nil {
				return passphraseError(err)
			}
			err = tui.RunWithProgress(app.Streams.In, app.Streams.Out, app.Streams.ErrOut, !IsNonInteractive(false, app.Streams.In), "正在回滚…", app.Verbose, func(update func(string)) error {
				update("正在恢复快照…")
				return core.Rollback(context.Background(), core.RollbackParams{
					SnapshotID: snapshotID,
					Passphrase: passphrase,
					Verbose:    app.verboseWriter(),
				})
			})
			if err != nil {
				if errors.Is(err, types.ErrSnapshotNotFound) {
					return exitf(2, err, "没有可回滚的快照")
				}
				return exportCoreError("回滚失败", err)
			}
			if snapshotID == "" {
				snapshotID = "latest"
			}
			PrintRollbackSummary(app.Streams.Out, snapshotID)
			return nil
		},
	}
	cmd.Flags().StringVar(&snapshotID, "snapshot", "latest", "快照 ID，默认 latest")
	return cmd
}
