# OpenMigrate

[![CI](https://github.com/juan-xin-cai/openmigrate/actions/workflows/ci.yml/badge.svg)](https://github.com/juan-xin-cai/openmigrate/actions/workflows/ci.yml)
[![Release Please](https://github.com/juan-xin-cai/openmigrate/actions/workflows/release-please.yml/badge.svg)](https://github.com/juan-xin-cai/openmigrate/actions/workflows/release-please.yml)

Portable, encrypted migration packages for Claude Code on macOS.

OpenMigrate exports a curated subset of Claude Code and companion Claude Desktop data into an encrypted `.ommigrate` package, previews what is inside, rewrites host-specific paths during import, checks Desktop account compatibility, and lets you roll back the last import if needed.

[中文说明](./README_CN.md)

## Why OpenMigrate

- Move your Claude Code setup to another Mac without copying your full home directory.
- Keep migration packages inspectable through a sidecar metadata file.
- Strip sensitive fields from exported JSON before packaging.
- Catch risky imports early with environment checks, conflict review, and Desktop account verification.
- Recover from a bad import with an encrypted snapshot and a one-command rollback.

## What It Covers

OpenMigrate currently targets Claude Code `v2` on macOS and, when present, companion Claude Desktop `v1` data that matters for continuity.

Included data:

- Claude Code settings, project history, project data, skills, plugins, agents, and commands.
- Claude Desktop MCP/configuration files and selected settings.
- Claude Desktop session-related data that can be migrated safely.
- Git worktree metadata and path-bearing text files that need rewriting on the target machine.

Excluded or sanitized data:

- Claude Code raw session transcripts under `.claude/sessions/**`.
- Claude Desktop `.audit-key` files.
- OAuth and token-like fields in supported JSON files.
- Device- or host-specific values that should not be copied verbatim.
- Claude Desktop runtime bundles and VM directories.

## How It Works

1. `openmigrate export` scans supported files, strips sensitive fields, and writes:
   - an encrypted `*.ommigrate` archive
   - a readable `*.meta.json` metadata file
2. `openmigrate inspect` previews package metadata before import.
3. `openmigrate doctor` checks the source or target environment.
4. `openmigrate import` suggests path mappings, detects conflicts, rewrites absolute paths, and writes files into the target home.
5. OpenMigrate creates an encrypted snapshot before import so `openmigrate rollback` can restore the previous state.

## Command Overview

| Command | Purpose |
| --- | --- |
| `openmigrate doctor [package-or-meta]` | Check the current environment before export or import |
| `openmigrate export` | Export Claude Code data into an encrypted package |
| `openmigrate inspect <package.ommigrate>` | Preview package metadata |
| `openmigrate import <package.ommigrate>` | Import a package with path mapping and conflict handling |
| `openmigrate rollback --snapshot latest` | Restore the latest import snapshot |

Useful flags:

- `export --only settings,projects,skills`
- `export --exclude sessions`
- `export --no-history`
- `import --yes`
- `import --skip-desktop-session-check`
- `--verbose` on any command for extra logs

## Installation

### Build from source

```bash
git clone https://github.com/juan-xin-cai/openmigrate.git
cd openmigrate
CGO_ENABLED=0 go build -o openmigrate ./cmd/openmigrate
```

Requirements:

- macOS
- Go 1.16+
- Claude Code installed and available as `claude`

### Release artifacts

The repository includes CI plus a release workflow that builds a universal macOS binary archive and checksums. Once tagged releases are published, download them from [GitHub Releases](https://github.com/juan-xin-cai/openmigrate/releases).

## Quick Start

### 1. Export from the source machine

```bash
./openmigrate export --out ~/Desktop/openmigrate
```

You will be prompted for a passphrase. OpenMigrate writes both the package and its metadata file.

For unattended flows, set the passphrase in the environment:

```bash
export OPENMIGRATE_PASSPHRASE='your-passphrase'
./openmigrate export --out ~/Desktop/openmigrate
```

### 2. Inspect the package

```bash
./openmigrate inspect ~/Desktop/openmigrate/openmigrate.ommigrate
```

### 3. Check the target machine

```bash
./openmigrate doctor ~/Desktop/openmigrate/openmigrate.ommigrate
```

If the package includes Claude Desktop data, fix any Full Disk Access or account issues before importing.

### 4. Import on the target machine

```bash
./openmigrate import ~/Desktop/openmigrate/openmigrate.ommigrate
```

By default, OpenMigrate:

- suggests a target-home mapping
- lets you review conflicts interactively
- checks Claude Desktop account compatibility when Desktop sessions are present
- creates a rollback snapshot before writing

### 5. Roll back if needed

```bash
./openmigrate rollback --snapshot latest
```

## Safety Model

- Export packages are encrypted with your passphrase.
- Sensitive JSON fields are stripped before packaging instead of after import.
- Import does not blindly overwrite everything; it previews conflicts first.
- Desktop session imports are blocked on account mismatch unless you explicitly skip that check.
- A snapshot is created before import so the last write set can be restored.

## Project Status

OpenMigrate is in an early public stage. The current implementation is focused on:

- macOS
- Claude Code `v2`
- companion Claude Desktop `v1` data needed for migration continuity

Cross-platform support and broader agent coverage are not part of the current release scope yet.

## Development

Run the test suite with:

```bash
CGO_ENABLED=0 go test -count=1 ./...
```

The repository also ships with:

- GitHub Actions CI on pushes and pull requests to `main`
- `release-please` automation for changelog and release flow
- a macOS packaging script at [`scripts/release-macos.sh`](./scripts/release-macos.sh)

## Contributing

Issues and pull requests are welcome, especially around:

- migration safety
- path rewrite coverage
- Claude Desktop compatibility checks
- packaging and release polish

If you plan to change the package schema or supported data scope, please open an issue first so we can agree on migration guarantees.
