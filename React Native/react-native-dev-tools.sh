echo "Welcome, this script will install the dependencies and libraries that will help you throughout React Native's mobile development"
source ./Misc/check-brew-installation.sh
echo "Installing iTerm2..."
brew update && brew cask install iterm2
echo "Installing Postman..."
brew update && brew cask install postman
echo "Installing VSCode..."
brew update && brew cask install visual-studio-code
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
brew update && brew tap AdoptOpenJDK/openjdk && brew cask install adoptopenjdk11
echo "Installing Android Studio..."
brew update && brew cask install android-studio
echo "Installing React Native Debugger"
brew update && brew cask install react-native-debugger
echo "Installing Facebook's Flipper"
brew update && brew cask install flipper
echo "Installing Mac App Store CLI..."
brew update && brew install mas
echo "Installing Xcode by using the M.A.S CLI..."
mas install 497799835
echo "Installing Fastlane for app automation..."
brew install fastlane