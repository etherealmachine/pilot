#!/bin/sh
GOOS=linux GOARCH=arm GOARM=6 go build -o pilot *.go