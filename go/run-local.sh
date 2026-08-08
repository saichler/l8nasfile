#!/usr/bin/env bash
set -e

# clean up
rm -rf go.sum
rm -rf go.mod
rm -rf vendor

# fetch dependencies
go mod init
GOPROXY=direct GOPRIVATE=github.com go mod tidy
go mod vendor

# build binaries
rm -rf demo
mkdir -p demo
cd nas/web
echo "Building NAS file server"
go build -o ../../demo/nasfile_demo
cd ../..

# copy web assets
cp -r ./nas/web/web ./demo/web

cd demo

echo "cd .." > ./kill_demo.sh
echo "rm -rf demo" >> ./kill_demo.sh
echo "pkill -9 nasfile_demo" >> ./kill_demo.sh
chmod +x ./kill_demo.sh

./nasfile_demo &
sleep 3

EXTERNAL_IP=$(ip route get 1.1.1.1 | grep -oP 'src \K[0-9.]+')
echo "NAS Remote File Manager running at: https://${EXTERNAL_IP}:3443/"
echo "Login with admin / admin"

read -p "Press Enter to kill the demo"
./kill_demo.sh
