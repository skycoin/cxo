#!/bin/sh

set -e

echo "build cxod"
cd ../cxod
go build
echo "build cli"
cd ../cli
go build
echo "build test"
cd ../test
go build
echo "build source"
cd source
go build
echo "build drain"
cd ../drain
go build
echo "start tests"
cd ..
./test
