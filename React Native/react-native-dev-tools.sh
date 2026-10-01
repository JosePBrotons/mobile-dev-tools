#!/usr/bin/env bash
echo "Welcome, this script will install the dependencies and libraries that will help you throughout React Native's mobile development"
source ./Misc/install-command-line-tools.sh
source ./Misc/install-brew.sh
source ./Misc/brew-helpers.sh
echo "Installing Git..."
brew_formula git
echo "Installing GitHub CLI..."
brew_formula gh
echo "Installing iTerm2..."
brew_cask iterm2
echo "Installing Postman..."
brew_cask postman
echo "Installing VSCode..."
brew_cask visual-studio-code
echo "Installing nvm (Node Version Manager)..."
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.8/install.sh | bash
export NVM_DIR="$HOME/.nvm"
# shellcheck source=/dev/null
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
echo "Installing Node.js LTS version..."
nvm install --lts
nvm use --lts
nvm alias default 'lts/*'
echo "Installing pnpm..."
brew_formula pnpm
echo "Check for Android Home Path"
source ./Misc/set-android-home-path.sh
echo "Installing Facebook's Watchman..."
brew_formula watchman
echo "Installing CocoaPods..."
brew_formula cocoapods
echo "Installing Java Development Kit (JDK)..."
brew_cask zulu@17
echo "Installing Android Studio..."
brew_cask android-studio
echo "Installing Mac App Store CLI..."
brew_formula mas
echo "Installing Xcode by using the M.A.S CLI (requires being signed in to the App Store)..."
mas install 497799835
echo "Installing Fastlane for app automation..."
brew_formula fastlane
source "./React Native/mobile-global-modules.sh"
