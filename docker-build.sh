#!/usr/bin/env bash

# 使用当前时间作为版本号，格式为 YYYYMMDDHHMM
VERSION=$(date +%Y%m%d%H%M)

docker buildx build --platform=linux/amd64 -t mylxsw/secure-proxy:$VERSION -t mylxsw/secure-proxy:latest . --push
