package ingestion

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// Clone clones the repository at remoteURL into a temporary directory.
// The returned cleanup function removes the temporary directory.
// Callers must call cleanup() when done with the cloned repo.
func Clone(ctx context.Context, remoteURL string) (dir string, cleanup func(), err error) {
	tmp, err := os.MkdirTemp("", "repo-mri-*")
	if err != nil {
		return "", nil, fmt.Errorf("ingestion clone: create temp dir: %w", err)
	}

	cleanup = func() {
		_ = os.RemoveAll(tmp)
	}

	//nolint:gosec // remoteURL is validated by the caller as a https:// URL
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", remoteURL, tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("ingestion clone: git clone %s: %w\n%s", remoteURL, err, out)
	}

	return tmp, cleanup, nil
}
