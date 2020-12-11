echo "Welcome, this script will install the dependencies and libraries that will help you throughout Flutter mobile development"
source ./Misc/install-brew.sh
echo "Installing iTerm2..."
brew update && brew install --cask iterm2
echo "Installing Postman..."
brew update && brew install --cask postman
echo "Installing VSCode..."
brew update && brew install --cask visual-studio-code
echo "Check for Android Home Path"
source ./Misc/set-android-home-path.sh
echo "Installing CocoaPods... (Admin's Password is needed)"
sudo gem install cocoapods
echo "Installing Java Development Kit (JDK)..."
brew update && brew tap AdoptOpenJDK/openjdk && brew install --cask adoptopenjdk11
echo "Installing Android Studio..."
brew update && brew install --cask android-studio
echo "Installing Flutter..."
brew update && brew install --cask flutter
echo "Installing Mac App Store CLI..."
brew update && brew install mas
echo "Installing Xcode by using the M.A.S CLI..."
mas install 497799835
echo "Installing Fastlane for app automation..."
brew install fastlane
