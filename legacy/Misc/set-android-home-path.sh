#!/usr/bin/env bash
# Adds ANDROID_HOME and the Android SDK tool folders to the user's shell profile.

currentOsVersion="$(sw_vers -productVersion)"
requiredVersion="10.15.0"
if [ "$(printf '%s\n' "$requiredVersion" "$currentOsVersion" | sort -V | head -n1)" = "$requiredVersion" ]; then
    profileFile="$HOME/.zprofile"
else
    profileFile="$HOME/.bash_profile"
fi

echo "Verifying if $profileFile file does exist..."
if [ ! -e "$profileFile" ]; then
    echo "Creating $profileFile file..."
    touch "$profileFile"
fi

# shellcheck disable=SC2016 # Lines are written literally so they expand when the profile loads.
if grep -Fxq 'export ANDROID_HOME=$HOME/Library/Android/sdk' "$profileFile"; then
    echo "Found modified $profileFile, not modifying..."
else
    echo "Adding ANDROID_HOME Paths..."
    {
        echo 'export ANDROID_HOME=$HOME/Library/Android/sdk'
        echo 'export PATH=$PATH:$ANDROID_HOME/emulator'
        echo 'export PATH=$PATH:$ANDROID_HOME/cmdline-tools/latest/bin'
        echo 'export PATH=$PATH:$ANDROID_HOME/platform-tools'
    } >>"$profileFile"
fi
