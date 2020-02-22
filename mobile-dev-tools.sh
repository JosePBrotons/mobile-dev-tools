echo "Welcome, this script will install the dependencies and libraries that will help you throughout mobile development"
echo "Installing Homebrew..."
/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"
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
"Verifying if .bash_profile file does exist..."
cd --
    if [ -e .bash_profile ]
        then
            echo ".bash_profile file exists."
                if grep -Fxq 'export ANDROID_HOME=$HOME/Library/Android/sdk' .bash_profile
                    then
                        echo 'Found modified .bash_profile, not modifying...'
                    else
                        echo "Adding ANDROID_HOME Paths..."
                        echo 'export ANDROID_HOME=$HOME/Library/Android/sdk' >> .bash_profile
                        echo 'export PATH=$PATH:$ANDROID_HOME/emulator' >> .bash_profile
                        echo 'export PATH=$PATH:$ANDROID_HOME/tools' >> .bash_profile
                        echo 'export PATH=$PATH:$ANDROID_HOME/tools/bin' >> .bash_profile
                        echo 'export PATH=$PATH:$ANDROID_HOME/platform-tools' >> .bash_profile
                fi
    else
        echo "Creating .bash_profile file..."
        touch .bash_profile
            echo "Adding ANDROID_HOME Paths..."
            echo 'export ANDROID_HOME=$HOME/Library/Android/sdk' >> .bash_profile
            echo 'export PATH=$PATH:$ANDROID_HOME/emulator' >> .bash_profile
            echo 'export PATH=$PATH:$ANDROID_HOME/tools' >> .bash_profile
            echo 'export PATH=$PATH:$ANDROID_HOME/tools/bin' >> .bash_profile
            echo 'export PATH=$PATH:$ANDROID_HOME/platform-tools' >> .bash_profile
    fi
echo "Installing Yarn Package Manager..."
curl -o- -L https://yarnpkg.com/install.sh | bash
echo "Installing Facebook's Watchman..."
brew update && brew install watchman
echo "Installing CocoaPods... (Admin's Password is needed)"
sudo gem install cocoapods
echo "Installing Java Development Kit (JDK)..."
brew update && brew tap AdoptOpenJDK/openjdk && brew cask install adoptopenjdk8
echo "Installing Android Studio..."
brew update && brew cask install android-studio
echo "Installing React Native Debugger"
brew update && brew cask install react-native-debugger
echo "Installing Mac App Store CLI..."
brew update && brew install mas
echo "Installing Xcode by using the M.A.S CLI..."
mas install 497799835
echo "Installing Fastlane for app automation..."
brew install fastlane