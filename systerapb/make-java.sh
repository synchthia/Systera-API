#!/bin/bash

cd $(dirname $0)
BASEMENT=$PWD

echo "Generating Java Protoc..."
EXPORTDIR="${BASEMENT}/java/src/main/proto"
mkdir -p $EXPORTDIR
cp -Rfv $BASEMENT/*.proto $EXPORTDIR/
cd java
if [ "$1" == "deploy" ]; then
    mvn clean deploy
else
    mvn clean install
fi