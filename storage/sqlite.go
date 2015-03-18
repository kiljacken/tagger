package storage

import (
	"code.google.com/p/go-uuid/uuid"
	"database/sql"
	"github.com/kiljacken/tagger"
	// Black import of go-sqlite3 to ensure the database engine is available
	_ "github.com/mattn/go-sqlite3"
	"log"
)

// SqliteStorage implents a tagger.StorageProvide backed by a sqlite database
type SqliteStorage struct {
	db *sql.DB
}

// NewSqliteStorage returns a new storage engine backed by an in memory sqlite database
func NewSqliteStorage(descriptor string) (*SqliteStorage, error) {
	// Open up a sqlite memory connection
	db, err := sql.Open("sqlite3", descriptor)
	if err != nil {
		// If an error occurs, returns this error
		return nil, err
	}

	// Create a empty sqlite storage struct, and store the db connection in it
	storage := new(SqliteStorage)
	storage.db = db

	// Setup database tables
	storage.init()

	// Return the new storage engine
	return storage, nil
}

func (s *SqliteStorage) init() {
	setupStmt := `
	PRAGMA foreign_keys = ON;
	`

	// Setup database settings
	_, err := s.db.Exec(setupStmt)
	if err != nil {
		// If an error occurs die with an error message
		log.Fatal(err)
	}

	tableStmt := `
	CREATE TABLE IF NOT EXISTS file(
		uuid TEXT NOT NULL,
		path TEXT,
		PRIMARY KEY (uuid)
		UNIQUE(path) ON CONFLICT REPLACE
	);
	CREATE TABLE IF NOT EXISTS tags(
		uuid TEXT NOT NULL,
		name TEXT NOT NULL,
		value INTEGER,
		FOREIGN KEY(uuid) REFERENCES file(uuid)
		PRIMARY KEY (uuid, name)	
	);
	`
	/*
		CREATE TABLE named_tags(
			uuid TEXT NOT NULL,
			name TEXT NOT NULL,
			FOREIGN KEY(uuid) REFERENCES file(uuid)
			PRIMARY KEY (uuid, name)
		);
		CREATE TABLE value_tags(
			uuid TEXT NOT NULL,
			name TEXT NOT NULL,
			value INTEGER NOT NULL,
			FOREIGN KEY(uuid) REFERENCES file(uuid)
			PRIMARY KEY (uuid, name)
		);
	*/

	// Setup database tables
	_, err = s.db.Exec(tableStmt)
	if err != nil {
		// If an error occurs die with an error message
		log.Fatal(err)
	}
}

// Close closes alle resources associated with the storage provider
func (s *SqliteStorage) Close() error {
	return s.db.Close()
}

const getFileStmt = `SELECT * FROM file WHERE uuid = ?`

// GetFile returns the file matching the provided UUID.
func (s *SqliteStorage) GetFile(u uuid.UUID) (tagger.File, error) {
	// Prepare the statement
	st, err := s.db.Prepare(getFileStmt)
	if err != nil {
		// If we get an error here its due to programmer error
		log.Fatal(err)
	}
	defer st.Close()

	// Fetch the row with the file
	row := st.QueryRow(u.String())

	// Get the values from the row
	var rowUUID, path sql.NullString
	err = row.Scan(&rowUUID, &path)
	if err == sql.ErrNoRows {
		// If no row was found, no such file exists
		return tagger.File{}, tagger.ErrNoFile
	} else if err != nil {
		// If another error occurs return it
		return tagger.File{}, err
	}

	// Construct a file struct and return it
	return tagger.NewFile(uuid.Parse(rowUUID.String), path.String), nil
}

const getFileForPathStmt = `SELECT * FROM file WHERE path = ?`

// GetFileForPath returns the file at the given path.
func (s *SqliteStorage) GetFileForPath(path string) (tagger.File, error) {
	// Prepare the statement
	st, err := s.db.Prepare(getFileForPathStmt)
	if err != nil {
		// If we get an error here its due to programmer error
		log.Fatal(err)
	}
	defer st.Close()

	// Fetch the row with the file
	row := st.QueryRow(path)

	// Get the values from the row
	var rowUUID, rowPath sql.NullString
	err = row.Scan(&rowUUID, &rowPath)
	if err == sql.ErrNoRows {
		// If no row was found, no such file exists
		return tagger.File{}, tagger.ErrNoFile
	} else if err != nil {
		// If another error occurs return it
		return tagger.File{}, err
	}

	// Construct a file struct and return it
	return tagger.NewFile(uuid.Parse(rowUUID.String), rowPath.String), nil
}

const getAllFilesStmt = `SELECT * FROM file`

// GetAllFiles returns a slice containing all files in the storage provider
func (s *SqliteStorage) GetAllFiles() ([]tagger.File, error) {
	// Prepare the statement
	st, err := s.db.Prepare(getAllFilesStmt)
	if err != nil {
		// If we get an error here its due to programmer error
		log.Fatal(err)
	}
	defer st.Close()

	// Fetch the row with the file
	rows, err := st.Query()
	if err != nil {
		// An error shouldn't happen here according to docs.
		// If no row was found row.Scan will return ErrNoRow.
		return nil, err
	}
	defer rows.Close()

	// Create an empty array of files
	var files []tagger.File

	// Loop through each row in the query
	for rows.Next() {
		// Get the values from the row
		var rowUUID, path sql.NullString
		err = rows.Scan(&rowUUID, &path)
		if err != nil {
			// If an error occured, return the error
			return nil, err
		}

		files = append(files, tagger.NewFile(uuid.Parse(rowUUID.String), path.String))
	}

	// If an error occured during the query, return the error
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	// Return the array of files
	return files, nil
}

