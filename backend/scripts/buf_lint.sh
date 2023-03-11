#!/bin/sh

RootDir=$( cd "$(dirname $0)/.." && pwd)
ProtoDir=$RootDir/proto

cd $ProtoDir
buf lint