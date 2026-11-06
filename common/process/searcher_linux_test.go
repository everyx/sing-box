//go:build linux

package process

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	F "github.com/sagernet/sing/common/format"

	"github.com/stretchr/testify/require"
)

func TestTrimDeletedPathSuffix(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "not deleted",
			path:     "/usr/bin/Telegram",
			expected: "/usr/bin/Telegram",
		},
		{
			name:     "deleted",
			path:     "/usr/bin/Telegram (deleted)",
			expected: "/usr/bin/Telegram",
		},
		{
			name:     "deleted once",
			path:     "/usr/bin/foo (deleted) (deleted)",
			expected: "/usr/bin/foo (deleted)",
		},
		{
			name:     "suffix in name",
			path:     "/usr/bin/foo(deleted)",
			expected: "/usr/bin/foo(deleted)",
		},
		{
			name:     "empty",
			path:     "",
			expected: "",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.expected, TrimDeletedPathSuffix(testCase.path))
		})
	}
}

const processPathHelperEnvironment = "SING_BOX_TEST_PROCESS_PATH_HELPER"

// TestReadProcessPathAfterUnlink reproduces a package upgrade replacing the
// executable of a running process: the kernel then reports the path with a
// " (deleted)" suffix, which must not reach rule matching.
func TestReadProcessPathAfterUnlink(t *testing.T) {
	executablePath, err := os.Executable()
	require.NoError(t, err)
	targetPath := filepath.Join(t.TempDir(), "Telegram")
	executableContent, err := os.ReadFile(executablePath)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(targetPath, executableContent, 0o755))

	command := exec.Command(targetPath, "-test.run=^TestProcessPathHelperProcess$")
	command.Env = append(os.Environ(), processPathHelperEnvironment+"=1")
	require.NoError(t, command.Start())
	t.Cleanup(func() {
		_ = command.Process.Kill()
		_ = command.Wait()
	})
	require.NoError(t, os.Remove(targetPath))

	procPath := F.ToString(pathProc, "/", command.Process.Pid)
	require.Eventually(t, func() bool {
		processPath, readErr := readProcessPath(procPath)
		return readErr == nil && processPath == targetPath
	}, 10*time.Second, 10*time.Millisecond, "process path should not contain the deleted suffix")
}

func TestProcessPathHelperProcess(t *testing.T) {
	if os.Getenv(processPathHelperEnvironment) != "1" {
		t.Skip("helper process for TestReadProcessPathAfterUnlink")
	}
	time.Sleep(time.Minute)
}
