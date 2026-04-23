package cli

import (
	"errors"
	"fmt"

	"github.com/openmigrate/openmigrate/internal/buildinfo"
	"github.com/spf13/cobra"
)

func Execute(streams Streams) int {
	app := &App{Streams: streams}
	cmd := NewRootCommand(app)
	if err := cmd.Execute(); err != nil {
		if !errors.Is(err, ErrUserCanceled) && err.Error() != "" {
			fmt.Fprintln(streams.ErrOut, err.Error())
		}
		return exitCode(err)
	}
	return 0
}

func NewRootCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "openmigrate",
		Short:         "OpenMigrate CLI",
		Version:       buildinfo.Summary(),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetVersionTemplate("{{printf \"%s\\n\" .Version}}")
	cmd.SetIn(app.Streams.In)
	cmd.SetOut(app.Streams.Out)
	cmd.SetErr(app.Streams.ErrOut)
	cmd.PersistentFlags().BoolVar(&app.Verbose, "verbose", false, "输出详细日志到 stderr")
	cmd.AddCommand(
		NewDoctorCommand(app),
		NewExportCommand(app),
		NewInspectCommand(app),
		NewImportCommand(app),
		NewRollbackCommand(app),
	)
	return cmd
}
