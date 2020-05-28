package stark

import (
	"testing"
)

// TestRecord tests the record constructor.
func TestRecord(t *testing.T) {
	testAlias := "test label for a record"
	rec, err := NewRecord(SetAlias(testAlias), SetDescription(testDescription))
	if err != nil {
		t.Fatal(err)
	}
	if rec.GetAlias() != testAlias {
		t.Fatal("did not set alias for record")
	}

	t.Log(rec)
}
