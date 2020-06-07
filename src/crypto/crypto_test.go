package crypto

import (
	"testing"
)

var (
	password = "tonystark"
	data     = "I am Iron Man"
)

// TestCipherKey will test the key generator.
func TestCipherKey(t *testing.T) {
	if _, err := Password2cipherkey(password); err != nil {
		t.Fatal(err)
	}
	if _, err := Password2cipherkey(""); err == nil {
		t.Fatal("generated cipher key from empty password")
	}
}

// TestEncryption will test encyption and decryption.
func TestEncryption(t *testing.T) {

	// get the cipher key
	key, err := Password2cipherkey(password)
	if err != nil {
		t.Fatal(err)
	}

	// test encryption
	eData, err := Encrypt(data, key)
	if err != nil {
		t.Fatal(err)
	}

	// test decryption
	dData, err := Decrypt(eData, key)
	if err != nil {
		t.Fatal(err)
	}

	// test retrieval
	t.Log(string(eData), string(dData))
	if string(data) != string(dData) {
		t.Fatal("could not decrypt data")
	}
}
