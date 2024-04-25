#!/bin/bash

rootdir=$(cd `dirname $0`; cd ..; pwd)
output=$rootdir/dist
version=`cat ./util/const.go | grep -E "Version.+ string =" | cut -d"=" -f2 | xargs`

# 创建输出目录
mkdir -p $output

# 定义构建函数
build() {
    local os=$1
    local arch=$2
    local file_suffix=$3

    echo "start build coscli for $os on $arch"
    cd $rootdir
    env GOOS=$os GOARCH=$arch go build -o $output/coscli-$version-$os-$arch$file_suffix
    echo "coscli for $os on $arch built successfully"
}

# 定义计算哈希值的函数
calc_hash() {
    cd $output
    for file in $(ls *); do
        sha256sum $file >> sha256sum.log
    done
}

# 构建不同平台的二进制文件
build darwin amd64 ""
build darwin arm64 ""
build windows 386 ".exe"
build windows amd64 ".exe"
build linux 386 ""
build linux amd64 ""
build linux arm ""
build linux arm64 ""

# 计算哈希值
calc_hash