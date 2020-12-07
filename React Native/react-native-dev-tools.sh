echo "Welcome, this script will install the dependencies and libraries that will help you throughout React Native's mobile development"
source ./Misc/install-command-line-tools.sh
source ./Misc/check-brew-installation.sh
echo "Installing iTerm2..."
brew update && brew install --cask iterm2
echo "Installing Postman..."
brew update && brew install --cask postman
echo "Installing VSCode..."
brew update && brew install --cask visual-studio-code
echo "Installing Node 10 LTS Version..."
brew update && brew install node@10
echo "Forcing link of Node 10..."
brew link --force node@10
echo "Check for Android Home Path"
source ./Misc/set-android-home-path.sh
echo "Installing Yarn Package Manager..."
curl -o- -L https://yarnpkg.com/install.sh | bash
echo "Installing Facebook's Watchman..."
brew update && brew install watchman
echo "Installing CocoaPods... (Admin's Password is needed)"
sudo gem install cocoapods
echo "Installing Java Development Kit (JDK)..."
brew update && brew tap AdoptOpenJDK/openjdk && brew install --cask adoptopenjdk11
echo "Installing Android Studio..."
brew update && brew install --cask android-studio
echo "Installing React Native Debugger"
brew update && brew install --cask react-native-debugger
echo "Installing Facebook's Flipper"
brew update && brew install --cask flipper
echo "Installing Mac App Store CLI..."
brew update && brew install mas
echo "Installing Xcode by using the M.A.S CLI..."
mas install 497799835
echo "Installing Fastlane for app automation..."
brew install fastlane
