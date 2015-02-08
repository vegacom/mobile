#!/usr/bin/env bash
# All rights reserved. Use of this source code is governed by a BSD-style license that can be found in the LICENSE file.

set -e

if [ ! -f make.bash ]; then
	echo 'make.bash must be run from $GOPATH/src/github.com/vegacom/mobile/golife'
	exit 1
fi

mkdir -p jni/armeabi
CGO_ENABLED=1 GOOS=android GOARCH=arm GOARM=7 \
	go build -ldflags="-shared" -o jni/armeabi/libsprite.so .
ndk-build NDK_DEBUG=1
ant debug
