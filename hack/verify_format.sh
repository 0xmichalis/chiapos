#!/bin/bash -e

FILES=$(gofmt -s -l cmd pkg)

if [[ -n "${FILES}" ]]; then
    echo You have go format errors in the below files, please run "gofmt -s -w cmd pkg"
    echo ${FILES}
    exit 1
fi

FILES=$(goimports -e -l -local=github.com/kargakis/chiapos cmd pkg)

if [[ -n "${FILES}" ]]; then
    echo You have go import errors in the below files, please run "goimports -e -w -local=github.com/kargakis/chiapos cmd pkg"
    echo ${FILES}
    exit 1
fi
