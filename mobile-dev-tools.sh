if [[ $1 = "-rn" ]]; then
    echo "Installing React Native Dev Tools"
    source ./React\ Native/react-native-dev-tools.sh
elif [[ $1 = "-flutter" ]]; then
    echo "Installing Flutter Dev Tools"
    source ./Flutter/flutter-dev-tools.sh
elif [[ $1 = "-java" ]]; then
    echo "Installing Java Dev Tools"
    source ./Java/java-dev-tools.sh
else
    echo "You need to input the following arguments:"
    echo "-rn for React Native"
    echo "-flutter for Flutter"
    echo "-java for Java"
fi
