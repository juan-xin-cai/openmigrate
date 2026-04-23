package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

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

func TestPrintInspectResult(t *testing.T) {
	var out bytes.Buffer
	PrintInspectResult(&out, types.PackageMeta{
		Hostname:       "macbook",
		CreatedAt:      time.Date(2026, 4, 23, 12, 0, 0, 0, time.FixedZone("CST", 8*3600)),
		AgentTypes:     []string{"claude-code", "claude-desktop"},
		AgentVersion:   "v2",
		FileCount:      12,
		TotalSize:      1536,
		OwnerAccountID: "acct-1",
	})
	text := out.String()
	for _, needle := range []string{"主机名:", "macbook", "Agent 类型: claude-code, claude-desktop", "Agent 版本: v2", "文件数:     12", "总大小:     1.5 KB", "Desktop 账号 ID: acct-1"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("inspect output missing %q: %q", needle, text)
		}
	}
}

func TestPrintInspectResultOmitsOwnerAccountIDWhenEmpty(t *testing.T) {
	var out bytes.Buffer
	PrintInspectResult(&out, types.PackageMeta{Hostname: "macbook"})
	if strings.Contains(out.String(), "Desktop 账号 ID") {
		t.Fatalf("unexpected owner account line: %q", out.String())
	}
}

func TestDoctorSuggestionForDesktopFDA(t *testing.T) {
	suggestion := doctorSuggestion(types.DoctorItem{
		Name:    "claude-desktop-full-disk-access",
		Status:  types.DoctorBlock,
		Message: "Full Disk Access for Claude Desktop 未授权",
	})
	if !strings.Contains(suggestion, "Claude Desktop") || strings.Contains(suggestion, "终端") {
		t.Fatalf("unexpected suggestion: %q", suggestion)
	}
}

func TestDoctorSuggestionForDiskSpaceDoesNotMentionDesktop(t *testing.T) {
	suggestion := doctorSuggestion(types.DoctorItem{
		Name:    "disk-space",
		Status:  types.DoctorBlock,
		Message: "磁盘空间不足",
	})
	if strings.Contains(suggestion, "Claude Desktop") {
		t.Fatalf("unexpected suggestion: %q", suggestion)
	}
}
