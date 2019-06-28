#!/bin/bash

cd "$(dirname "$0")"/..

if [ ! -e $HOME/.m2/ ]; then
    mkdir $HOME/.m2/
fi

if [ ! -e $PWD/.env ]; then
    echo "!! .env does not exists!"
    exit 1
fi
source .env
echo -n ${CI_MVN_SETTINGS} > $HOME/.m2/settings.xml
