#!/bin/sh

set -e

cd ../cxod
go build
cd ../cli
go build
cd ../test
go build
cd source
go build
cd ../drain
go build
cd ..
./test
