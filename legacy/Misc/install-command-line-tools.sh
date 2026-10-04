#!/usr/bin/env bash
if xcode-select -p >/dev/null 2>&1; then
    echo "Xcode command line tools are already installed."
else
    echo "Installing Xcode command line tools..."
    xcode-select --install
fi
