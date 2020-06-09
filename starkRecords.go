package stark

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/uuid"
	starkcrypto "github.com/will-rowe/stark/src/crypto"
	"google.golang.org/protobuf/proto"
)

// SetAlias is an option setter for the NewRecord constructor
// that sets the human readable label of a Record.
func SetAlias(alias string) RecordOption {
	return func(x *Record) error {
		return x.setAlias(alias)
	}
}

// SetDescription is an option setter for the NewRecord constructor
// that sets the description of a Record.
func SetDescription(description string) RecordOption {
	return func(x *Record) error {
		return x.setDescription(description)
	}
}

// NewComment creates a comment.
func NewComment(comment, prevCID string) *RecordComment {
	return &RecordComment{
		Timestamp:   ptypes.TimestampNow(),
		Text:        comment,
		PreviousCID: prevCID,
	}
}

// NewRecord creates a record.
func NewRecord(options ...RecordOption) (*Record, error) {

	// create the base record
	x := &Record{
		Uuid:            uuid.New().String(),
		PreviousCID:     "",
		History:         []*RecordComment{},
		LinkedSamples:   make(map[string]string),
		LinkedLibraries: make(map[string]string),
		Barcodes:        make(map[string]int32),
	}

	// start the Record history
	x.AddComment("record created.")

	// add the user provided options
	for _, option := range options {
		err := option(x)
		if err != nil {
			return nil, err
		}
	}

	return x, nil
}

// NewRecordFromReader creates a Record from a reader.
// Accepts either JSON or Protobuf encoded Record.
func NewRecordFromReader(data io.Reader, ienc string) (*Record, error) {
	x := &Record{}
	switch ienc {
	case "json":
		um := &jsonpb.Unmarshaler{}
		if err := um.Unmarshal(data, x); err != nil {
			return nil, err
		}
	case "proto":
		buf, err := ioutil.ReadAll(data)
		if err != nil {
			return nil, err
		}
		if err := proto.Unmarshal(buf, x); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported format: %s", ienc)
	}
	return x, nil
}

// AddComment adds a timestamped comment to the Record's history, along with the last known CID to enable rollbacks.
func (x *Record) AddComment(text string) {
	x.History = append(x.History, NewComment(text, x.PreviousCID))
	return
}

// LinkSample links a Sample to a Record.
func (x *Record) LinkSample(sampleUUID uuid.UUID, sampleLocation string) error {

	// convert the UUID to string and check if it's already linked
	uuid := sampleUUID.String()
	if _, exists := x.LinkedSamples[uuid]; exists {
		return ErrLinkExists
	}

	// link it
	x.LinkedSamples[uuid] = sampleLocation
	x.AddComment(fmt.Sprintf("linked record to sample (%s)", uuid))
	return nil
}

// LinkLibrary links a Library to a Record.
func (x *Record) LinkLibrary(libraryUUID uuid.UUID, libraryLocation string) error {

	// convert the UUID to string and check if it's already linked
	uuid := libraryUUID.String()
	if _, exists := x.LinkedLibraries[uuid]; exists {
		return ErrLinkExists
	}

	// link it
	x.LinkedLibraries[uuid] = libraryLocation
	x.AddComment(fmt.Sprintf("linked record to library (%s)", uuid))
	return nil
}

// GetCreatedTimestamp returns the timestamp for when the record was created.
func (x *Record) GetCreatedTimestamp() *timestamp.Timestamp {
	return x.GetHistory()[0].Timestamp
}

// Encrypt will encrypt certain fields of a Record.
//
// Note: Currently only the Record UUID is encrypted.
func (x *Record) Encrypt(cipherKey []byte) error {
	if x.Encrypted {
		return ErrEncrypted
	}

	// encrypt the UUID
	encField, err := starkcrypto.Encrypt(x.Uuid, cipherKey)
	if err != nil {
		return err
	}
	x.Uuid = encField

	// TODO: encrypt other fields

	// set the Record to encrypted
	x.Encrypted = true
	return nil
}

// Decrypt will decrypt certain fields of a Record.
// Unencrypted Records are ignored and errors are
// reported for unsuccessful decrypts.
//
// Note: Currently only the Record UUID is decrypted.
func (x *Record) Decrypt(cipherKey []byte) error {
	if !x.Encrypted {
		return nil
	}

	// decrypt the UUID
	decField, err := starkcrypto.Decrypt(x.Uuid, cipherKey)
	if err != nil {
		return err
	}
	x.Uuid = decField

	// TODO: decrypt other fields

	// set the Record to decrypted
	x.Encrypted = false
	return nil
}

// GetLastUpdatedTimestamp returns the timestamp for when the record was created.
func (x *Record) GetLastUpdatedTimestamp() *timestamp.Timestamp {
	histLength := len(x.GetHistory())
	return x.GetHistory()[histLength-1].Timestamp
}

func (x *Record) setAlias(alias string) error {
	if len(alias) != 0 {
		x.Alias = alias
		x.AddComment("alias updated.")
	}
	return nil
}

func (x *Record) setDescription(description string) error {
	if len(description) != 0 {
		x.Description = description
		x.AddComment("description updated.")
	}
	return nil
}