// GetMatchingFiles returns all files from the storage provider that matches
// the provided filter.
func (s *SqliteStorage) GetMatchingFiles(f tagger.Filter) ([]tagger.File, error) {
	// XXX: This is really bad practice. Database engines should make optimized
	// sql statements for filtering.
	var matches []tagger.File

	// Get ALL files
	files, err := s.GetAllFiles()
	if err != nil {
		return nil, err
	}

	// Loop through ALL files
	for _, file := range files {
		// Get the files tags
		tags, err := s.GetTags(file)
		if err != nil {
			// TODO: We fail fast now, maybe try other files first?
			return nil, err
		}

		// Add file to result only if it's tags match the filter
		if f.Matches(tags) {
			matches = append(matches, file)
		}
	}

	return matches, nil
}

const updateTagStmt = `INSERT OR REPLACE INTO tags (uuid, name, value) VALUES (?, ?, ?)`

// UpdateTag updates a tag on the file or creates it if it doesn't exist.
func (s *SqliteStorage) UpdateTag(f tagger.File, t tagger.Tag) error {
	// Prepare the statement
	st, err := s.db.Prepare(updateTagStmt)
	if err != nil {
		// If we get an error here its due to programmer error
		log.Fatal(err)
	}
	defer st.Close()

	if t.HasValue() {
		// If the tag has a value, update with value
		_, err = st.Exec(f.UUID().String(), t.Name(), t.Value())
	} else {
		// If the tag doesn't have a value, update value to NULL
		_, err = st.Exec(f.UUID().String(), t.Name(), nil)
	}

	// If an error occurs, return it
	if err != nil {
		return err
	}

	return nil
}

const removeTagStmt = `DELETE FROM tags WHERE uuid = ? AND name = ?`

// RemoveTag removes a tag from a file.
func (s *SqliteStorage) RemoveTag(f tagger.File, t tagger.Tag) error {
	// Prepare the statement
	st, err := s.db.Prepare(removeTagStmt)
	if err != nil {
		// If we get an error here its due to programmer error
		log.Fatal(err)
	}
	defer st.Close()

	// Execute the statement
	_, err = st.Exec(f.UUID().String(), t.Name())

	// If an error occurs, return it
	if err != nil {
		return err
	}

	return nil
}

const getTagsStmt = `SELECT name, value FROM tags WHERE uuid = ?`

// GetTags gets all tags associated with a file
func (s *SqliteStorage) GetTags(f tagger.File) ([]tagger.Tag, error) {
	// Prepare the statement
	st, err := s.db.Prepare(getTagsStmt)
	if err != nil {
		// If we get an error here its due to programmer error
		log.Fatal(err)
	}
	defer st.Close()

	// Execute the query
	rows, err := st.Query(f.UUID().String())
	if err != nil {
		// An error shouldn't happen here according to docs.
		// If no row was found row.Scan will return ErrNoRow.
		return nil, err
	}
	defer rows.Close()

	// Create an empty array of tags
	var tags []tagger.Tag

	// Loop through each row in the query
	for rows.Next() {
		// Get the values from the row
		var name sql.NullString
		var value sql.NullInt64
		err = rows.Scan(&name, &value)
		if err != nil {
			// If an error occured, return the error
			return nil, err
		}

		// Depending on if we have a value, create a value tag or a name tag
		var tag tagger.Tag
		if value.Valid {
			tag = tagger.NewValueTag(name.String, int(value.Int64))
		} else {
			tag = tagger.NewNamedTag(name.String)
		}

		// Add the tag to our array
		tags = append(tags, tag)
	}

	// If an error occured during the query, return the error
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	// Return the array of tags
	return tags, nil
}

const updateFileStmt = `INSERT OR REPLACE INTO file (uuid, path) VALUES (?, ?)`

// UpdateFile updates all files associated with the provided file. If the file
// doesn't exist in the storage provider, it is created.
func (s *SqliteStorage) UpdateFile(f tagger.File, t []tagger.Tag) error {
	// Prepare the statement
	st, err := s.db.Prepare(updateFileStmt)
	if err != nil {
		// If we get an error here its due to programmer error
		log.Fatal(err)
	}
	defer st.Close()

	// If the tag has a value, update with value
	_, err = st.Exec(f.UUID().String(), f.Path())
	// If an error occurs, return it
	if err != nil {
		return err
	}

	// For each tag associated with file, update the tag.
	for _, tag := range t {
		err := s.UpdateTag(f, tag)
		// If an error occurs return it
		if err != nil {
			return err
		}
	}

	return nil
}

const removeFileStmt = `DELETE FROM file WHERE uuid = ?`

// RemoveFile removes a file from the storage provider
func (s *SqliteStorage) RemoveFile(f tagger.File) error {
	// Loop through all tags associated with the file and remove them
	tags, err := s.GetTags(f)
	if err != nil {
		return err
	}

	for _, tag := range tags {
		err := s.RemoveTag(f, tag)
		if err != nil {
			return err
		}
	}

	// Prepare the statement
	st, err := s.db.Prepare(removeFileStmt)
	if err != nil {
		// If we get an error here its due to programmer error
		log.Fatal(err)
	}
	defer st.Close()

	// Execute the query
	_, err = st.Exec(f.UUID().String())
	if err != nil {
		return err
	}

	return nil
}
