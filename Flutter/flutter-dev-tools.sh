#!/usr/bin/env bash
echo "Welcome, this script will install the dependencies and libraries that will help you throughout Flutter mobile development"
source ./Misc/install-command-line-tools.sh
source ./Misc/install-brew.sh
source ./Misc/brew-helpers.sh
echo "Installing iTerm2..."
brew_cask iterm2
echo "Installing Postman..."
brew_cask postman
echo "Installing VSCode..."
brew_cask visual-studio-code
echo "Check for Android Home Path"
source ./Misc/set-android-home-path.sh
echo "Installing CocoaPods..."
brew_formula cocoapods
echo "Installing Java Development Kit (JDK)..."
brew_cask zulu@17
echo "Installing Android Studio..."
brew_cask android-studio
echo "Installing Flutter..."
brew_cask flutter
echo "Installing Mac App Store CLI..."
brew_formula mas
echo "Installing Xcode by using the M.A.S CLI (requires being signed in to the App Store)..."
mas install 497799835
echo "Installing Fastlane for app automation..."
brew_formula fastlane
