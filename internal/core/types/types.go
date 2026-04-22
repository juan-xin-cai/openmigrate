package types

import (
	"os"
	"path"
	"strings"
	"time"
)

type Strategy string

const (
	StrategyInclude    Strategy = "include"
	StrategyExclude    Strategy = "exclude"
	StrategyFieldStrip Strategy = "field-strip"
)

type AgentConfig struct {
	Agent   string           `json:"agent"`
	Version string           `json:"version"`
	Roots   []string         `json:"roots"`
	Entries []WhitelistEntry `json:"entries"`
}

type WhitelistEntry struct {
	Path     string   `json:"path"`
	Strategy Strategy `json:"strategy"`
	Fields   []string `json:"fields,omitempty"`
}

type Manifest struct {
	SourceHome string
	Entries    []FileEntry
	Links      []LinkRelation
	TotalSize  int64
}

type FileEntry struct {
	SourcePath       string
	RelativePath     string
	Mode             os.FileMode
	Size             int64
	Strategy         Strategy
	IsDir            bool
	IsSymlink        bool
	SymlinkTarget    string
	ResolvedPath     string
	ExternalSymlink  bool
	GroupKey         string
	ContentSHA256    string
	OriginalContents []byte
}

type LinkRelation struct {
	LinkRelativePath   string `json:"link_relative_path"`
	TargetRelativePath string `json:"target_relative_path,omitempty"`
	OriginalTarget     string `json:"original_target"`
	External           bool   `json:"external"`
	Warning            string `json:"warning,omitempty"`
}

type PathScanResult struct {
	HomePrefix    string   `json:"home_prefix"`
	ProjectRoots  []string `json:"project_roots"`
	ExternalPaths []string `json:"external_paths"`
}

type PathPair struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type PathMapping struct {
	SourceHome      string     `json:"source_home"`
	TargetHome      string     `json:"target_home"`
	ProjectMappings []PathPair `json:"project_mappings"`
	ExternalPaths   []string   `json:"external_paths"`
}

type RewriteWarning struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type RewriteReport struct {
	RewrittenFiles int              `json:"rewritten_files"`
	SkippedBinary  []string         `json:"skipped_binary"`
	Warnings       []RewriteWarning `json:"warnings"`
	ExternalPaths  []string         `json:"external_paths"`
	ProjectRoots   []string         `json:"project_roots"`
}

type ConflictItem struct {
	Type        string `json:"type"`
	Key         string `json:"key"`
	PackagePath string `json:"package_path,omitempty"`
	TargetPath  string `json:"target_path,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type ConflictBucket struct {
	Additions  []ConflictItem `json:"additions"`
	Conflicts  []ConflictItem `json:"conflicts"`
	TargetOnly []ConflictItem `json:"target_only"`
}

type ConflictReport struct {
	Buckets map[string]ConflictBucket `json:"buckets"`
}

type DecisionAction string

const (
	DecisionOverwrite  DecisionAction = "overwrite"
	DecisionKeepTarget DecisionAction = "keep-target"
	DecisionSkip       DecisionAction = "skip"
)

type ConflictDecision struct {
	Actions map[string]DecisionAction `json:"actions"`
}

type SnapshotMeta struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	ArchivePath string    `json:"archive_path"`
	Targets     []string  `json:"targets"`
}

type PackageMeta struct {
	SchemaVersion int            `json:"schema_version"`
	Hostname      string         `json:"hostname"`
	CreatedAt     time.Time      `json:"created_at"`
	Agent         string         `json:"agent"`
	AgentVersion  string         `json:"agent_version"`
	PathScan      PathScanResult `json:"path_scan"`
	FileCount     int            `json:"file_count"`
	TotalSize     int64          `json:"total_size"`
	Links         []LinkRelation `json:"links"`
}

type ImportPreview struct {
	Meta             PackageMeta    `json:"meta"`
	PathScan         PathScanResult `json:"path_scan"`
	SuggestedMapping PathMapping    `json:"suggested_mapping"`
}

type DoctorMode string

const (
	DoctorModeExport DoctorMode = "export"
	DoctorModeImport DoctorMode = "import"
)

type DoctorStatus string

const (
	DoctorPass  DoctorStatus = "pass"
	DoctorWarn  DoctorStatus = "warn"
	DoctorBlock DoctorStatus = "block"
)

type DoctorItem struct {
	Name    string       `json:"name"`
	Status  DoctorStatus `json:"status"`
	Message string       `json:"message"`
}

type DoctorReport struct {
	Items []DoctorItem `json:"items"`
}

type CheckStatus string

const (
	CheckPass CheckStatus = "pass"
	CheckWarn CheckStatus = "warn"
)

type CheckItem struct {
	Category string      `json:"category"`
	Name     string      `json:"name"`
	Status   CheckStatus `json:"status"`
	Message  string      `json:"message"`
}

type CheckReport struct {
	Items []CheckItem `json:"items"`
}

type ImportResult struct {
	Written  []string      `json:"written"`
	Updated  []string      `json:"updated"`
	Skipped  []string      `json:"skipped"`
	LogPath  string        `json:"log_path"`
	Snapshot SnapshotMeta  `json:"snapshot"`
	Checks   []CheckItem   `json:"checks"`
	Rewrite  RewriteReport `json:"rewrite"`
}

func GroupKey(relPath string) string {
	clean := path.Clean(strings.TrimPrefix(relPath, "/"))
	if clean == "." {
		return ""
	}
	parts := strings.Split(clean, "/")
	if clean == ".claude.json" {
		return clean
	}
	if len(parts) >= 2 && parts[0] == ".claude" {
		switch parts[1] {
		case "skills", "projects", "plugins", "agents", "commands":
			if len(parts) >= 3 {
				return path.Join(parts[0], parts[1], parts[2])
			}
		default:
			return path.Join(parts[0], parts[1])
		}
	}
	return clean
}
