#!/bin/bash

echo "mode: set" > cover.out
for Dir in $(find ./* -maxdepth 10 -type d ); 
do
	if ls $Dir/*.go &> /dev/null;
	then
		returnval=`go test -coverprofile=temp.out $Dir`
		echo ${returnval}
		if [[ ${returnval} != *FAIL* ]]
		then
    		if [ -f temp.out ]
    		then
        		cat temp.out | grep -v "mode: set" >> cover.out 
    		fi
    	else
    		exit 1
    	fi	
    fi
done

goveralls -service=travis-ci -coverprofile=cover.out

rm -rf ./temp.out
rm -rf ./cover.out
