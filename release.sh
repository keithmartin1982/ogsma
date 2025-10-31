#!/bin/bash
wsEndpoint="ws"
addr="10.1.10.194:8443"
names=("chad" "stacy")
passwords=("password1234!" "password1234!")
num_names=${#names[@]}
programName="ogsma"

# build required executables
cd ./config_gen/ || exit
go build .
cd ../keystore_gen/ || exit
go build .
cd ../

# Generate keystore files for names
for (( i=0; i<num_names; i++ )); do
  echo "generating keystore file for: ${names[$i]} with pass: ${passwords[$i]}"
  ./keystore_gen/keystore_gen  --password "${passwords[$i]}" --new "${names[$i]}" --keystore "${names[$i]}.keystore"
done

# Add contacts to keystore
for (( i=0; i<num_names; i++ )); do
  targetName="${names[$i]}"
  echo "adding contacts to keystore for: ${targetName}"
  for (( j=0; j<num_names; j++ )); do
    contactName="${names[$j]}"
    if [[ "$targetName" != "$contactName" ]]; then
      echo "adding: ${contactName} to  keystore for: ${targetName}"
      ./keystore_gen/keystore_gen --password "${passwords[$i]}" --add "${contactName}.keyshare" --keystore "${targetName}.keystore"
    fi
  done
done

rm ./*.keyshare

# generate config.json
for (( i=0; i<num_names; i++ )); do
  targetName="${names[$i]}"
  keystoreString=$(cat "${targetName}.keystore")
  echo "generating config.json file for: ${targetName}"
  ./config_gen/config_gen --keystore "${keystoreString}" --addr "${addr}" --endpoint "${wsEndpoint}" > "${targetName}_config.json"
done

rm ./*.keystore

# build client for names
for (( i=0; i<num_names; i++ )); do
    targetName="${names[$i]}"
    echo "building executable for ${targetName}"
    cp "${targetName}_config.json" ./client/config.json
    cd ./client || exit
    go build -o "../${targetName}_${programName}" .
    ANDROID_NDK_HOME="$HOME/Android/android-ndk-r21e" fyne p --os android/arm64
    mv ./ogsma.apk "../${targetName}_${programName}.apk"
    rm config.json
    cd ../
done