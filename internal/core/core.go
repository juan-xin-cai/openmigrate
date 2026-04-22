package core

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/openmigrate/openmigrate/internal/core/conflict"
	"github.com/openmigrate/openmigrate/internal/core/doctor"
	omlog "github.com/openmigrate/openmigrate/internal/core/log"
	"github.com/openmigrate/openmigrate/internal/core/manifest"
	"github.com/openmigrate/openmigrate/internal/core/pack"
	"github.com/openmigrate/openmigrate/internal/core/pathscan"
	"github.com/openmigrate/openmigrate/internal/core/postcheck"
	"github.com/openmigrate/openmigrate/internal/core/rewrite"
	"github.com/openmigrate/openmigrate/internal/core/snapshot"
	"github.com/openmigrate/openmigrate/internal/core/symlink"
	"github.com/openmigrate/openmigrate/internal/core/types"
	"github.com/openmigrate/openmigrate/internal/core/whitelist"
	"github.com/openmigrate/openmigrate/internal/core/writer"
)

type ExportParams struct {
	SourceHome string
	Agent      string
	Version    string
	OutputDir  string
	Passphrase string
	Verbose    io.Writer
}

type ExportResult struct {
	PackagePath string
	MetaPath    string
	PathScan    types.PathScanResult
	LogPath     string
}

type ImportParams struct {
	PackagePath string
	Passphrase  string
	TargetHome  string
	Mapping     types.PathMapping
	Verbose     io.Writer
}

type ImportPreviewParams struct {
	PackagePath string
	Passphrase  string
	TargetHome  string
	Verbose     io.Writer
}

type ImportApplyParams struct {
	PackagePath string
	Passphrase  string
	Mapping     types.PathMapping
	Decisions   types.ConflictDecision
	Verbose     io.Writer
}

type DoctorParams struct {
	Mode                types.DoctorMode
	ExpectedPackageSize int64
	AbortOnSkew         bool
	PackageAgentVersion string
	Verbose             io.Writer
}

type RollbackParams struct {
	Snapshot   types.SnapshotMeta
	SnapshotID string
	Passphrase string
	Verbose    io.Writer
}

func Export(ctx context.Context, params ExportParams) (ExportResult, error) {
	if err := ctx.Err(); err != nil {
		return ExportResult{}, err
	}
	logger, err := omlog.New(params.Verbose)
	if err != nil {
		return ExportResult{}, err
	}
	defer logger.Close()

	sourceHome := params.SourceHome
	if sourceHome == "" {
		sourceHome, err = os.UserHomeDir()
		if err != nil {
			return ExportResult{}, err
		}
	}
	cfg, err := whitelist.Load(params.Agent, params.Version)
	if err != nil {
		return ExportResult{}, err
	}
	manifestResult, err := manifest.Build(sourceHome, cfg)
	if err != nil {
		return ExportResult{}, err
	}
	manifestResult, err = symlink.Resolve(manifestResult, sourceHome, logger)
	if err != nil {
		return ExportResult{}, err
	}
	scanResult, err := pathscan.Scan(manifestResult)
	if err != nil {
		return ExportResult{}, err
	}
	if err := os.MkdirAll(params.OutputDir, 0o755); err != nil {
		return ExportResult{}, err
	}
	stamp := time.Now().Format("20060102-150405")
	packagePath := filepath.Join(params.OutputDir, stamp+".ommigrate")
	metaPath := filepath.Join(params.OutputDir, stamp+".meta.json")
	host, _ := os.Hostname()
	meta := types.PackageMeta{
		SchemaVersion: 1,
		Hostname:      host,
		CreatedAt:     time.Now(),
		Agent:         params.Agent,
		AgentVersion:  params.Version,
		PathScan:      scanResult,
		FileCount:     len(manifestResult.Entries),
		TotalSize:     manifestResult.TotalSize,
		Links:         manifestResult.Links,
	}
	if err := pack.CreatePackage(manifestResult, meta, packagePath, metaPath, params.Passphrase); err != nil {
		return ExportResult{}, err
	}
	return ExportResult{PackagePath: packagePath, MetaPath: metaPath, PathScan: scanResult, LogPath: logger.Path()}, nil
}

func Import(ctx context.Context, params ImportParams) (types.ConflictReport, error) {
	if err := ctx.Err(); err != nil {
		return types.ConflictReport{}, err
	}
	logger, err := omlog.New(params.Verbose)
	if err != nil {
		return types.ConflictReport{}, err
	}
	defer logger.Close()
	root, meta, err := pack.UnpackPackage(params.PackagePath, params.Passphrase)
	if err != nil {
		return types.ConflictReport{}, err
	}
	defer os.RemoveAll(filepath.Dir(root))
	mapping, err := normalizeImportMapping(params.Mapping, params.TargetHome, meta.PathScan)
	if err != nil {
		return types.ConflictReport{}, err
	}
	if _, err := rewrite.RewriteTree(root, mapping, meta.PathScan, logger); err != nil {
		return types.ConflictReport{}, err
	}
	return conflict.Detect(root, mapping.TargetHome)
}

