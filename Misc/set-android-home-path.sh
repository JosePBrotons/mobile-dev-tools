echo "Verifying if .bash_profile file does exist..."
cd --
currentOsVersion="$(sw_vers -productVersion)"
requiredVersion="10.15.0"
if [ "$(printf '%s\n' "$requiredVersion" "$currentOsVersion" | sort -V | head -n1)" = "$requiredVersion" ]; then
    if [ -e .zprofile ]; then
        echo ".zprofile file exists."
        if grep -Fxq 'export ANDROID_HOME=$HOME/Library/Android/sdk' .zprofile; then
            echo 'Found modified .zprofile, not modifying...'
        else
            echo "Adding ANDROID_HOME Paths..."
            echo 'export ANDROID_HOME=$HOME/Library/Android/sdk' >>.zprofile
            echo 'export PATH=$PATH:$ANDROID_HOME/emulator' >>.zprofile
            echo 'export PATH=$PATH:$ANDROID_HOME/tools' >>.zprofile
            echo 'export PATH=$PATH:$ANDROID_HOME/tools/bin' >>.zprofile
            echo 'export PATH=$PATH:$ANDROID_HOME/platform-tools' >>.zprofile
        fi
    else
        echo "Creating .zprofile file..."
        touch .zprofile
        echo "Adding ANDROID_HOME Paths..."
        echo 'export ANDROID_HOME=$HOME/Library/Android/sdk' >>.zprofile
        echo 'export PATH=$PATH:$ANDROID_HOME/emulator' >>.zprofile
        echo 'export PATH=$PATH:$ANDROID_HOME/tools' >>.zprofile
        echo 'export PATH=$PATH:$ANDROID_HOME/tools/bin' >>.zprofile
        echo 'export PATH=$PATH:$ANDROID_HOME/platform-tools' >>.zprofile
    fi

else
    echo "Verifying if .bash_profile file does exist..."
    if [ -e .bash_profile ]; then
        echo ".bash_profile file exists."
        if grep -Fxq 'export ANDROID_HOME=$HOME/Library/Android/sdk' .bash_profile; then
            echo 'Found modified .bash_profile, not modifying...'
        else
            echo "Adding ANDROID_HOME Paths..."
            echo 'export ANDROID_HOME=$HOME/Library/Android/sdk' >>.bash_profile
            echo 'export PATH=$PATH:$ANDROID_HOME/emulator' >>.bash_profile
            echo 'export PATH=$PATH:$ANDROID_HOME/tools' >>.bash_profile
            echo 'export PATH=$PATH:$ANDROID_HOME/tools/bin' >>.bash_profile
            echo 'export PATH=$PATH:$ANDROID_HOME/platform-tools' >>.bash_profile
        fi
    else
        echo "Creating .bash_profile file..."
        touch .bash_profile
        echo "Adding ANDROID_HOME Paths..."
        echo 'export ANDROID_HOME=$HOME/Library/Android/sdk' >>.bash_profile
        echo 'export PATH=$PATH:$ANDROID_HOME/emulator' >>.bash_profile
        echo 'export PATH=$PATH:$ANDROID_HOME/tools' >>.bash_profile
        echo 'export PATH=$PATH:$ANDROID_HOME/tools/bin' >>.bash_profile
        echo 'export PATH=$PATH:$ANDROID_HOME/platform-tools' >>.bash_profile
    fi
fi
