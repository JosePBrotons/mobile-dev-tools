echo "Welcome, this script will install the dependencies and libraries that will help you throughout React Native's mobile development"
source ./Misc/install-command-line-tools.sh
source ./Misc/install-brew.sh
echo "Installing iTerm2..."
brew update && brew install --cask iterm2
echo "Installing Postman..."
brew update && brew install --cask postman
echo "Installing VSCode..."
brew update && brew install --cask visual-studio-code
echo "Installing nvm (Node Version Manager)..."
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.4/install.sh | bash
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
echo "Installing Node.js LTS version..."
nvm install --lts
nvm use --lts
nvm alias default 'lts/*'
echo "Enabling Corepack for Yarn..."
corepack enable
corepack prepare yarn@stable --activate
echo "Check for Android Home Path"
source ./Misc/set-android-home-path.sh
echo "Installing Facebook's Watchman..."
brew update && brew install watchman
echo "Installing CocoaPods... (Admin's Password is needed)"
brew install cocoapods
echo "Installing Java Development Kit (JDK)..."
brew update && brew install --cask zulu@17
echo "Installing Android Studio..."
brew update && brew install --cask android-studio
echo "Installing Mac App Store CLI..."
brew update && brew install mas
echo "Installing Xcode by using the M.A.S CLI..."
mas install 497799835
echo "Installing Fastlane for app automation..."
brew install fastlane
