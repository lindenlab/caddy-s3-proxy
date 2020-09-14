#!/bin/bash
BRANCH=$DRONE_COMMIT_BRANCH
if [ "$BRANCH" = "" ]; then
	BRANCH=$(git rev-parse --abbrev-ref HEAD)
fi


calc_version () {
	VERSION_FILE=$1
	DIR=$(dirname $VERSION_FILE)
	if [ "$DIR" = "." ]; then
		MODULE_PATH=
	else
		MODULE_PATH=$DIR/
	fi

	echo $MODULE_PATH"v$(cat $VERSION_FILE)"
}

if [ "$1" = "check_version" ]; then
	shift
	for var in "$@"
	do
		VERSION=$(calc_version $var)
		VER_EXIST=$(git tag -l $VERSION)
		echo $VER_EXIST
		if [ "$DRONE_COMMIT_BRANCH" != master ] && [ -n "$VER_EXIST" ]; then echo "Need to update $var - exiting" && exit 1; fi
	done
	exit
fi

git checkout $BRANCH
git pull origin $BRANCH

for var in "$@"
do
	VERSION=$(calc_version $var)

	if [ "$BRANCH" != "master" ]; then
		git describe --match "$VERSION-pre.*" --abbrev=0 HEAD --tags 2> /dev/null
		COUNTER=1
		while [  $COUNTER -lt 15 ]; do
			tag="$VERSION-pre.$COUNTER"
			git describe --match $tag --abbrev=0 HEAD --tags 2> /dev/null
			if [ $? != 0 ]; then
				break;
			fi
	        	let COUNTER=COUNTER+1 
		done

		VERSION=$tag
	fi
	git tag $VERSION
done

git push --tags origin $BRANCH 