func PreviewImport(ctx context.Context, params ImportPreviewParams) (types.ImportPreview, error) {
	if err := ctx.Err(); err != nil {
		return types.ImportPreview{}, err
	}
	logger, err := omlog.New(params.Verbose)
	if err != nil {
		return types.ImportPreview{}, err
	}
	defer logger.Close()
	root, meta, err := pack.UnpackPackage(params.PackagePath, params.Passphrase)
	if err != nil {
		return types.ImportPreview{}, err
	}
	defer os.RemoveAll(filepath.Dir(root))

	targetHome := params.TargetHome
	if targetHome == "" {
		targetHome, err = os.UserHomeDir()
		if err != nil {
			return types.ImportPreview{}, err
		}
	}
	suggested := buildSuggestedMapping(meta.PathScan, targetHome)
	return types.ImportPreview{
		Meta:             meta,
		PathScan:         meta.PathScan,
		SuggestedMapping: suggested,
	}, nil
}

func ApplyImport(ctx context.Context, params ImportApplyParams) (types.ImportResult, error) {
	if err := ctx.Err(); err != nil {
		return types.ImportResult{}, err
	}
	logger, err := omlog.New(params.Verbose)
	if err != nil {
		return types.ImportResult{}, err
	}
	defer logger.Close()

	root, meta, err := pack.UnpackPackage(params.PackagePath, params.Passphrase)
	if err != nil {
		return types.ImportResult{}, err
	}
	defer os.RemoveAll(filepath.Dir(root))

	mapping, err := normalizeImportMapping(params.Mapping, "", meta.PathScan)
	if err != nil {
		return types.ImportResult{}, err
	}
	rewriteReport, err := rewrite.RewriteTree(root, mapping, meta.PathScan, logger)
	if err != nil {
		return types.ImportResult{}, err
	}
	files, err := writer.CollectFiles(root)
	if err != nil {
		return types.ImportResult{}, err
	}
	conflictReport, err := conflict.Detect(root, mapping.TargetHome)
	if err != nil {
		return types.ImportResult{}, err
	}
	targets := affectedTargets(files, mapping.TargetHome, params.Decisions)
	snapshotMeta, err := snapshot.CreateSnapshot(targets, params.Passphrase, logger)
	if err != nil {
		return types.ImportResult{}, err
	}
	written, updated, skipped, err := writer.Write(files, mapping.TargetHome, params.Decisions, conflictReport, logger)
	if err != nil {
		return types.ImportResult{}, err
	}
	links := rewrite.RewriteLinkRelations(meta.Links, mapping)
	if err := symlink.Restore(mapping.TargetHome, links, logger); err != nil {
		return types.ImportResult{}, err
	}
	checks, err := postcheck.Check(mapping.TargetHome, rewriteReport)
	if err != nil {
		return types.ImportResult{}, err
	}
	return types.ImportResult{
		Written:  written,
		Updated:  updated,
		Skipped:  skipped,
		LogPath:  logger.Path(),
		Snapshot: snapshotMeta,
		Checks:   checks.Items,
		Rewrite:  rewriteReport,
	}, nil
}

func Doctor(ctx context.Context, params DoctorParams) (types.DoctorReport, error) {
	if err := ctx.Err(); err != nil {
		return types.DoctorReport{}, err
	}
	logger, err := omlog.New(params.Verbose)
	if err != nil {
		return types.DoctorReport{}, err
	}
	defer logger.Close()
	return doctor.Run(doctor.Params{
		Mode:                params.Mode,
		ExpectedPackageSize: params.ExpectedPackageSize,
		AbortOnSkew:         params.AbortOnSkew,
		PackageAgentVersion: params.PackageAgentVersion,
	}, logger)
}

func Rollback(ctx context.Context, params RollbackParams) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	logger, err := omlog.New(params.Verbose)
	if err != nil {
		return err
	}
	defer logger.Close()
	if params.Snapshot.ArchivePath == "" {
		meta, err := snapshot.ResolveSnapshot(params.SnapshotID)
		if err != nil {
			return err
		}
		params.Snapshot = meta
	}
	return snapshot.Rollback(params.Snapshot, params.Passphrase, logger)
}

func affectedTargets(files []types.FileEntry, targetHome string, decisions types.ConflictDecision) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, 16)
	for _, file := range files {
		group := file.GroupKey
		action := decisions.Actions[group]
		if action == types.DecisionKeepTarget || action == types.DecisionSkip {
			continue
		}
		target := filepath.Join(targetHome, filepath.FromSlash(group))
		if _, ok := seen[target]; ok {
			continue
		}
		seen[target] = struct{}{}
		out = append(out, target)
	}
	return out
}

func normalizeImportMapping(mapping types.PathMapping, fallbackTargetHome string, scan types.PathScanResult) (types.PathMapping, error) {
	normalized := mapping
	if normalized.SourceHome == "" {
		normalized.SourceHome = scan.HomePrefix
	}
	if normalized.TargetHome == "" {
		normalized.TargetHome = fallbackTargetHome
	}
	if normalized.TargetHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return types.PathMapping{}, err
		}
		normalized.TargetHome = home
	}
	if len(scan.ProjectRoots) > 0 && len(normalized.ProjectMappings) == 0 {
		return types.PathMapping{}, types.ErrPathMappingRequired
	}
	if len(normalized.ExternalPaths) == 0 {
		normalized.ExternalPaths = append([]string(nil), scan.ExternalPaths...)
	}
	return normalized, nil
}

func buildSuggestedMapping(scan types.PathScanResult, targetHome string) types.PathMapping {
	suggested := types.PathMapping{
		SourceHome:    scan.HomePrefix,
		TargetHome:    targetHome,
		ExternalPaths: append([]string(nil), scan.ExternalPaths...),
	}
	for _, projectRoot := range scan.ProjectRoots {
		suggested.ProjectMappings = append(suggested.ProjectMappings, types.PathPair{
			From: projectRoot,
			To:   "",
		})
	}
	return suggested
}
