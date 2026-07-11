package core

import "fmt"

// schemaReleases maps a document schema version (PRAGMA user_version) to the
// earliest Financy release that can open a file at that schema. A release can
// open any file whose schema is <= its own latest schema (older files are
// migrated up on open; newer files are refused — see ErrFileTooNew). So the
// value here is the *floor*: the first release whose schema reached this level.
// A backup written at schema v can therefore be opened by this release or any
// later one — older releases predate the format.
//
// Keep this in sync when adding migrations to db.go: a new migration bumps the
// schema, so add an entry for it pointing at the release that ships it.
var schemaReleases = map[int]string{
	1: "v0.1.0",
	2: "v0.4.0",
	3: "v0.5.0",
	4: "v0.5.0",
	5: "v0.6.0",
	6: "v0.6.0",
	7: "v0.6.0",
	8: "v0.8.0",
	9: "v0.13.0",
}

// BackupSuffix returns the filename suffix for a pre-migration backup of a file
// at the given schema version, e.g. BackupSuffix(6) == ".v6.bak". Appending it
// to the document path keeps the schema visible in the backup's name so a user
// knows which Financy version still reads it.
func BackupSuffix(schema int) string {
	return fmt.Sprintf(".v%d.bak", schema)
}

// ReleaseForVersion returns the earliest Financy release that can open a file at
// the given schema version, or "" if the schema is unknown to this build.
func ReleaseForVersion(schema int) string {
	return schemaReleases[schema]
}
