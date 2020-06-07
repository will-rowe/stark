//Package crypto is used to encrypt and decrypt string data using symmetric key encryption.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

const (

	// DefaultCipherKeyLength is the required number of bytes for a cipher key.
	DefaultCipherKeyLength = 32
)

var (

	// ErrCipherKeyMissing is issued when an encrypt/decrypt needed but we don't have a cipher key.
	ErrCipherKeyMissing = fmt.Errorf("no cipher key provided")

	// ErrCipherKeyLength is issued when a key is not long enough.
	ErrCipherKeyLength = fmt.Errorf("cipher key must be %d bytes", DefaultCipherKeyLength)

	// ErrCipherPassword is issued when a cipher key cannot be generated from the provided password.
	ErrCipherPassword = fmt.Errorf("cannot generate cipher key from provided password")
)

// CipherKeyCheck will check the key meets
// starkdb requirements.
func CipherKeyCheck(key []byte) error {
	if len(key) == 0 {
		return ErrCipherKeyMissing
	}
	if len(key) != DefaultCipherKeyLength {
		return ErrCipherKeyLength
	}
	return nil
}

// Password2cipherkey will take a password and produce
// a 32 byte cipher key.
func Password2cipherkey(password string) ([]byte, error) {
	if len(password) == 0 {
		return nil, ErrCipherPassword
	}
	hasher := md5.New()
	if _, err := hasher.Write([]byte(password)); err != nil {
		return nil, err
	}
	key := []byte(hex.EncodeToString(hasher.Sum(nil)))
	if err := CipherKeyCheck(key); err != nil {
		return nil, err
	}
	return key, nil
}

// Encrypt will encrypt plaintext using symmetric key encryption.
func Encrypt(data string, cipherKey []byte) (string, error) {

	// check the key
	if err := CipherKeyCheck(cipherKey); err != nil {
		return "", err
	}

	// prepare the cipher
	block, err := aes.NewCipher(cipherKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// encode
	ciphertextByte := gcm.Seal(nonce, nonce, []byte(data), nil)
	return base64.StdEncoding.EncodeToString(ciphertextByte), err

}

// Decrypt will decrypt plaintext using symmetric key encryption.
func Decrypt(data string, cipherKey []byte) (string, error) {

	// check the key
	if err := CipherKeyCheck(cipherKey); err != nil {
		return "", err
	}

	// prepare cipher
	block, err := aes.NewCipher(cipherKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()

	// decode
	ciphertextByte, _ := base64.StdEncoding.DecodeString(data)
	nonce, ciphertextByteClean := ciphertextByte[:nonceSize], ciphertextByte[nonceSize:]
	plaintextByte, err := gcm.Open(nil, nonce, ciphertextByteClean, nil)
	if err != nil {
		return "", err
	}
	return string(plaintextByte), nil
}
