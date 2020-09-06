echo "Installing Java Development Kit (JDK)..."
brew update && brew tap AdoptOpenJDK/openjdk && brew cask install adoptopenjdk11
echo "Installing intelliJ IDEA Community Edition..."
brew update && brew cask install intellij-idea-ce
echo "Installing Maven..."
brew update && brew install maven