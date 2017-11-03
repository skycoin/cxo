#!/bin/evn bash

GOPATH=~/gonor:$GOPATH

# $1 - os
# $2 - arch
function goBuildOsArch {
	local os="$1"
	local arch="$2"
	local humanOsName="$1"
	local humanArchName="$2"
	local executable="cxodbfix"



	if [ "$os" == "darwin" ]
	then
		humanOsName="mac"
	fi


	if [ "$arch" == "amd64" ]
	then
		humanArchName="64"
	fi

	if [ "$os" == "dragonfly" ]
	then
		if [ "$arch" == "386" ]
		then
			return # skip dragonfly 386
		fi
		humanArchName=""
	fi

	if [ "$os" == "windows" ]
	then
		executable="cxodbfix.exe"
	fi

	echo "build $os $arch"

	GOOS="$os" GOARCH="$arch" go build -ldflags "-s"

	local place="$humanOsName"

	place+="$humanArchName"
	place+="_cxodbfix_2.1"

	mkdir -pv "$place"
	mv -v  "$executable" "$place"

	local ar="$place"

	if [ "$os" == "windows" ]
	then
		ar+=".zip"
		zip -9 -r "$ar" "$place"
	else
		ar+=".tar.gz"
		tar -zcvf "$ar" "$place"
	fi

	rm -rf "$place"
}

for os in dragonfly freebsd linux darwin openbsd windows
do
	for arch in 386 amd64
	do
		goBuildOsArch $os $arch
	done
done
