package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/openmigrate/openmigrate/internal/core/types"
)

func TestPrintDoctorReportShowsSuggestionForBlock(t *testing.T) {
	var out bytes.Buffer
	PrintDoctorReport(&out, types.DoctorReport{
		Items: []types.DoctorItem{
			{Name: "full-disk-access", Status: types.DoctorBlock, Message: "Full Disk Access 未授权"},
		},
	})
	text := out.String()
	if !strings.Contains(text, "BLOCK") || !strings.Contains(text, "建议") {
		t.Fatalf("doctor report = %q", text)
	}
	if !strings.Contains(text, "Full Disk Access") {
		t.Fatalf("doctor suggestion missing: %q", text)
	}
}

func TestPrintPostInstallChecklistAndNoPassphraseLeak(t *testing.T) {
	var out bytes.Buffer
	PrintPostInstallChecklist(&out, []types.CheckItem{
		{Category: "hooks", Name: "hooks.pre.command", Status: types.CheckWarn, Message: "file does not exist"},
	})
	text := out.String()
	if !strings.Contains(text, "不会自动修改任何配置") {
		t.Fatalf("checklist = %q", text)
	}
	if strings.Contains(strings.ToLower(text), "passphrase") {
		t.Fatalf("passphrase leaked: %q", text)
	}
}
