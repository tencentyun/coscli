#!/bin/bash

rootdir=$(cd `dirname $0`; cd ..; pwd)
output=$rootdir/dist
version=`cat ./util/const.go | grep -E "Version.+ string =" | cut -d"=" -f2 | xargs`

# 两个发布文件夹（名称固定，不随版本变动）：
#   coscli-versioned 目录：文件名带版本号，例如 coscli-v1.0.9-darwin-arm64
#   coscli           目录：文件名不带版本号，例如 coscli-darwin-arm64
versioned_dir=$output/coscli-versioned
plain_dir=$output/coscli

# 重新创建输出目录（清理旧产物，避免历史文件混入哈希）
rm -rf $versioned_dir $plain_dir
mkdir -p $versioned_dir $plain_dir

# 定义构建函数
build() {
    local os=$1
    local arch=$2
    local file_suffix=$3

    echo "start build coscli for $os on $arch"
    cd $rootdir

    local versioned_path=$versioned_dir/coscli-$version-$os-$arch$file_suffix
    local plain_path=$plain_dir/coscli-$os-$arch$file_suffix

    env GOOS=$os GOARCH=$arch go build -o $versioned_path
    if [ $? -ne 0 ]; then
        echo "build failed for $os/$arch"
        exit 1
    fi

    # 复制一份到不带版本号的目录，避免重复编译
    cp $versioned_path $plain_path
    echo "coscli for $os on $arch built successfully"
}

# 定义计算哈希值的函数（对指定目录内所有产物计算 sha256）
calc_hash() {
    local dir=$1
    cd $dir
    rm -f sha256sum.log # 删除现有的 sha256sum.log 文件（如果存在）
    for file in $(ls *); do
        if [ "$file" != "sha256sum.log" ]; then
            sha256sum $file >> sha256sum.log
        fi
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

# 分别计算两个目录的哈希值
calc_hash $versioned_dir
calc_hash $plain_dir

echo ""
echo "build done."
echo "  versioned: $versioned_dir"
echo "  plain    : $plain_dir"
