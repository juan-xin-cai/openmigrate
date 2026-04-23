package cli

import (
	"context"
	"errors"

	"github.com/openmigrate/openmigrate/internal/core"
	"github.com/openmigrate/openmigrate/internal/core/types"
	"github.com/spf13/cobra"
)

func NewInspectCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect <package.ommigrate>",
		Short: "预览包元信息",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			meta, err := core.Inspect(context.Background(), types.InspectParams{PkgPath: args[0]})
			if err != nil {
				if errors.Is(err, types.ErrMetaNotFound) {
					return exitf(2, err, "找不到元信息文件：%v", err)
				}
				return exitf(2, err, "inspect 失败: %v", err)
			}
			PrintInspectResult(app.Streams.Out, meta)
			return nil
		},
	}
	return cmd
}
