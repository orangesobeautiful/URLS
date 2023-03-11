#!/bin/sh

RootDir=$( cd "$(dirname $0)/.." && pwd)
ProtoDir=$RootDir/proto
GenDir=$ProtoDir/gen

[ -e $GenDir ] && rm -r $GenDir
cd $ProtoDir
buf generate