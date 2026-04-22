package cli

import (
	"context"
	"os"
	"path/filepath"

	"github.com/openmigrate/openmigrate/internal/core"
	"github.com/openmigrate/openmigrate/internal/core/pack"
	"github.com/openmigrate/openmigrate/internal/core/types"
	"github.com/spf13/cobra"
)

func NewDoctorCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor [package-or-meta]",
		Short: "检查当前环境",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := core.DoctorParams{Mode: types.DoctorModeExport, Verbose: app.verboseWriter()}
			if len(args) == 1 {
				params.Mode = types.DoctorModeImport
				meta, err := readMetaForDoctor(args[0])
				if err != nil {
					return exitf(2, err, "读取包元信息失败: %v", err)
				}
				params.PackageAgentVersion = meta.AgentVersion
			}
			report, err := core.Doctor(context.Background(), params)
			if err != nil {
				return exitf(2, err, "doctor 失败: %v", err)
			}
			PrintDoctorReport(app.Streams.Out, report)
			for _, item := range report.Items {
				if item.Status == types.DoctorBlock {
					return exitf(2, nil, "")
				}
			}
			return nil
		},
	}
	return cmd
}

func readMetaForDoctor(path string) (types.PackageMeta, error) {
	if filepath.Ext(path) == ".json" {
		return pack.ReadMeta(path)
	}
	metaPath := path[:len(path)-len(filepath.Ext(path))] + ".meta.json"
	if _, err := os.Stat(metaPath); err == nil {
		return pack.ReadMeta(metaPath)
	}
	return types.PackageMeta{}, os.ErrNotExist
}
