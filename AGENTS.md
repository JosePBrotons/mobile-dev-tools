# AGENTS.md

Conventions for anyone (human or AI agent) changing this repo. `CLAUDE.md` imports this file.

## Project overview

Installs mobile and Java development tools on macOS. Today it is a set of bash scripts driven by `mobile-dev-tools.sh -rn|-flutter|-java`. The plan is a Go + Bubble Tea TUI (`mdt`) where the user picks what to install. Read [ROADMAP.md](ROADMAP.md) before starting larger work and keep it updated when a phase changes.

## Repo map

```
mobile-dev-tools.sh                 entry point; cds to repo root and sources a profile
React Native/react-native-dev-tools.sh
React Native/mobile-global-modules.sh   npm globals + ngrok, sourced by the RN profile
Flutter/flutter-dev-tools.sh
Java/java-dev-tools.sh
Web/web-dev-tools.sh
Misc/install-command-line-tools.sh  Xcode CLT (skips if present)
Misc/install-brew.sh                Homebrew (skips if present, loads shellenv)
Misc/brew-helpers.sh                brew_formula / brew_cask helpers, one brew update
Misc/set-android-home-path.sh       ANDROID_HOME in ~/.zprofile (idempotent)
ROADMAP.md                          plan for the TUI
```

Planned Go layout (Phase 2+): `cmd/mdt/`, `internal/{catalog,detect,plan,runner,shellenv,tui}/`, `catalog/tools.yaml`.

## Shell conventions

- Every script starts with `#!/usr/bin/env bash`. Invoke with `bash`, not `sh`.
- Profile and `Misc/` scripts are `source`d from the repo root. Use paths relative to the root (`./Misc/...`) and never `cd` inside a sourced script.
- Use `$HOME/...` for user files, never `~` inside quotes and never a relative path into the home folder.
- Every step must be idempotent: check before installing. Use `brew_formula` / `brew_cask` from `Misc/brew-helpers.sh`; do not call `brew install` or `brew update` directly in profiles.
- Profile edits (`~/.zprofile`) must check for an existing line before appending.
- Echo a short "Installing X..." line before each step, matching existing scripts.
- Pin downloaded installer versions (for example the nvm install URL) and bump them deliberately.

## Go conventions (Phase 1+)

- Format with `gofmt`, lint with `golangci-lint`, test with `go test ./...`.
- Only `internal/runner` executes commands. Everything else takes a `Runner` interface so tests can use a fake.
- New tools are added to `catalog/tools.yaml`, not hardcoded in Go.
- Prefer table-driven tests. Tests must never touch the real home folder; use `t.TempDir()`.
- TUI code (`internal/tui`) holds no install logic; it calls `internal/plan` and `internal/runner`.

## Adding or updating a tool

1. Confirm the exact Homebrew formula or cask name upstream and check it is not deprecated (`deprecate!` / `disable!` in the cask file).
2. Add it to every relevant profile script using the helpers (later: one entry in `catalog/tools.yaml`).
3. Update the tool list in `README.md`.
4. Run the checks below.

## Verification

Agent sandboxes and CI runners are usually Linux, so the install scripts cannot run there. Never execute them outside a real Mac. Instead:

```
for f in mobile-dev-tools.sh */*.sh; do bash -n "$f"; done
shellcheck -x -e SC1091 mobile-dev-tools.sh Misc/*.sh Java/*.sh Flutter/*.sh Web/*.sh "React Native"/*.sh
```

Both must pass with no output. For Go code, also run `go test ./...` and `mdt install --dry-run --profile <x>`. Say in your summary that scripts were statically checked only, unless they were run on macOS.

## Commits and branches

- Short imperative subject line ("Fix xcode-select flag"), one logical change per commit.
- Update `README.md` whenever the set of installed tools or the usage changes.
- Do not commit secrets, machine-specific paths or generated binaries.

## Writing style

- Plain, short sentences. No em dashes in docs, comments or commit messages.
