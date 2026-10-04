# AGENTS.md

Conventions for anyone (human or AI agent) changing this repo. `CLAUDE.md` imports this file.

## Project overview

Installs mobile and Java development tools on macOS. `mdt` is a Go CLI and Bubble Tea TUI that installs from the catalog. The bash scripts in `legacy/` are deprecated and frozen. Read [ROADMAP.md](ROADMAP.md) before starting larger work and keep it updated when a phase changes.

## Repo map

```
legacy/                             deprecated bash scripts, frozen (see legacy/README.md)
legacy/mobile-dev-tools.sh          entry point; cds to its own folder and sources a profile
legacy/Misc/brew-helpers.sh         brew_formula / brew_cask helpers, one brew update
install.sh                          downloads the latest mdt release, verifies its checksum
.goreleaser.yaml                    universal darwin build, archive, checksums, Homebrew cask
.github/workflows/                  ci.yml (tests, lint, shellcheck, macOS dry runs), release.yml (on v* tags)
go.mod                              Go module
catalog/tools.yaml, catalog/embed.go  tool catalog and its go:embed
internal/catalog/                   loader and validator for the catalog
internal/runner/                    the only package that executes commands; Fake for tests
internal/shellenv/                  prelude for every step, idempotent ~/.zprofile edits
internal/detect/                    runs each tool's check
internal/doctor/                    read-only environment checks for mdt doctor
internal/plan/                      resolve, dry-run output, execute with progress events
internal/upgrade/                   finds outdated brew tools and a newer Node LTS, runs the upgrades
internal/uninstall/                 removal plan with refusals, runs the uninstalls
internal/sysinfo/                   macOS version, arch, free disk for the welcome screen
internal/tui/                       Bubble Tea TUI (no install logic)
cmd/mdt/                            CLI: mdt (TUI), mdt install, mdt doctor, mdt upgrade, mdt uninstall, mdt list
docs/catalog-refresh.md             checklist for reviewing the catalog
ROADMAP.md                          plan for the TUI
```

## Shell conventions

These apply to `install.sh` and, if a critical fix is ever needed, to `legacy/`. The legacy scripts are frozen: do not add tools to them.

- Every script starts with `#!/usr/bin/env bash`. Invoke with `bash`, not `sh`.
- Legacy profile and `Misc/` scripts are `source`d from `legacy/`. Use paths relative to it (`./Misc/...`) and never `cd` inside a sourced script.
- Use `$HOME/...` for user files, never `~` inside quotes and never a relative path into the home folder.
- Every step must be idempotent: check before installing. Use `brew_formula` / `brew_cask` from `Misc/brew-helpers.sh`; do not call `brew install` or `brew update` directly in profiles.
- Profile edits (`~/.zprofile`) must check for an existing line before appending.
- Echo a short "Installing X..." line before each step, matching existing scripts.
- Pin downloaded installer versions (for example the nvm install URL) and bump them deliberately.

## Go conventions

- Format with `gofmt`, lint with `golangci-lint`, test with `go test ./...`.
- Only `internal/runner` executes commands. Everything else takes a `Runner` interface so tests can use a fake.
- New tools are added to `catalog/tools.yaml`, not hardcoded in Go. `go:embed` cannot reach parent folders, so the embed lives in `catalog/embed.go`.
- A script tool that asks the user questions (license prompts, sign-ins) sets `interactive: true`. It runs last, with the terminal attached, and no tool may require it.
- A new `post_install` hook goes in `catalog.KnownPostInstall` and in the `hooks` table in `internal/plan`. Give it an `undo` command if `mdt uninstall` should reverse it.
- A script tool that `mdt uninstall` can remove needs an `uninstall` list in the catalog. Without it, the tool is refused.
- The dry-run golden file lives in `internal/plan/testdata/`. After an intended change, run `go test ./internal/plan -update` and review the diff.
- Prefer table-driven tests. Tests must never touch the real home folder; use `t.TempDir()`.
- TUI code (`internal/tui`) holds no install logic; it calls `internal/plan` and `internal/runner`.

## Adding or updating a tool

1. Confirm the exact Homebrew formula or cask name upstream and check it is not deprecated (`deprecate!` / `disable!` in the cask file).
2. Add one entry to `catalog/tools.yaml`. Do not touch `legacy/`.
3. Update the tool list in `README.md`.
4. Run the checks below.

To review the whole catalog for stale pins and deprecated packages, follow [docs/catalog-refresh.md](docs/catalog-refresh.md).

## Verification

Agent sandboxes and CI runners are usually Linux, so install steps cannot run there. Never execute them outside a real Mac. Instead:

```
for f in install.sh legacy/mobile-dev-tools.sh legacy/*/*.sh; do bash -n "$f"; done
shellcheck -x -e SC1091 install.sh legacy/*.sh legacy/*/*.sh
```

Both must pass with no output. For Go code, also run `go test ./...` and `mdt install --dry-run --profile <x>`. Say in your summary that scripts were statically checked only, unless they were run on macOS.

## Releasing

CI (`.github/workflows/ci.yml`) runs on every PR. To release, tag master and push the tag: `git tag vX.Y.Z && git push origin vX.Y.Z`. `release.yml` runs GoReleaser, which publishes the GitHub release (universal darwin binary, `checksums.txt`) that `install.sh` downloads, and updates the Homebrew cask in `JosePBrotons/homebrew-tap` when the `HOMEBREW_TAP_GITHUB_TOKEN` secret is set (a fine-grained token with Contents read and write on that repo; without it the tap is skipped). Check a config change locally with `goreleaser release --snapshot --clean`.

## Commits and branches

- Short imperative subject line ("Fix xcode-select flag"), one logical change per commit.
- Update `README.md` whenever the set of installed tools or the usage changes.
- Do not commit secrets, machine-specific paths or generated binaries.

## Writing style

- Plain, short sentences. No em dashes in docs, comments or commit messages.
