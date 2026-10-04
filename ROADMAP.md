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

## Phase 4: Updates and maintenance

- `mdt doctor` (done): `mdt doctor [--profile <id> | --only a,b]` runs each tool's check and verifies the profile edits (Homebrew shellenv and nvm loading, `ANDROID_HOME` lines and SDK folder), printing a fix for each problem. Exit 1 when problems are found. Deferred: the mkcert hook is not verified. Found a gap: nothing writes `brew shellenv` to `~/.zprofile`, so a fresh Mac may not find `brew` in new terminals; fix it with a `post_install` hook later.
- `mdt upgrade`: wrap `brew outdated` / `brew upgrade` for catalog tools and bump Node to the latest LTS via nvm.
- Optional `mdt uninstall <id>`.
- A recurring catalog refresh checklist: JDK level required by React Native and the Android Gradle Plugin, latest nvm release, deprecated casks, Xcode install method.

## Phase 5: Distribution and CI

- GoReleaser builds a universal darwin binary on each tag.
- Homebrew tap: `brew install josepbrotons/tap/mdt`.
- `install.sh` bootstrap that downloads the right binary on a Mac without Homebrew.
- GitHub Actions: `go test ./...`, `golangci-lint`, `shellcheck` on the legacy scripts, and a `macos-latest` job running `mdt install --dry-run` for every profile.

## Phase 6: Retire the legacy scripts

Once Phase 2 reaches parity, move the scripts to `legacy/` with a deprecation note in the README, then remove them in a later release.

## Open questions

- Install Xcode with `mas` (needs App Store sign-in) or [`xcodes`](https://github.com/XcodesOrg/xcodes) (supports multiple versions)?
- Should Ruby be managed (rbenv) for CocoaPods and Fastlane, or keep the Homebrew formulas?
- Install Android SDK components (platforms, build tools, emulator images) with `sdkmanager` instead of relying on Android Studio's first-run wizard?
