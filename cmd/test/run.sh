#!/bin/sh

set -e

echo "build cxod"
cd ../cxod
go build -race
echo "build cli"
cd ../cli
go build -race
echo "build test"
cd ../test
go build -race
echo "build source"
cd source
go build -race
echo "build drain"
cd ../drain
go build -race
echo "start tests"
cd ..
./test
