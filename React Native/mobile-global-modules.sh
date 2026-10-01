#!/usr/bin/env bash
# Global tooling for React Native. Yarn 4 has no `yarn global`, so npm (from nvm) is used.
# Sourced by react-native-dev-tools.sh after nvm is loaded; can also run on its own.
echo "Installing global modules..."
echo "Installing TypeScript Globally..."
npm install -g typescript
echo "Installing ngrok..."
if ! declare -F brew_cask >/dev/null; then
    source ./Misc/brew-helpers.sh
fi
brew_cask ngrok
