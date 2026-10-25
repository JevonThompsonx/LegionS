package unpack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRemoveAllEscapesRoot documents that RemoveAll on a staging path can
// escape its intended root if the path is constructed from unsanitized input.
// SAFE BEHAVIOR: paths must be validated to stay within an allowed root before deletion.
func TestRemoveAllEscapesRoot(t *testing.T) {
	// Create a temp root with a safe directory inside
	root := t.TempDir()
	safe := filepath.Join(root, "safe")
	if err := os.MkdirAll(safe, 0o755); err != nil {
		t.Fatal(err)
	}

	// Simulate a malicious GameDir that resolves outside root
	malicious := root + "/../evil"
	// Clean the path (this is what SHOULD happen but doesn't)
	cleaned := filepath.Clean(malicious)

	// If RemoveAll is called on cleaned path, it deletes outside root
	// This test documents the vulnerability: no lexical root check exists
	if !strings.HasPrefix(cleaned, root) {
		t.Logf("VULNERABILITY: path %q escapes root %q", cleaned, root)
		// In the current code, stage = GameDir + ".unpack-staging" with no validation
	}
}

// TestZipSlipExtraction documents that tar extraction does not validate
// extracted paths stay within the destination directory.
// SAFE BEHAVIOR: extraction must reject entries with ".." or absolute paths.
func TestZipSlipExtraction(t *testing.T) {
	// compress.Extract runs: tar -x --zstd -f archive -C dest
	// There is NO check for path traversal in archive entries.
	// A malicious tar containing "../../etc/cron.d/evil" would write outside dest.

	// Verify by checking the Extract function signature has no safety params
	// (This is a documentation test — actual tar content testing is integration-only)
	t.Log("VULNERABILITY: compress.Extract has no Zip-Slip protection")
	t.Log("tar is invoked with -C dest but no --restrict or path validation")
}

// TestPrefixRestoreNonAtomic documents that prefix restore extracts
// directly into the live CompatData directory without staging.
// SAFE BEHAVIOR: extract to staging dir, then atomically swap.
func TestPrefixRestoreNonAtomic(t *testing.T) {
	// restorePrefix calls compress.Extract directly to c.CompatData
	// If extraction fails midway, the prefix is left in a broken state.
	// There is no rollback mechanism.
	t.Log("VULNERABILITY: prefix restore is non-atomic")
	t.Log("restorePrefix extracts directly to CompatData with no staging or rollback")
}
