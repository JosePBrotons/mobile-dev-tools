# Roadmap: from install scripts to a TUI

## Vision

One binary, `mdt`, that you run on a fresh Mac. It checks the machine, shows what is already installed, lets you start from a profile (React Native, Flutter, Java, Web or Custom), and then toggle individual tools on and off. You review the plan, confirm, and watch each tool install with live progress. It ends with a summary and next steps (open a new terminal, sign in to the App Store, accept Android SDK licenses).

The same engine runs without the UI for automation:

    mdt install --profile rn --yes
    mdt install --only node,watchman,cocoapods --dry-run

**Stack:** Go + [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles) and [Lip Gloss](https://github.com/charmbracelet/lipgloss). Go produces a single static binary with no runtime to install first, which matters because the target machine has nothing on it yet.

## Phase 0: Stabilize the current scripts (done)

- Fixed `xcode-select --install` (the flag contained a non-ASCII dash and never worked).
- Homebrew, Xcode CLT and every package are now detected before installing, so reruns are safe.
- One `brew update` per run instead of one per package (`Misc/brew-helpers.sh`).
- `set-android-home-path.sh` no longer changes the working directory and uses `cmdline-tools/latest/bin` instead of the removed `tools` folders.
- Yarn 4 has no `yarn global`: TypeScript is installed with npm, ngrok with its Homebrew cask, global Jest dropped. Global modules now run as part of the React Native profile.
- Version refresh: nvm `v0.40.8`; `intellij-idea-ce` (discontinued) replaced by `intellij-idea`; JDK stays on Zulu 17 as React Native still recommends it.

## Phase 1: Declarative tool catalog (done)

Shipped: `catalog/tools.yaml` (embedded through `catalog/embed.go`) and `internal/catalog` with strict loading, validation (unknown ids, cycles, missing packages) and a test that each profile matches the legacy scripts. Schema additions: `install` (shell lines for `script` tools), `url`, and top-level `categories` and `profiles` lists. A Web profile (pnpm, Bun, mkcert, OrbStack, extra browsers) was added to the catalog and to `Web/web-dev-tools.sh`, and pnpm replaced Yarn.

Move "what to install" out of scripts and into data: `catalog/tools.yaml`, embedded in the binary with `go:embed`.

```yaml
- id: cocoapods
  name: CocoaPods
  description: Dependency manager for iOS projects
  category: mobile-sdks
  profiles: [rn, flutter]
  method: brew            # brew | cask | mas | script
  package: cocoapods
  check: command -v pod
  requires: [homebrew]
  notes: []
- id: android-studio
  method: cask
  package: android-studio
  post_install: [android-home-env]
```

- Categories: system, terminals and editors, runtimes, mobile SDKs, build and release, CLIs.
- Profiles are only preselections; the user can always add or remove tools.
- `requires` drives ordering and auto-selection (Node requires nvm, CocoaPods requires Homebrew).
- `post_install` hooks cover environment changes such as `ANDROID_HOME`.
- `notes` surface warnings in the UI (needs sudo, needs Apple ID, large download).

## Phase 2: Go core, no UI yet (done)

Shipped: `mdt install (--profile <id> | --only a,b) [--dry-run] [--yes]` and `mdt list`. `internal/runner` (bash runner plus a fake), `internal/shellenv` (a prelude that loads Homebrew and nvm in every step, idempotent `~/.zprofile` edits), `internal/detect` (runs each tool's `check`) and `internal/plan` (dependency resolution, dry-run table, execution with progress events for the TUI). Dependencies that no pending tool needs are dropped, so Node.js installed outside nvm does not pull in nvm. Logs go to `~/Library/Logs/mobile-dev-tools/`. Real installs refuse to run outside macOS. `TestCommandsMatchLegacyScripts` checks parity: every package the legacy scripts install, and every script line, comes from the catalog.

Deferred: version detection (the catalog has no version command yet) and the `.bash_profile` fallback for macOS older than 10.15. Not yet run on a real Mac.

Original plan:

```
cmd/mdt/            entry point, flag parsing
internal/catalog/   load and validate tools.yaml
internal/detect/    is a tool installed? which version?
internal/plan/      resolve dependencies, order steps, dry-run output
internal/runner/    the only place that executes commands (interface + fake for tests)
internal/shellenv/  idempotent edits to ~/.zprofile
```

- Flags: `--profile`, `--only`, `--dry-run`, `--yes`.
- Logs written to `~/Library/Logs/mobile-dev-tools/`.
- A failed tool does not stop the rest; it is reported at the end.
- Exit criteria: `mdt install --profile <x>` reaches parity with each legacy script.

## Phase 3: Bubble Tea TUI (done)

Shipped: `mdt` with no arguments (or `mdt tui`) opens the TUI when stdin and stdout are terminals. `internal/tui` has all six screens below on Bubble Tea v1. Engine changes it needed: `runner.Exec` takes an optional `Stdin` (nil in the TUI so it keeps the terminal) and `runner.SudoValidate` caches the admin password through `tea.ExecProcess`; `plan.Build` splits resolution from detection, `plan.Rows` and `plan.Notes` feed the review screen, `plan.NextSteps` and the catalog `next_steps` field feed the summary; `internal/sysinfo` powers the welcome screen; detection runs eight checks at a time. On other systems than macOS the TUI works as a preview and refuses to install.

Deferred: no real install has been run through the TUI yet (tests use the fake runner, and a real Mac run stopped at the review screen). Cancelling mid-install kills the running step and the rest fail fast, it does not stop cleanly between tools.

Original plan:

Screens, in order:

1. **Welcome / system check**: macOS version, architecture, Xcode CLT, Homebrew, disk space.
2. **Profile picker**: React Native, Flutter, Java, Web, Custom.
3. **Tool checklist** grouped by category. Installed tools are marked and unchecked by default; dependencies are auto-checked with a reason ("required by CocoaPods").
4. **Review**: exact commands to run, plus sudo and Apple ID warnings.
5. **Progress**: per-tool spinner, scrollable live log pane.
6. **Summary**: installed, skipped, failed (with log path) and next steps.

Keys: `space` toggle, `a` select all, `/` filter, `enter` continue, `esc` back, `?` help.

## Phase 4: Updates and maintenance (done)

- `mdt doctor` (done): `mdt doctor [--profile <id> | --only a,b]` runs each tool's check and verifies the profile edits (Homebrew shellenv and nvm loading, `ANDROID_HOME` lines and SDK folder, mkcert CA trust), printing a fix for each problem. Exit 1 when problems are found. The gap it found is closed: a `brew-shellenv` `post_install` hook on the Homebrew tool writes one arch-agnostic line to `~/.zprofile` (skipped when a `brew shellenv` line is already there), in `mdt install` and in `Misc/install-brew.sh`.
- `mdt upgrade` (done): `mdt upgrade [--profile <id> | --only a,b] [--dry-run] [--yes]` runs `brew update` and `brew outdated --json=v2`, matches the result to catalog packages, and upgrades them; Node.js moves to the latest LTS through nvm with `--reinstall-packages-from=default`. Pinned formulae are listed and skipped. Deferred: Xcode (mas), nvm itself and other script tools are not upgraded, and `--greedy` casks are not included because those apps update themselves.
- `mdt uninstall` (done): `mdt uninstall <id>... [--dry-run] [--yes]` removes brew and cask tools, plus script tools that have an `uninstall` list in the catalog (TypeScript). It refuses tools that an installed tool still requires, mas tools and script tools without an uninstall, and says why. A hook can have an `undo` (`mkcert -uninstall`). Profile lines are not edited; the summary lists them. Deferred: removing nvm, Node.js and the Android SDK folder.
- Catalog refresh checklist (done): [docs/catalog-refresh.md](docs/catalog-refresh.md), with `go test ./internal/catalog -run TestUpstream -upstream` to check every package against Homebrew for missing, deprecated or disabled entries.

None of these have been run for real on a Mac beyond dry runs: `upgrade --dry-run`, `uninstall --dry-run` and the upstream test.

## Phase 5: Distribution and CI (done)

Shipped: `.goreleaser.yaml` builds a universal darwin binary on each `v*` tag (`.github/workflows/release.yml`), with `checksums.txt` and a Homebrew cask pushed to `JosePBrotons/homebrew-tap`. `install.sh` downloads the latest release, verifies the checksum and installs to `~/.local/bin`. `mdt version` reports the release. CI (`.github/workflows/ci.yml`) runs gofmt, `go vet`, `go test`, `golangci-lint`, shellcheck, and a `macos-latest` job that dry-runs every profile plus `upgrade` and `uninstall`.

Before the first release: create the `JosePBrotons/homebrew-tap` repo and add the `HOMEBREW_TAP_GITHUB_TOKEN` secret (see AGENTS.md, Releasing). The binary is not signed or notarized; the cask clears the quarantine flag, and `install.sh` downloads with curl, which does not set it. Not yet run for real: no release has been cut, and only GoReleaser snapshots and `install.sh` against a local copy were tried.

## Phase 6: Retire the legacy scripts

Step 1 (done): the scripts moved to `legacy/` with a deprecation notice, and the parity tests were removed, so the catalog no longer has to match them. Step 2: delete `legacy/` in a later release, once `mdt` has run on a real Mac.

## Decisions

- **Xcode:** `mas` stays in the rn and flutter profiles. `xcodes` is an optional tool for people who need several versions; its sign-in prompts do not fit the TUI well.
- **Ruby:** CocoaPods and Fastlane stay on their Homebrew formulas. No rbenv; projects that pin a Ruby use bundler.
- **Android SDK:** the `android-sdk` tool installs components with `sdkmanager` after the user accepts the licenses. It is the first `interactive` tool: it runs last with the terminal attached (`tea.ExecProcess` in the TUI). Platform `android-36` and build tools `36.0.0` are pinned; Gradle downloads anything else a project needs. Removing the SDK folder is not part of `mdt uninstall`.
- **Homebrew tap:** added next to `install.sh`. A fresh Mac has no Homebrew, so `install.sh` stays the main path; the tap serves people who already have it and gives them `brew upgrade mdt`.
