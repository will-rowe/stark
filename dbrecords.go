package stark

import (
	"fmt"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/uuid"
)

// RecOption is a wrapper struct used to pass functional
// options to the Record constructor.
type RecOption func(Record *Record) error

// SetAlias is an option setter for the NewRecord constructor
// that sets the human readable label of a Record.
func SetAlias(alias string) RecOption {
	return func(rec *Record) error {
		return rec.setAlias(alias)
	}
}

// SetDescription is an option setter for the NewRecord constructor
// that sets the description of a Record.
func SetDescription(description string) RecOption {
	return func(rec *Record) error {
		return rec.setDescription(description)
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
func NewRecord(options ...RecOption) (*Record, error) {

	// create the base record
	rec := &Record{
		Uuid:            uuid.New().String(),
		PreviousCID:     "",
		History:         []*RecordComment{},
		LinkedSamples:   make(map[string]string),
		LinkedLibraries: make(map[string]string),
		Barcodes:        make(map[string]int32),
	}

	// start the Record history
	rec.AddComment("record created.")

	// add the user provided options
	for _, option := range options {
		err := option(rec)
		if err != nil {
			return nil, err
		}
	}

	return rec, nil
}

// AddComment adds a timestamped comment to the Record's history, along with the last known CID to enable rollbacks.
func (rec *Record) AddComment(text string) {
	rec.History = append(rec.History, NewComment(text, rec.PreviousCID))
	return
}

// LinkSample links a Sample to a Record.
func (rec *Record) LinkSample(sampleUUID uuid.UUID, sampleLocation string) error {

	// convert the UUID to string and check if it's already linked
	uuid := sampleUUID.String()
	if _, exists := rec.LinkedSamples[uuid]; exists {
		return ErrLinkExists
	}

	// link it
	rec.LinkedSamples[uuid] = sampleLocation
	rec.AddComment(fmt.Sprintf("linked record to sample (%s)", uuid))
	return nil
}

// LinkLibrary links a Library to a Record.
func (rec *Record) LinkLibrary(libraryUUID uuid.UUID, libraryLocation string) error {

	// convert the UUID to string and check if it's already linked
	uuid := libraryUUID.String()
	if _, exists := rec.LinkedLibraries[uuid]; exists {
		return ErrLinkExists
	}

	// link it
	rec.LinkedLibraries[uuid] = libraryLocation
	rec.AddComment(fmt.Sprintf("linked record to library (%s)", uuid))
	return nil
}

// GetCreatedTimestamp returns the timestamp for when the record was created.
func (rec *Record) GetCreatedTimestamp() *timestamp.Timestamp {
	return rec.GetHistory()[0].Timestamp
}

// GetLastUpdatedTimestamp returns the timestamp for when the record was created.
func (rec *Record) GetLastUpdatedTimestamp() *timestamp.Timestamp {
	histLength := len(rec.GetHistory())
	return rec.GetHistory()[histLength-1].Timestamp
}

func (rec *Record) setAlias(alias string) error {
	rec.Alias = alias
	rec.AddComment("alias updated.")
	return nil
}

func (rec *Record) setDescription(description string) error {
	rec.Description = description
	rec.AddComment("description updated.")
	return nil
}
