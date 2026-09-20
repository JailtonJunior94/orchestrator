package taskloop

import (
	"errors"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

var ErrTasksFileLockTimeout = errors.New("taskloop: tasks.md lock timeout")

func withTasksFileLock(fsys fs.FileSystem, tasksFile string, fn func() error) error {
	if _, ok := fsys.(*fs.OSFileSystem); !ok {
		return fn()
	}
	release, err := acquireTasksFileLock(tasksFile)
	if err != nil {
		return err
	}
	defer func() { _ = release() }()
	return fn()
}
