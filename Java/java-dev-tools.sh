#!/usr/bin/env bash
echo "Welcome, this script will install the dependencies and libraries that will help you throughout Java development"
source ./Misc/install-command-line-tools.sh
source ./Misc/install-brew.sh
source ./Misc/brew-helpers.sh
echo "Installing Java Development Kit (JDK)..."
brew_cask zulu@17
echo "Installing IntelliJ IDEA..."
brew_cask intellij-idea
echo "Installing Maven..."
brew_formula maven
