# Catalog refresh checklist

The catalog pins a few versions and names a lot of Homebrew packages. Upstream moves on, so review it on a schedule: before each release, and at least once a quarter.

Make each change in `catalog/tools.yaml` and in the legacy scripts together until Phase 6 retires them. `TestProfilesMatchLegacyScripts` and `TestCommandsMatchLegacyScripts` fail when they drift.

## 1. Check every package against Homebrew

```
go test ./internal/catalog -run TestUpstream -upstream
```

This needs Homebrew and the network, so it is off by default. It fails for a formula or cask that no longer exists or that is deprecated or disabled. For each failure:

- Renamed: update `package` and the `check` line, and the helper call in each legacy script.
- Deprecated with a replacement: switch to it, as was done for `intellij-idea-ce` to `intellij-idea`.
- Disabled with no replacement: remove the tool from the catalog, the scripts and the README list.

The test does not cover `mas` tools. Xcode (`497799835`) is checked by hand: open its App Store page.

## 2. Pinned versions

| What | Where | Look at |
| --- | --- | --- |
| JDK 17 (`zulu@17`) | `catalog/tools.yaml`, `Java/`, `Flutter/`, `React Native/` scripts | The JDK that React Native and the Android Gradle Plugin require. Check the React Native "Set up your environment" page and the AGP release notes. Move all four places together. |
| nvm `v0.40.8` | `catalog/tools.yaml`, `Web/`, `React Native/` scripts | The latest release at https://github.com/nvm-sh/nvm/releases. Bump the install URL in all three places. |
| Node.js | not pinned, installs the latest LTS | Nothing to bump. `mdt upgrade` moves existing machines to a newer LTS. |

## 3. Install method for Xcode

Xcode installs with `mas`, which needs an App Store sign-in. The open question in ROADMAP.md is whether [`xcodes`](https://github.com/XcodesOrg/xcodes) is a better fit. Revisit it when `mas` or the App Store flow changes.

## 4. Notes, links and next steps

- Open each `url`. Fix the ones that redirect or 404.
- Reread `notes` and `next_steps`. Remove warnings that are no longer true (for example "Large download") and add new ones (for example a new minimum macOS).
- Confirm the profiles still match what each stack recommends.

## 5. Verify

```
go test ./...
mdt install --profile <id> --dry-run     # for each profile
mdt doctor
```

Then update `README.md` if the tool list changed.
