package filen

import (
	"testing"

	"github.com/FilenCloudDienste/filen-sdk-go/filen/types"
)

func TestNamesEqual(t *testing.T) {
	cases := []struct {
		a     string
		b     string
		equal bool
	}{
		{"photos", "photos", true},
		{"Photos", "photos", true},
		{"B.TXT", "b.txt", true},
		{"Äpfel", "äpfel", true},
		{"photos", "photo", false},
		{"a.txt", "b.txt", false},
	}
	for _, c := range cases {
		if NamesEqual(c.a, c.b) != c.equal {
			t.Errorf("NamesEqual(%q, %q) = %v, expected %v", c.a, c.b, !c.equal, c.equal)
		}
	}
}

func TestFindByName(t *testing.T) {
	dirs := []*types.Directory{
		{UUID: "1", Name: "photos"},
		{UUID: "2", Name: "Photos"},
		{UUID: "3", Name: "Music"},
	}

	t.Run("Exact match preferred", func(t *testing.T) {
		if dir := findByName(dirs, "Photos"); dir == nil || dir.UUID != "2" {
			t.Fatalf("expected directory 2, got %#v", dir)
		}
	})
	t.Run("Case-insensitive match", func(t *testing.T) {
		if dir := findByName(dirs, "music"); dir == nil || dir.UUID != "3" {
			t.Fatalf("expected directory 3, got %#v", dir)
		}
	})
	t.Run("First case-insensitive match", func(t *testing.T) {
		if dir := findByName(dirs, "PHOTOS"); dir == nil || dir.UUID != "1" {
			t.Fatalf("expected directory 1, got %#v", dir)
		}
	})
	t.Run("No match", func(t *testing.T) {
		if dir := findByName(dirs, "videos"); dir != nil {
			t.Fatalf("expected nil, got %#v", dir)
		}
	})
}
