package util

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/andeya/goutil"
)

func MD5(s string) string {
	md5New := md5.New()
	md5New.Write([]byte(s))
	b0 := md5New.Sum(nil)[:]
	return hex.EncodeToString(b0)
}

// Crypto ...
type Crypto struct {
	key string
}

// NewCrypto ...
func NewCrypto(key string) *Crypto {
	return &Crypto{key: key}
}

// Key ...
func (c *Crypto) Key() string {
	c.key = strings.TrimSpace(strings.Replace(c.key, "-", "", -1))
	return c.key
}

// EncryptCBC ...
func (c *Crypto) EncryptCBC(val string) string {
	return string(goutil.AESCBCEncrypt([]byte(c.Key()), []byte(val)))
}

// DecryptCBC ...
func (c *Crypto) DecryptCBC(ciphertext string) (string, error) {
	v, err := goutil.AESCBCDecrypt([]byte(c.Key()), []byte(ciphertext))
	if err != nil {
		return "", err
	}
	return string(v), nil
}

// Encrypt ...
func (c *Crypto) Encrypt(val string) string {
	return string(goutil.AESEncrypt([]byte(c.Key()), []byte(val)))
}

// Decrypt ...
func (c *Crypto) Decrypt(ciphertext string) (string, error) {
	v, err := goutil.AESDecrypt([]byte(c.Key()), []byte(ciphertext))
	if err != nil {
		return "", err
	}
	return string(v), nil
}

const key = "3f756b58-1656-11ec-879b-3c7d0a0ab31b"

var innerCrypto = NewCrypto(TrimKey(key)[:32])

// SetInnerCrypto ... must replace your own crypto
func SetInnerCrypto(k string) {
	innerCrypto = NewCrypto(TrimKey(k)[:32])
}

func TrimKey(k string) string {
	return strings.TrimSpace(strings.Replace(k, "-", "", -1))
}

// EncryptPassword ...
func EncryptPassword(p string) string {
	return strings.ToUpper(innerCrypto.Encrypt(p))
}

// DecryptPassword ...
func DecryptPassword(p string) (s string, err error) {
	return innerCrypto.Decrypt(strings.ToLower(p))
}

// CheckPassword ...c cipher_password
// p plaintext
func CheckPassword(p, c string) error {
	s, err := DecryptPassword(c)
	if err != nil {
		return err
	}
	if s != p {
		return ErrPassword
	}
	return nil
}

var ErrPassword = errors.New("密码错误")
