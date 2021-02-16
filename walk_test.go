package walk_test

import (
	"errors"
	"io/fs"
	"reflect"
	"testing"
	"testing/fstest"

	"kr.dev/walk"
)

var tree = fstest.MapFS{
	"a/x": &fstest.MapFile{},
	"a/y": &fstest.MapFile{},
	"b":   &fstest.MapFile{},
}

var treeWithError = dirErrorFS{tree, map[string]error{
	"a": errors.New("incomplete readdir"),
}}

func TestWalk(t *testing.T) {
	tree := fstest.MapFS{
		"a":     &fstest.MapFile{},
		"b":     &fstest.MapFile{Mode: fs.ModeDir},
		"c":     &fstest.MapFile{},
		"d/x":   &fstest.MapFile{},
		"d/y":   &fstest.MapFile{Mode: fs.ModeDir},
		"d/z/u": &fstest.MapFile{},
		"d/z/v": &fstest.MapFile{},
	}

	var got []string
	want := []string{
		".",
		"a",
		"b",
		"c",
		"d",
		"d/x",
		"d/y",
		"d/z",
		"d/z/u",
		"d/z/v",
	}

	walker := walk.New(tree, ".")
	for walker.Next() {
		if err := walker.Err(); err != nil {
			t.Errorf("no error expected, found: %s", err)
		}
		got = append(got, walker.Path())
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("visited %v, want %v", got, want)
	}
}

func TestPartialReadDir(t *testing.T) {
	var got []string
	want := []string{
		".",
		"a",
		"a",   // report error reading a
		"a/x", // walk (possibly incomplete) contents of a
		"a/y",
		"b",
	}

	walker := walk.New(treeWithError, ".")
	for walker.Next() {
		got = append(got, walker.Path())
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestMutateAfterEnd(t *testing.T) {
	tree := fstest.MapFS{
		"a": &fstest.MapFile{Mode: fs.ModeDir},
	}

	walker := walk.New(tree, ".")
	for walker.Next() {
	}
	tree["a/b"] = &fstest.MapFile{}
	for walker.Next() {
		t.Errorf("Next returned true after false: %s", walker.Path())
	}
}

// Three SkipDir cases:
//   1. Visiting a regular file
//   2. Visiting a directory (first visit, to skip read)
//   3. Visiting a directory on error (with partial results)

func TestSkipDirOnFile(t *testing.T) {
	var got []string
	want := []string{".", "a", "a/x", "a/y", "b"}

	walker := walk.New(tree, ".")
	for walker.Next() {
		got = append(got, walker.Path())
		if walker.Path() == "a/x" {
			walker.SkipDir()
		}
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSkipDirOnPreVisit(t *testing.T) {
	var got []string
	want := []string{".", "a", "b"}

	walker := walk.New(tree, ".")
	for walker.Next() {
		got = append(got, walker.Path())
		if walker.Path() == "a" {
			walker.SkipDir()
		}
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSkipDirOnDirError(t *testing.T) {
	var got []string
	want := []string{".", "a", "a", "b"}

	walker := walk.New(treeWithError, ".")
	for walker.Next() {
		got = append(got, walker.Path())
		if walker.Path() == "a" && walker.Err() != nil {
			walker.SkipDir()
		}
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// Three SkipParent cases:
//   1. Visiting a regular file
//   2. Visiting a directory (first visit, to skip read)
//   3. Visiting a directory on error (with partial results)

func TestSkipParentOnFile(t *testing.T) {
	var got []string
	want := []string{".", "a", "a/x", "b"}

	walker := walk.New(tree, ".")
	for walker.Next() {
		got = append(got, walker.Path())
		if walker.Path() == "a/x" {
			walker.SkipParent()
		}
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSkipParentOnPreVisit(t *testing.T) {
	var got []string
	want := []string{".", "a"}

	walker := walk.New(tree, ".")
	for walker.Next() {
		got = append(got, walker.Path())
		if walker.Path() == "a" {
			walker.SkipParent()
		}
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSkipParentOnDirError(t *testing.T) {
	var got []string
	want := []string{".", "a", "a"}

	walker := walk.New(treeWithError, ".")
	for walker.Next() {
		got = append(got, walker.Path())
		if walker.Path() == "a" && walker.Err() != nil {
			walker.SkipParent()
		}
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

type dirErrorFS struct {
	fstest.MapFS
	errors map[string]error
}

func (fsys dirErrorFS) ReadDir(name string) ([]fs.DirEntry, error) {
	ents, err := fsys.MapFS.ReadDir(name)
	if err != nil {
		return ents, err // return the real error, if any
	}
	return ents, fsys.errors[name] // return the fake error, if any
}
