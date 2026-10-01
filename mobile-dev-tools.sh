#!/usr/bin/env bash
# Run from the repo root so the relative `source` paths in each script resolve.
cd "$(dirname "$0")" || exit 1

if [[ $1 = "-rn" ]]; then
    echo "Installing React Native Dev Tools"
    source ./React\ Native/react-native-dev-tools.sh
elif [[ $1 = "-flutter" ]]; then
    echo "Installing Flutter Dev Tools"
    source ./Flutter/flutter-dev-tools.sh
elif [[ $1 = "-java" ]]; then
    echo "Installing Java Dev Tools"
    source ./Java/java-dev-tools.sh
elif [[ $1 = "-web" ]]; then
    echo "Installing Web Dev Tools"
    source ./Web/web-dev-tools.sh
else
    echo "You need to input the following arguments:"
    echo "-rn for React Native"
    echo "-flutter for Flutter"
    echo "-java for Java"
echo "-web for Web"
fi
