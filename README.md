# Mobile Dev Tools

Scripts that install on macOS the tools needed to develop in Java and to build mobile apps with React Native and Flutter.

> A TUI that lets you choose exactly what to install is planned. See [ROADMAP.md](ROADMAP.md).

# What does this script install?

- [Xcode Command Line Tools](https://developer.apple.com/xcode/resources/)
- [Homebrew Package Manager](https://brew.sh/)
- [iTerm2 Terminal Emulator](https://iterm2.com/)
- [Postman for API Development](https://www.postman.com/)
- [Visual Studio Code Editor](https://code.visualstudio.com/)
- [nvm - Node Version Manager](https://github.com/nvm-sh/nvm)
- [Node.js LTS](https://nodejs.org/en/)
- [Yarn Package Manager v4 (via Corepack)](https://yarnpkg.com/)
- [Flutter](https://flutter.dev)
- [Facebook's Watchman - A File Watching Service](https://facebook.github.io/watchman/)
- [CocoaPods - Dependency Manager for Swift and Objective-C](https://cocoapods.org/)
- [JDK 17 - Azul Zulu](https://www.azul.com/downloads/)
- [Android Studio - Android's IDE](https://developer.android.com/studio/)
- [Xcode - iOS IDE](https://developer.apple.com/xcode/)
- [Fastlane - App automation](https://fastlane.tools/)
- [TypeScript - Globally (via npm)](https://www.typescriptlang.org/)
- [ngrok - Secure introspectable tunnel to localhost (via Homebrew)](https://ngrok.com/)
- [Maven](https://maven.apache.org/)
- [IntelliJ IDEA](https://www.jetbrains.com/idea/)

Jest is no longer installed globally: React Native projects ship it as a project dependency.

# Usage

First make sure you clone this project using `git`:

    $ git clone git@github.com:JosePBrotons/mobile-dev-tools.git

Then run the script with `bash` (from any directory):

    $ bash mobile-dev-tools.sh -rn (React Native)
    $ bash mobile-dev-tools.sh -flutter (Flutter)
    $ bash mobile-dev-tools.sh -java (Java)

Running it again is safe: tools that are already installed are skipped.

To keep in mind:

- Installing [Xcode](https://developer.apple.com/xcode/) **requires being signed in with an Apple ID already inside the App Store App.**
- Some steps use `sudo`, so you may be asked for your admin password.
- `ANDROID_HOME` is added to `~/.zprofile`. Open a new terminal after the script finishes so it takes effect.

# Contributing

Conventions for humans and AI agents live in [AGENTS.md](AGENTS.md).
