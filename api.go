package tagger

import (
	"code.google.com/p/go-uuid/uuid"
	"errors"
	"io"
)

type (
	// StorageProvider specifies the interface that must be implemented by tag storage backends.
	StorageProvider interface {
		io.Closer

		GetFile(u uuid.UUID) (File, error)
		GetFileForPath(path string) (File, error)
		GetAllFiles() ([]File, error)
		GetMatchingFiles(f Filter) ([]File, error)

		UpdateTag(f File, t Tag) error
		RemoveTag(f File, t Tag) error
		GetTags(f File) ([]Tag, error)
		// GetAllTags() ([]Tag, error) // TODO: Reconsider this method. Maybe split into two? (tags, values)

		UpdateFile(f File, t []Tag) error
		RemoveFile(f File) error
	}

	// File is a structure that represents a file in the database
	File struct {
		uuid uuid.UUID
		path string
	}

	// Tag is an interface representing the needed methods on a tag
	Tag interface {
		Name() string
		HasValue() bool
		Value() int
	}

	// NamedTag is a tag with just a name
	NamedTag struct {
		name string
	}

	// ValueTag is a tag with both a name and a value
	ValueTag struct {
		name  string
		value int
	}
)

// NewFile creates a new file struct an populates it's fields
func NewFile(uuid_ uuid.UUID, path string) File {
	return File{uuid: uuid_, path: path}
}

// NewNamedTag creates a new NamedTag struct an populates it's fields
func NewNamedTag(name string) *NamedTag {
	return &NamedTag{name: name}
}

// NewValueTag creates a new ValueTag struct an populates it's fields
func NewValueTag(name string, value int) *ValueTag {
	return &ValueTag{name: name, value: value}
}

// UUID returns the UUID of a file
func (f File) UUID() uuid.UUID { return f.uuid }

// Path returns the path of a file
func (f File) Path() string { return f.path }

// Name returns the name of a tag
func (t NamedTag) Name() string { return t.name }

// Name returns the name of a tag
func (t ValueTag) Name() string { return t.name }

// HasValue returns whether the tag has a value
func (t NamedTag) HasValue() bool { return false }

// HasValue returns whether the tag has a value
func (t ValueTag) HasValue() bool { return true }

// Value returns -1 on a named tag
func (t NamedTag) Value() int { return -1 }

// Value returns the value of a value tag
func (t ValueTag) Value() int { return t.value }

// Errors
var (
	ErrNoFile       = errors.New("tagger: No such file in storage")
	ErrNoTag        = errors.New("tagger: No such tag on file")
	ErrNoMatches    = errors.New("tagger: No matching files in storage")
	ErrInvalidValue = errors.New("tagger: Invalid tag value")
)
