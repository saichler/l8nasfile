#!/usr/bin/env bash
set -e

rm -rf go.sum
rm -rf go.mod
rm -rf vendor

go mod init
GOPROXY=direct GOPRIVATE=github.com go mod tidy
go mod vendor

go build -o fileManager
export pw=$PWD

zip -r fileManager.zip ./fileManager ./web
#scp fileManager.zip $1:/root/fileManager.zip
rm fileManager

#mv fileManager.zip $pw1/.
