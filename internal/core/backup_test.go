package core

import "testing"

func TestBackupSuffix(t *testing.T) {
	cases := map[int]string{
		1: ".v1.bak",
		6: ".v6.bak",
		7: ".v7.bak",
	}
	for schema, want := range cases {
		if got := BackupSuffix(schema); got != want {
			t.Errorf("BackupSuffix(%d) = %q, want %q", schema, got, want)
		}
	}
}

func TestReleaseForVersion(t *testing.T) {
	known := map[int]string{
		1: "v0.1.0",
		2: "v0.4.0",
		4: "v0.5.0",
		7: "v0.6.0",
	}
	for schema, want := range known {
		if got := ReleaseForVersion(schema); got != want {
			t.Errorf("ReleaseForVersion(%d) = %q, want %q", schema, got, want)
		}
	}

	// Every shipped schema (up to the current build) must map to a release so
	// the downgrade hint is never blank for a real backup.
	for v := 1; v <= LatestVersion(); v++ {
		if ReleaseForVersion(v) == "" {
			t.Errorf("ReleaseForVersion(%d) is empty; add it to schemaReleases", v)
		}
	}

	// Unknown schemas fall back to "".
	for _, v := range []int{0, -1, 99} {
		if got := ReleaseForVersion(v); got != "" {
			t.Errorf("ReleaseForVersion(%d) = %q, want \"\"", v, got)
		}
	}
}
