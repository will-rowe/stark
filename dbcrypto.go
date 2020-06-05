package stark

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"io"
)

// cipherKeyCheck will check the key meets
// starkdb requirements.
func cipherKeyCheck(key []byte) error {
	if len(key) == 0 {
		return ErrCipherKeyMissing
	}
	if len(key) != DefaultCipherKeyLength {
		return ErrCipherKeyLength
	}
	return nil
}

// password2cipherkey will take a password and produce
// a 32 byte cipher key.
func password2cipherkey(password string) ([]byte, error) {
	if len(password) == 0 {
		return nil, ErrCipherPassword
	}
	hasher := md5.New()
	if _, err := hasher.Write([]byte(password)); err != nil {
		return nil, err
	}
	key := []byte(hex.EncodeToString(hasher.Sum(nil)))
	if err := cipherKeyCheck(key); err != nil {
		return nil, err
	}
	return key, nil
}

// encrypt will encrypt plaintext using symmetric key encryption.
func encrypt(data string, cipherKey []byte) (string, error) {

	// check the key
	if err := cipherKeyCheck(cipherKey); err != nil {
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

// decrypt will decrypt plaintext using symmetric key encryption.
func decrypt(data string, cipherKey []byte) (string, error) {

	// check the key
	if err := cipherKeyCheck(cipherKey); err != nil {
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
