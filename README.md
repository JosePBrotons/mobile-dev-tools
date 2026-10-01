# Mobile Dev Tools

Scripts that install on macOS the tools needed to develop in Java and to build mobile apps with React Native and Flutter.

> A Go CLI and TUI, `mdt`, is in preview (see below). See [ROADMAP.md](ROADMAP.md).

# What does this script install?

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

These optional tools are in the `mdt` catalog but belong to no profile, so they are never preselected: Google Chrome, Zed, Warp, Cursor, Google Antigravity, and the Claude, ChatGPT and Gemini desktop apps (Gemini needs Apple silicon and macOS 15 or newer). Pick them in the TUI or with `mdt install --only chrome,zed,claude`.

Jest is no longer installed globally: React Native projects ship it as a project dependency.

# Usage

First make sure you clone this project using `git`:

    $ git clone git@github.com:JosePBrotons/mobile-dev-tools.git

Then run the script with `bash` (from any directory):

    $ bash mobile-dev-tools.sh -rn (React Native)
    $ bash mobile-dev-tools.sh -flutter (Flutter)
    $ bash mobile-dev-tools.sh -java (Java)
    $ bash mobile-dev-tools.sh -web (Web)

Running it again is safe: tools that are already installed are skipped.

To keep in mind:

- Installing [Xcode](https://developer.apple.com/xcode/) **requires being signed in with an Apple ID already inside the App Store App.**
- Some steps use `sudo`, so you may be asked for your admin password.
- The web profile runs `mkcert -install`, which asks for your admin password. OrbStack is free for personal use only.
- `ANDROID_HOME` is added to `~/.zprofile`. Open a new terminal after the script finishes so it takes effect.

# mdt (preview)

`mdt` is the Go replacement for the scripts. It reads the same tool list, skips what is already installed, and installs only what a profile or your own selection needs. It needs [Go](https://go.dev/) 1.24 or newer for now:

    $ go run ./cmd/mdt                                       (interactive TUI)
    $ go run ./cmd/mdt list                                  (profiles and tool ids)
    $ go run ./cmd/mdt install --profile rn --dry-run        (show the plan only)
    $ go run ./cmd/mdt install --profile rn                  (asks before installing)
    $ go run ./cmd/mdt install --only node,watchman --yes    (pick tools, no prompt)

Required tools are added automatically (Node.js brings nvm, CocoaPods brings Homebrew). A failed tool does not stop the rest; the summary lists it and the log is in `~/Library/Logs/mobile-dev-tools/`. 
Run `mdt` with no arguments in a terminal for the interactive mode. It checks the Mac, lets you start from a profile, toggle tools, review the exact commands, watch live progress and read a summary with next steps. Keys: `space` toggle, `a` select all, `/` filter, `enter` continue, `esc` back, `?` help, `q` quit. It asks for your admin password once before installing. The TUI has only been exercised with a fake runner and up to the review screen on a real Mac; no real install has been run through it yet.

# Contributing

Conventions for humans and AI agents live in [AGENTS.md](AGENTS.md).
