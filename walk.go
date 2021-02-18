// Package walk walks io/fs filesystems using an iterator style,
// as an alternative to the callback style of fs.WalkDir.
package walk

import (
	"errors"
	"io/fs"
	"path"
)

var errUsage = errors.New("walk: method Next must be called first")

// Walker provides a convenient interface for iterating over
// the descendants of a filesystem path.
// Successive calls to Next will step through
// each file or directory in the tree, including the root.
//
// The files are walked in lexical order, which makes the output
// deterministic but requires Walker to read an entire directory
// into memory before proceeding to walk that directory.
//
// Walker does not follow symbolic links found in directories,
// but if the root itself is a symbolic link, its target will be walked.
type Walker struct {
	fsys    fs.FS
	cur     visit
	stack   []visit
	descend bool
}

type visit struct {
	path    string
	info    fs.DirEntry
	err     error
	skipDir int
	skipPar int
}

// New returns a new Walker rooted at root on filesystem fsys.
func New(fsys fs.FS, root string) *Walker {
	info, err := fs.Stat(fsys, root)
	return &Walker{
		fsys:  fsys,
		cur:   visit{err: errUsage},
		stack: []visit{{root, infoDirEntry{info}, err, 0, 0}},
	}
}

// Next visits the next file or directory,
// which will then be available through the Path, Entry,
// and Err methods.
//
// Next must be called before each visit, including the first.
// It returns false when the walk stops at the end of the tree.
func (w *Walker) Next() bool {
	if w.descend && w.cur.err == nil && w.cur.info.IsDir() {
		dir, err := fs.ReadDir(w.fsys, w.cur.path)
		n := len(w.stack)
		for i := len(dir) - 1; i >= 0; i-- {
			p := path.Join(w.cur.path, dir[i].Name())
			w.stack = append(w.stack, visit{p, dir[i], nil, len(w.stack), n})
		}
		if err != nil {
			// Second visit, to report ReadDir error.
			w.cur.err = err
			w.stack = append(w.stack, w.cur)
		}
	}

	if len(w.stack) == 0 {
		w.descend = false
		return false
	}
	i := len(w.stack) - 1
	w.cur = w.stack[i]
	w.stack = w.stack[:i]
	w.descend = true
	return true
}

// Path returns the path to the most recent file or directory
// visited by a call to Next. It contains the root of w
// as a prefix; that is, if New is called with root "dir", which is
// a directory containing the file "a", Path will return "dir/a".
func (w *Walker) Path() string {
	return w.cur.path
}

// Entry returns the DirEntry for the most recent file or directory
// visited by a call to Next.
func (w *Walker) Entry() fs.DirEntry {
	return w.cur.info
}

// Err returns the error, if any, for the most recent attempt
// by Next to visit a file or directory.
//
// If a directory read has an error, it will be visited twice:
// the first visit is before the directory read is attempted
// and Err returns nil, giving the client a chance to call SkipDir
// and avoid the ReadDir entirely.
// The second visit is after a failed ReadDir and returns the error
// from ReadDir. (If ReadDir succeeds, there is no second visit.)
//
// It is possible for ReadDir to return entries along with an error.
// In that case, after the second visit reporting the error,
// w will walk through the entries returned by ReadDir.
// That set of entries may be incomplete because of the error.
// To avoid visiting them, call SkipDir.
func (w *Walker) Err() error {
	return w.cur.err
}

// SkipDir causes w not to walk through the directory named by Path.
// No directory read will be attempted on a skipped directory.
// If w is not on a directory, SkipDir skips nothing.
func (w *Walker) SkipDir() {
	w.descend = false
	w.stack = w.stack[:w.cur.skipDir]
}

// SkipParent causes w to skip the file or directory named by Path
// (like SkipDir)
// as well as any remaining items in its parent directory.
func (w *Walker) SkipParent() {
	w.descend = false
	w.stack = w.stack[:w.cur.skipPar]
}

type infoDirEntry struct{ f fs.FileInfo }

func (e infoDirEntry) Name() string               { return e.f.Name() }
func (e infoDirEntry) IsDir() bool                { return e.f.IsDir() }
func (e infoDirEntry) Type() fs.FileMode          { return e.f.Mode().Type() }
func (e infoDirEntry) Info() (fs.FileInfo, error) { return e.f, nil }
