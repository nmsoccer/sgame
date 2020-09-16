package comm

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

const (
	ENCRY_DES_KEY_LEN = 8
	ENCRY_AES_KEY_LEN = 16


	NET_ENCRYPT_NONE int8 = 0
	NET_ENCRYPT_DES_ECB int8 = 1 //desc-ecb
	NET_ENCRYPT_AES_CBC_128 int8 = 2 //aes-cbc-128
	NET_ENCRYPT_RSA int8     = 3 //rsa + des
)

//pad & unpad
func Pkcs5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}
func Pkcs5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

/*******************DES***********************/
//DES-ECB key must 8bits
//if @block != nil use it , or new block
func DesEncrypt(block cipher.Block , src []byte, key []byte) ([]byte, error) {
	var err error
	if block == nil {
		block, err = des.NewCipher(key)
		if err != nil {
			return nil, err
		}
	}
	bs := block.BlockSize()
	src = Pkcs5Padding(src, bs)
	if len(src)%bs != 0 {
		return nil, errors.New("block size illegal!")
	}
	out := make([]byte, len(src))
	dst := out
	for len(src) > 0 {
		block.Encrypt(dst, src[:bs])
		src = src[bs:]
		dst = dst[bs:]
	}
	return out , nil
}

//if @block != nil use it , or new block
func DesDecrypt(block cipher.Block , src []byte , key []byte) ([]byte, error) {
	var err error
	if block == nil {
		block, err = des.NewCipher(key)
		if err != nil {
			return nil, err
		}
	}
	out := make([]byte, len(src))
	dst := out
	bs := block.BlockSize()
	if len(src)%bs != 0 {
		return nil, errors.New("crypto/cipher: input not full blocks")
	}
	for len(src) > 0 {
		block.Decrypt(dst, src[:bs])
		src = src[bs:]
		dst = dst[bs:]
	}
	out = Pkcs5UnPadding(out)
	return out, nil
}

/*******************AES***********************/
//AES-CBC key must 16bits
//if @block != nil use it , or new block
func AesEncrypt(block cipher.Block , orig []byte, key []byte) ([]byte , error) {
	var err error
	if block == nil {
		// 分组秘钥
		block, err = aes.NewCipher(key)
		if err != nil {
			return nil, errors.New("new failed!" + err.Error())
		}
	}
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 补全码
	orig = Pkcs5Padding(orig, blockSize)
	// 加密模式
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	// 创建数组
	cryted := make([]byte, len(orig))
	// 加密
	blockMode.CryptBlocks(cryted, orig)

	//return base64.StdEncoding.EncodeToString(cryted)
    return cryted , nil
}

//AES-CBC key must 16bits
//if @block != nil use it , or new block
func AesDecrypt(block cipher.Block , cryted []byte, key []byte) ([]byte  , error) {
	var err error
	if block == nil {
		// 分组秘钥
		block, err = aes.NewCipher(key)
		if err != nil {
			return nil, errors.New("new failed! err:" + err.Error())
		}
	}
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 加密模式
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	// 创建数组
	orig := make([]byte, len(cryted))
	// 解密
	blockMode.CryptBlocks(orig, cryted)
	// 去补全码
	orig = Pkcs5UnPadding(orig)
	return orig , nil
}

/*******************RSA***********************/
// 加密
func RsaEncrypt(origData []byte , publicKey []byte) ([]byte, error) {
	//解密pem格式的公钥
	block, _ := pem.Decode(publicKey)
	if block == nil {
		return nil, errors.New("public key error")
	}
	// 解析公钥
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	// 类型断言
	pub := pubInterface.(*rsa.PublicKey)
	//加密
	return rsa.EncryptPKCS1v15(rand.Reader, pub, origData)
}

// 解密
func RsaDecrypt(ciphertext []byte , privateKey []byte) ([]byte, error) {
	//解密
	block, _ := pem.Decode(privateKey)
	if block == nil {
		return nil, errors.New("private key error!")
	}
	//解析PKCS1格式的私钥
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	// 解密
	return rsa.DecryptPKCS1v15(rand.Reader, priv, ciphertext)
}