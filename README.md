# Mobile Dev Tools

`mdt` installs on macOS the tools needed to develop in Java and to build mobile apps with React Native and Flutter.

> `mdt`, a Go CLI and TUI, replaces the bash scripts, which are deprecated (see [legacy/](legacy/README.md)). It is still in preview. See [ROADMAP.md](ROADMAP.md).

# What does mdt install?

- [Xcode Command Line Tools](https://developer.apple.com/xcode/resources/)
- [Homebrew Package Manager](https://brew.sh/)
- [Git](https://git-scm.com/)
- [GitHub CLI](https://cli.github.com/)
- [iTerm2 Terminal Emulator](https://iterm2.com/)
- [Postman for API Development](https://www.postman.com/)
- [RapidAPI](https://paw.cloud/)
- [Visual Studio Code Editor](https://code.visualstudio.com/)
- [nvm - Node Version Manager](https://github.com/nvm-sh/nvm)
- [Node.js LTS](https://nodejs.org/en/)
- [pnpm Package Manager](https://pnpm.io/)
- [Flutter](https://flutter.dev)
- [Facebook's Watchman - A File Watching Service](https://facebook.github.io/watchman/)
- [CocoaPods - Dependency Manager for Swift and Objective-C](https://cocoapods.org/)
- [JDK 17 - Azul Zulu](https://www.azul.com/downloads/)
- [Android Studio - Android's IDE](https://developer.android.com/studio/)
- [Android SDK command line tools and components](https://developer.android.com/tools) (platform tools, emulator, one platform and build tools, via `sdkmanager`)
- [Xcode - iOS IDE](https://developer.apple.com/xcode/)
- [Fastlane - App automation](https://fastlane.tools/)
- [TypeScript - Globally (via npm)](https://www.typescriptlang.org/)
- [ngrok - Secure introspectable tunnel to localhost (via Homebrew)](https://ngrok.com/)
- [Bun](https://bun.sh/)
- [mkcert - Local HTTPS certificates](https://github.com/FiloSottile/mkcert)
- [OrbStack - Docker and Linux on macOS](https://orbstack.dev/)
- [Firefox Developer Edition](https://www.mozilla.org/firefox/developer/)
- [Brave Browser](https://brave.com/)
- [ungoogled-chromium](https://github.com/ungoogled-software/ungoogled-chromium)
- [Maven](https://maven.apache.org/)
- [IntelliJ IDEA](https://www.jetbrains.com/idea/)

These optional tools are in the `mdt` catalog but belong to no profile, so they are never preselected: Google Chrome, Zed, Warp, Cursor, Google Antigravity, and the Claude, ChatGPT and Gemini desktop apps (Gemini needs Apple silicon and macOS 15 or newer). xcodes (several Xcode versions side by side) is optional too. Pick them in the TUI or with `mdt install --only chrome,zed,claude,xcodes`.

Jest is no longer installed globally: React Native projects ship it as a project dependency.

# Install

Install the `mdt` binary (no Homebrew or Go needed):

    $ curl -fsSL https://raw.githubusercontent.com/JosePBrotons/mobile-dev-tools/master/install.sh | bash

It downloads the latest release, checks its checksum and puts `mdt` in `~/.local/bin` (set `MDT_INSTALL_DIR` to change that, `MDT_VERSION` to pick a release). Other ways:

    $ brew install josepbrotons/tap/mdt
    $ go install github.com/josepbrotons/mobile-dev-tools/cmd/mdt@latest

The Homebrew tap and the first release exist only after a version is tagged (see Contributing).

To keep in mind:

- Installing [Xcode](https://developer.apple.com/xcode/) **requires being signed in with an Apple ID already inside the App Store App.**
- Some steps use `sudo`, so you may be asked for your admin password.
- The Android SDK step asks you to accept the Android SDK licenses in the terminal.
- The web profile runs `mkcert -install`, which asks for your admin password. OrbStack is free for personal use only.
- Homebrew (`brew shellenv`) and `ANDROID_HOME` are added to `~/.zprofile`. Open a new terminal after installing so they take effect.

# Using mdt (preview)

`mdt` is the Go replacement for the scripts. It reads the same tool list, skips what is already installed, and installs only what a profile or your own selection needs. From a clone of this repo, replace `mdt` with `go run ./cmd/mdt` (needs [Go](https://go.dev/)):

    $ mdt                                       (interactive TUI)
    $ mdt version
    $ mdt list                                  (profiles and tool ids)
    $ mdt doctor                                (check installed tools and shell setup)
    $ mdt doctor --profile rn                   (also report missing rn tools)
    $ mdt install --profile rn --dry-run        (show the plan only)
    $ mdt install --profile rn                  (asks before installing)
    $ mdt install --only node,watchman --yes    (pick tools, no prompt)
    $ mdt upgrade --dry-run                     (show what is outdated)
    $ mdt upgrade --only node,gh                (upgrade just those, asks first)
    $ mdt uninstall chrome typescript --dry-run (show what would be removed)

`mdt doctor` also checks that Homebrew is loaded by `~/.zprofile` and that the mkcert local CA is trusted, and prints a fix for each problem.

`mdt upgrade` runs `brew update`, then upgrades the catalog tools that `brew outdated` lists, and moves Node.js to the latest LTS through nvm (global npm packages such as TypeScript are kept). Without `--profile` or `--only` it checks every installed catalog tool and leaves other Homebrew packages alone. Casks that update themselves (Chrome, VS Code and similar) do not show up as outdated, and pinned formulae are listed but skipped. Xcode, nvm and other script installs are not upgraded.

`mdt uninstall <id>...` removes tools through Homebrew, plus `npm uninstall -g` for TypeScript, and undoes `mkcert -install` first. It refuses what it cannot undo safely: tools that another installed tool needs, Xcode, and script installs without an automatic removal (Homebrew, Xcode Command Line Tools, nvm, Node.js). The refusal says why and where to remove them by hand. Lines added to `~/.zprofile` are not removed; the summary lists them.

Required tools are added automatically (Node.js brings nvm, CocoaPods brings Homebrew). A failed tool does not stop the rest; the summary lists it and the log is in `~/Library/Logs/mobile-dev-tools/`. 
Run `mdt` with no arguments in a terminal for the interactive mode. It checks the Mac, lets you start from a profile, toggle tools, review the exact commands, watch live progress and read a summary with next steps. Keys: `space` toggle, `a` select all, `/` filter, `enter` continue, `esc` back, `?` help, `q` quit. It asks for your admin password once before installing. The TUI has only been exercised with a fake runner and up to the review screen on a real Mac; no real install has been run through it yet.

# Legacy scripts

The bash scripts live in [legacy/](legacy/README.md). They are deprecated and frozen, and will be removed in a later release.

    $ bash legacy/mobile-dev-tools.sh -rn (React Native)
    $ bash legacy/mobile-dev-tools.sh -flutter (Flutter)
    $ bash legacy/mobile-dev-tools.sh -java (Java)
    $ bash legacy/mobile-dev-tools.sh -web (Web)

# Contributing

Conventions for humans and AI agents live in [AGENTS.md](AGENTS.md).
