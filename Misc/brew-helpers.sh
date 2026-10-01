#!/usr/bin/env bash
# Idempotent Homebrew helpers. Source after Misc/install-brew.sh.

brew_formula() {
    if brew list --formula "$1" >/dev/null 2>&1; then
        echo "$1 is already installed, skipping."
    else
        brew install "$1"
    fi
}

brew_cask() {
    if brew list --cask "$1" >/dev/null 2>&1; then
        echo "$1 is already installed, skipping."
    else
        brew install --cask "$1"
    fi
}

echo "Updating Homebrew..."
brew update
