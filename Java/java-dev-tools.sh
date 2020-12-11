echo "Welcome, this script will install the dependencies and libraries that will help you throughout Java development"
source ./Misc/install-brew.sh
echo "Installing Java Development Kit (JDK)..."
brew update && brew tap AdoptOpenJDK/openjdk && brew install --cask adoptopenjdk11
echo "Installing intelliJ IDEA Community Edition..."
brew update && brew install --cask intellij-idea-ce
echo "Installing Maven..."
brew update && brew install maven