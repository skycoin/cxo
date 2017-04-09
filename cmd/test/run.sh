#!/bin/sh

echo "build cxod"
cd ../cxod && go build -race || ( pkill -P $$ && exit 1 )
echo "build cli"
cd ../cli && go build -race || ( pkill -P $$ && exit 1 )
echo "build test"
cd ../test && go build -race || ( pkill -P $$ && exit 1 )
echo "build source"
cd source && go build -race || ( pkill -P $$ && exit 1 )
echo "build drain"
cd ../drain && go build -race || ( pkill -P $$ && exit 1 )
echo "start tests"
cd .. && ./test || ( pkill -P $$ && exit 1 )
pkill -P $$
