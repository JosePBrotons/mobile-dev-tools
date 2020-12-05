which -s brew
echo "Checking if Homebrew is installed..."
if [[ $? != 0 ]]; then
    echo "Homebrew not installed, installing Homebrew..."
    /usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"
else
    echo "Homebrew's installed, updating Homebrew..."
    brew update
fi
