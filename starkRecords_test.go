package stark

import (
	"strings"
	"testing"

	starkcrypto "github.com/will-rowe/stark/src/crypto"
)

// TestRecord tests the record constructor and the UUID field encryption.
func TestRecord(t *testing.T) {

	// construct a record
	testAlias := "test label for a record"
	rec, err := NewRecord(SetAlias(testAlias), SetDescription(testDescription))
	if err != nil {
		t.Fatal(err)
	}
	if rec.GetAlias() != strings.ReplaceAll(testAlias, " ", "_") {
		t.Fatal("did not set alias for record")
	}
	originalUUID := rec.GetUuid()

	// get a cipher key
	cipherKey, err := starkcrypto.Password2cipherkey("some password")
	if err != nil {
		t.Fatal(err)
	}

	// encrypt
	if err := rec.Encrypt(cipherKey); err != nil {
		t.Fatal(err)
	}
	if rec.GetUuid() == originalUUID {
		t.Fatal("record UUID field was not encrypted")
	}

	// decrypt
	if err := rec.Decrypt(cipherKey); err != nil {
		t.Fatal(err)
	}
	if rec.GetUuid() != originalUUID {
		t.Fatal("record UUID field was not decrypted")
	}
}
