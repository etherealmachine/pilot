#!/bin/sh
go run *.go -build
GOOS=linux GOARCH=arm GOARM=6 go build -o pilot *.go