package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

const (
	AesKey       = "coscli-secret"
	AesBlockSize = 16
)

func DecryptSecret(encode string) (decryptStr string, err error) {
	decode, err := base64.StdEncoding.DecodeString(encode)
	if err != nil {
		return "", err
	}
	tool := NewAesTool([]byte(AesKey), AesBlockSize, ECB)
	decrypt, err := tool.Decrypt(decode)
	decryptStr = string(decrypt)
	return decryptStr, err
}

func EncryptSecret(src string) (encode string, err error) {
	tool := NewAesTool([]byte(AesKey), AesBlockSize, ECB)
	encrypt, err := tool.Encrypt([]byte(src))
	encode = base64.StdEncoding.EncodeToString(encrypt)
	return encode, err
}

const (
	ECB = 1
	CBC = 2
)

// AES ECB模式的加密解密
type AesTool struct {
	// 128 192  256位的其中一个 长度 对应分别是 16 24  32字节长度
	Key       []byte
	BlockSize int
	Mode      int
}

func NewAesTool(key []byte, blockSize int, mode int) *AesTool {
	return &AesTool{Key: key, BlockSize: blockSize, Mode: mode}
}

/**
注意：0填充方式
*/
func (this *AesTool) padding(src []byte) []byte {
	// 填充个数
	paddingCount := aes.BlockSize - len(src)%aes.BlockSize
	if paddingCount == 0 {
		return src
	} else {
		// 填充数据
		return append(src, bytes.Repeat([]byte{byte(0)}, paddingCount)...)
	}
}

// unpadding
func (this *AesTool) unPadding(src []byte) []byte {
	for i := len(src) - 1; i >= 0; i-- {
		if src[i] != 0 {
			return src[:i+1]
		}
	}
	return nil
}

func (this *AesTool) Encrypt(src []byte) ([]byte, error) {
	var encryptData []byte
	// key只能是 16 24 32长度
	key := this.padding(this.Key)
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}
	// padding
	src = this.padding(src)

	switch this.Mode {
	case ECB:
		encryptData = make([]byte, len(src))
		mode := NewECBEncrypter(block)
		mode.CryptBlocks(encryptData, src)
		break
	case CBC:
		// The IV needs to be unique, but not secure. Therefore it's common to
		// include it at the beginning of the ciphertext.
		encryptData := make([]byte, aes.BlockSize+len(src))
		iv := encryptData[:aes.BlockSize]
		if _, err := io.ReadFull(rand.Reader, iv); err != nil {
			panic(err)
		}

		mode := cipher.NewCBCEncrypter(block, iv)
		mode.CryptBlocks(encryptData[aes.BlockSize:], src)
		break
	}

	return encryptData, nil

}
func (this *AesTool) Decrypt(src []byte) (res []byte, err error) {
	defer func() {
		if err1 := recover(); err1 != nil {
			err = fmt.Errorf(fmt.Sprintf("%v", err1))
		}
	}()

	// key只能是 16 24 32长度
	key := this.padding(this.Key)

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}

	switch this.Mode {
	case ECB:
		mode := NewECBDecrypter(block)

		// CryptBlocks can work in-place if the two arguments are the same.
		mode.CryptBlocks(src, src)
		break
	case CBC:
		iv := src[:aes.BlockSize]
		src = src[aes.BlockSize:]
		fmt.Printf("decode iv :%x\n", iv)
		// CBC mode always works in whole blocks.
		if len(src)%aes.BlockSize != 0 {
			panic("ciphertext is not a multiple of the block size")
		}

		mode := cipher.NewCBCDecrypter(block, iv)

		// CryptBlocks can work in-place if the two arguments are the same.
		mode.CryptBlocks(src, src)
		break
	}

	return this.unPadding(src), nil
}

// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Electronic Code Book (ECB) mode.

// ECB provides confidentiality by assigning a fixed ciphertext block to each
// plaintext block.

// See NIST SP 800-38A, pp 08-09

type ecb struct {
	b         cipher.Block
	blockSize int
}

func newECB(b cipher.Block) *ecb {
	return &ecb{
		b:         b,
		blockSize: b.BlockSize(),
	}
}

type ecbEncrypter ecb

// NewECBEncrypter returns a BlockMode which encrypts in electronic code book
// mode, using the given Block.
func NewECBEncrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbEncrypter)(newECB(b))
}

func (x *ecbEncrypter) BlockSize() int { return x.blockSize }

func (x *ecbEncrypter) CryptBlocks(dst, src []byte) {
	if len(src)%x.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		x.b.Encrypt(dst, src[:x.blockSize])
		src = src[x.blockSize:]
		dst = dst[x.blockSize:]
	}
}

type ecbDecrypter ecb

// NewECBDecrypter returns a BlockMode which decrypts in electronic code book
// mode, using the given Block.
func NewECBDecrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbDecrypter)(newECB(b))
}

func (x *ecbDecrypter) BlockSize() int { return x.blockSize }

func (x *ecbDecrypter) CryptBlocks(dst, src []byte) {
	if len(src)%x.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		x.b.Decrypt(dst, src[:x.blockSize])
		src = src[x.blockSize:]
		dst = dst[x.blockSize:]
	}
}
