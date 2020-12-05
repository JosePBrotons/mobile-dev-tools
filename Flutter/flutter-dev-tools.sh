echo "Welcome, this script will install the dependencies and libraries that will help you throughout Flutter mobile development"
source ./Misc/check-brew-installation.sh
echo "Installing iTerm2..."
brew update && brew cask install iterm2
echo "Installing Postman..."
brew update && brew cask install postman
echo "Installing VSCode..."
brew update && brew cask install visual-studio-code
echo "Check for Android Home Path"
source ./Misc/set-android-home-path.sh
echo "Installing CocoaPods... (Admin's Password is needed)"
sudo gem install cocoapods
echo "Installing Java Development Kit (JDK)..."
brew update && brew tap AdoptOpenJDK/openjdk && brew cask install adoptopenjdk11
echo "Installing Android Studio..."
brew update && brew cask install android-studio
echo "Installing Flutter..."
brew update && brew cask install flutter
echo "Installing Mac App Store CLI..."
brew update && brew install mas
echo "Installing Xcode by using the M.A.S CLI..."
mas install 497799835
echo "Installing Fastlane for app automation..."
brew install fastlane
