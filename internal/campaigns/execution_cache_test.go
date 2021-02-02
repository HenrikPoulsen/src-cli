	"context"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/src-cli/internal/campaigns/graphql"
	"gopkg.in/yaml.v3"

const testDiff = `diff --git a/README.md b/README.md
new file mode 100644
index 0000000..3363c39
--- /dev/null
+++ b/README.md
@@ -0,0 +1,3 @@
+# README
+
+This is the readme
`

func TestExecutionDiskCache(t *testing.T) {
	ctx := context.Background()

	cacheTmpDir := func(t *testing.T) string {
		testTempDir, err := ioutil.TempDir("", "execution-disk-cache-test-*")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Remove(testTempDir) })

		return testTempDir
	}

	cacheKey1 := ExecutionCacheKey{Task: &Task{
		Repository: &graphql.Repository{Name: "src-cli"},
		Steps: []Step{
			{Run: "echo 'Hello World'", Container: "alpine:3"},
		},
	}}

	cacheKey2 := ExecutionCacheKey{Task: &Task{
		Repository: &graphql.Repository{Name: "documentation"},
		Steps: []Step{
			{Run: "echo 'Hello World'", Container: "alpine:3"},
		},
	}}

	value := executionResult{
		Diff: testDiff,
		ChangedFiles: &StepChanges{
			Added: []string{"README.md"},
		},
		Outputs: map[string]interface{}{},
	}

	onlyDiff := executionResult{
		Diff:         testDiff,
		ChangedFiles: &StepChanges{},
		Outputs:      map[string]interface{}{},
	}

	t.Run("cache contains v3 cache file", func(t *testing.T) {
		cache := ExecutionDiskCache{Dir: cacheTmpDir(t)}

		// Empty cache, no hits
		assertCacheMiss(t, cache, cacheKey1)
		assertCacheMiss(t, cache, cacheKey2)

		// Set the cache
		if err := cache.Set(ctx, cacheKey1, value); err != nil {
			t.Fatalf("cache.Set returned unexpected error: %s", err)
		}

		// Cache hit
		assertCacheHit(t, cache, cacheKey1, value)

		// Cache miss due to different key
		assertCacheMiss(t, cache, cacheKey2)

		// Cache miss due to cleared cache
		if err := cache.Clear(ctx, cacheKey1); err != nil {
			t.Fatalf("cache.Get returned unexpected error: %s", err)
		}
		assertCacheMiss(t, cache, cacheKey1)
	})

	t.Run("cache contains v1 cache file", func(t *testing.T) {
		cache := ExecutionDiskCache{Dir: cacheTmpDir(t)}

		// Empty cache, no hit
		assertCacheMiss(t, cache, cacheKey1)

		// Simulate old cache file lying around in cache
		oldFilePath := writeV1CacheFile(t, cache, cacheKey1, testDiff)

		// Cache hit, but only for the diff
		assertCacheHit(t, cache, cacheKey1, onlyDiff)

		// And the old file should be deleted
		assertFileDeleted(t, oldFilePath)
		// .. but we should still get a cache hit, because we rewrote the cache
		assertCacheHit(t, cache, cacheKey1, onlyDiff)
	})

	t.Run("cache contains v2 cache file", func(t *testing.T) {
		cache := ExecutionDiskCache{Dir: cacheTmpDir(t)}

		// Empty cache, no hit
		assertCacheMiss(t, cache, cacheKey1)

		// Simulate old cache file lying around in cache
		oldFilePath := writeV2CacheFile(t, cache, cacheKey1, testDiff)

		// Now we get a cache hit, but only for the diff
		assertCacheHit(t, cache, cacheKey1, onlyDiff)

		// And the old file should be deleted
		assertFileDeleted(t, oldFilePath)
		// .. but we should still get a cache hit, because we rewrote the cache
		assertCacheHit(t, cache, cacheKey1, onlyDiff)
	})

	t.Run("cache contains one old and one v3 cache file", func(t *testing.T) {
		cache := ExecutionDiskCache{Dir: cacheTmpDir(t)}

		// Simulate v2 and v3 files in cache
		oldFilePath := writeV1CacheFile(t, cache, cacheKey1, testDiff)

		if err := cache.Set(ctx, cacheKey1, value); err != nil {
			t.Fatalf("cache.Set returned unexpected error: %s", err)
		}

		// Cache hit
		assertCacheHit(t, cache, cacheKey1, value)

		// And the old file should be deleted
		assertFileDeleted(t, oldFilePath)
	})

	t.Run("cache contains multiple old cache files", func(t *testing.T) {
		cache := ExecutionDiskCache{Dir: cacheTmpDir(t)}

		// Simulate v1 and v2 files in cache
		oldFilePath1 := writeV1CacheFile(t, cache, cacheKey1, testDiff)
		oldFilePath2 := writeV1CacheFile(t, cache, cacheKey1, testDiff)

		// Now we get a cache hit, but only for the diff
		assertCacheHit(t, cache, cacheKey1, onlyDiff)

		// And the old files should be deleted
		assertFileDeleted(t, oldFilePath1)
		assertFileDeleted(t, oldFilePath2)
		// .. but we should still get a cache hit, because we rewrote the cache
		assertCacheHit(t, cache, cacheKey1, onlyDiff)
	})
}

