#!/bin/sh

# build binary releases for comments

build () {
  echo "building for $1 $2..."
  suffix=""
  if [ $1 = "windows" ]
  then
    suffix=".exe"
  fi
  GOOS=$1 GOARCH=$2 go build -o release/comments$suffix
  cd release
  if [ $1 = "linux" ]
  then
    tar cvf - comments/* comments$suffix | gzip -9 - > comments_$1_$2.tar.gz
  else
    7z a -tzip -r comments_$1_$2.zip comments comments$suffix
  fi
  rm -rf comments$suffix
  cd ..
}

rm -rf release
mkdir -p release

rsync -av templates/* static/* release/comments --delete --exclude public --exclude theme/node_modules

build linux 386
build linux amd64
build linux arm
build linux arm64

build darwin amd64
build darwin arm64

build windows 386
build windows amd64
build windows arm
build windows arm64

rm -rf release/comments