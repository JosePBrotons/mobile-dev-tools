
# Mobile Dev Tools

Script that install on macOS the necessary tools to start developing on Java and also mobile apps with React Native and Flutter.

# What does this script install?

- [Homebrew Package Manager](https://brew.sh/)
- [iTerm2 Terminal Emulator](https://iterm2.com/)
- [Postman for API Development](https://www.postman.com/)
- [Visual Studio Code Editor](https://code.visualstudio.com/)
- [Node 10 LTS](https://nodejs.org/en/)
- [Yarn Package Manager](https://yarnpkg.com/)
- [Flutter](https://flutter.dev)
- [Flipper](https://fbflipper.com)
- [Facebook's Watchman - A File Watching Service](https://facebook.github.io/watchman/)
- [CocoaPods - Dependency Manager for Swift and Objective-C](https://cocoapods.org/)
- [JDK - Java Development Kit](https://openjdk.java.net/)
- [Android Studio - Android's IDE](https://developer.android.com/studio/)
- [React Native Debugger](https://github.com/jhen0409/react-native-debugger)
- [Xcode - iOS IDE](https://developer.apple.com/xcode/)
- [TypeScript - Globally](https://www.typescriptlang.org/)
- [Jest - JavaScript Testing Framework](https://jestjs.io/)
- [ngrok - Secure introspectable tunnel to localhost](https://ngrok.com/)
- [Maven](https://maven.apache.org/)
- [IntelliJ's IDEA](https://www.jetbrains.com/idea/)

# Usage

First make sure you clone this project using `git`:

    $ git clone git@github.com:JosePBrotons/mobile-dev-tools.git

Then, enter the project's folder and execute the script by using `sh` command:

    $ sh mobile-dev-tools.sh -rn (React Native)
    $ sh mobile-dev-tools.sh -flutter (Flutter)
    $ sh mobile-dev-tools.sh -java (Java)

To keep in mind:

- Installing [Xcode](https://developer.apple.com/xcode/) **requires being signed in with an Apple ID already inside the App Store App.**
- By using the `sudo` commmand inside this script you'll be needed to type your admin password.
- To install _yarn global modules_ you should restart the terminal an execute the `mobile-global-modules.sh` file inside the React Native folder.