func TestSortCacheFiles(t *testing.T) {
	tests := []struct {
		paths []string
		want  []string
	}{
		{
			paths: []string{"file.v3.json", "file.diff", "file.json"},
			want:  []string{"file.v3.json", "file.diff", "file.json"},
		},
		{
			paths: []string{"file.json", "file.diff", "file.v3.json"},
			want:  []string{"file.v3.json", "file.json", "file.diff"},
		},
		{
			paths: []string{"file.diff", "file.v3.json"},
			want:  []string{"file.v3.json", "file.diff"},
		},
		{
			paths: []string{"file1.v3.json", "file2.v3.json"},
			want:  []string{"file1.v3.json", "file2.v3.json"},
		},
	}

	for _, tt := range tests {
		sortCacheFiles(tt.paths)
		if diff := cmp.Diff(tt.paths, tt.want); diff != "" {
			t.Errorf("wrong cached result (-have +want):\n\n%s", diff)
		}
	}
}

func assertFileDeleted(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("file exists: %s", path)
	} else if os.IsNotExist(err) {
		// Seems to be deleted, all good
	} else {
		t.Fatalf("could not determine whether file exists: %s", err)
	}
}

func writeV1CacheFile(t *testing.T, c ExecutionDiskCache, k ExecutionCacheKey, diff string) (path string) {
	t.Helper()

	hashedKey, err := k.Key()
	if err != nil {
		t.Fatalf("failed to hash cacheKey: %s", err)
	}
	// The v1 file format ended in .json
	path = filepath.Join(c.Dir, hashedKey+".json")

	// v1 contained a fully serialized ChangesetSpec
	spec := ChangesetSpec{CreatedChangeset: &CreatedChangeset{
		Commits: []GitCommitDescription{
			{Diff: testDiff},
		},
	}}

	raw, err := json.Marshal(&spec)
	if err != nil {
		t.Fatal(err)
	}

	if err := ioutil.WriteFile(path, raw, 0600); err != nil {
		t.Fatalf("writing the cache file failed: %s", err)
	}

	return path
}

func writeV2CacheFile(t *testing.T, c ExecutionDiskCache, k ExecutionCacheKey, diff string) (path string) {
	t.Helper()

	hashedKey, err := k.Key()
	if err != nil {
		t.Fatalf("failed to hash cacheKey: %s", err)
	}

	// The v2 file format ended in .json
	path = filepath.Join(c.Dir, hashedKey+".diff")

	// v2 contained only a diff
	if err := ioutil.WriteFile(path, []byte(diff), 0600); err != nil {
		t.Fatalf("writing the cache file failed: %s", err)
	}

	return path
}

func assertCacheHit(t *testing.T, c ExecutionDiskCache, k ExecutionCacheKey, want executionResult) {
	t.Helper()

	have, found, err := c.Get(context.Background(), k)
	if err != nil {
		t.Fatalf("cache.Get returned unexpected error: %s", err)
	}
	if !found {
		t.Fatalf("cache miss when hit was expected")
	}

	if diff := cmp.Diff(have, want); diff != "" {
		t.Errorf("wrong cached result (-have +want):\n\n%s", diff)
	}
}

func assertCacheMiss(t *testing.T, c ExecutionDiskCache, k ExecutionCacheKey) {
	t.Helper()

	_, found, err := c.Get(context.Background(), k)
	if err != nil {
		t.Fatalf("cache.Get returned unexpected error: %s", err)
	}
	if found {
		t.Fatalf("cache hit when miss was expected")
	}
}