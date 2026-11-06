package main

import "testing"

func TestFormatVersion(t *testing.T) {
	tests := []struct {
		name  string
		v     string
		rev   string
		dirty string
		want  string
	}{
		{name: "release tag", v: "v0.1.0", rev: "abcdef1234567890", want: "quest v0.1.0 (abcdef123456)"},
		{name: "module version", v: "v0.0.0-20261101134705-07fdf5c4dcd9", rev: "07fdf5c4dcd9", want: "quest v0.0.0-20261101134705-07fdf5c4dcd9 (07fdf5c4dcd9)"},
		{name: "local build", v: "(devel)", want: "quest devel"},
		{name: "dirty tree", v: "(devel)", rev: "abcdef123456", dirty: "-dirty", want: "quest devel (abcdef123456-dirty)"},
		{name: "module dirty suffix", v: "v0.1.0+dirty", rev: "abcdef1", want: "quest v0.1.0 (abcdef1)"},
		{name: "unknown", want: "quest devel"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatVersion(tt.v, tt.rev, tt.dirty); got != tt.want {
				t.Fatalf("formatVersion(%q, %q, %q) = %q, want %q", tt.v, tt.rev, tt.dirty, got, tt.want)
			}
		})
	}
}
