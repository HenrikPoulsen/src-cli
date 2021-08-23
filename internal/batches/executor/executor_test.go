	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/sourcegraph/src-cli/internal/batches/docker"
	"github.com/sourcegraph/src-cli/internal/batches/git"

		wantFinished        int
		wantFinishedWithErr int
				{Repo: testRepo1, Files: map[string]string{
				{Repo: testRepo2, Files: map[string]string{
				{Repository: testRepo1},
				{Repository: testRepo2},
				testRepo1.ID: filesByPath{
				testRepo2.ID: {
			wantFinished: 2,
				{Repo: testRepo1, Files: map[string]string{
				{Repository: testRepo1},
				testRepo1.ID: filesByPath{
			wantFinished: 1,
				{Repo: testRepo1, Files: map[string]string{"README.md": "line 1"}},
				{Repository: testRepo1},
			executorTimeout:     100 * time.Millisecond,
			wantErrInclude:      "execution in github.com/sourcegraph/src-cli failed: Timeout reached. Execution took longer than 100ms.",
			wantFinishedWithErr: 1,
				{Repo: testRepo1, Files: map[string]string{
				{Repository: testRepo1},
				testRepo1.ID: filesByPath{
			wantFinished: 1,
				{Repo: testRepo1, Path: "", Files: map[string]string{
				{Repo: testRepo1, Path: "a", Files: map[string]string{
				{Repo: testRepo1, Path: "a/b", Files: map[string]string{
				{Repo: testRepo1, AdditionalFiles: map[string]string{
				{Repository: testRepo1, Path: ""},
				{Repository: testRepo1, Path: "a"},
				{Repository: testRepo1, Path: "a/b"},
				testRepo1.ID: filesByPath{
			wantFinished: 3,
				{Repo: testRepo1, Files: map[string]string{
				{Repo: testRepo2, Files: map[string]string{
				{Repository: testRepo1},
				{Repository: testRepo2, Path: "sub/directory/of/repo"},
				testRepo1.ID: filesByPath{
				testRepo2.ID: {
			wantFinished: 2,
				{Repo: testRepo1, Files: map[string]string{
				{Repo: testRepo2, Files: map[string]string{
					If:  fmt.Sprintf(`${{ eq repository.name %q }}`, testRepo2.Name),
				{Repository: testRepo1},
				{Repository: testRepo2},
				testRepo1.ID: filesByPath{
				testRepo2.ID: {},
			wantErrInclude:      "execution in github.com/sourcegraph/sourcegraph failed: run: exit 1",
			wantFinished:        1,
			wantFinishedWithErr: 1,
			images := make(map[string]docker.Image)
			for _, step := range tc.steps {
				images[step.Container] = &mock.Image{RawDigest: step.Container}
				Creator:     workspace.NewCreator(context.Background(), "bind", testTempDir, testTempDir, images),
				Fetcher:     batches.NewRepoFetcher(client, testTempDir, false),
				Logger:      mock.LogNoOpManager{},
				EnsureImage: imageMapEnsurer(images),
			dummyUI := newDummyTaskExecutionUI()
			executor.Start(context.Background(), tc.tasks, dummyUI)
					if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.wantErrInclude)) {
						t.Errorf("wrong error. have=%q want included=%q", err, tc.wantErrInclude)

			// Make sure that all the Tasks have been updated correctly
			if have, want := len(dummyUI.finished), tc.wantFinished; have != want {
				t.Fatalf("wrong number of finished tasks. want=%d, have=%d", want, have)
			}
			if have, want := len(dummyUI.finishedWithErr), tc.wantFinishedWithErr; have != want {
				t.Fatalf("wrong number of finished-with-err tasks. want=%d, have=%d", want, have)
			}
		AllowOptionalPublished:   true,
	}
}

func TestExecutor_CachedStepResults(t *testing.T) {
	t.Run("single step cached", func(t *testing.T) {
		archive := mock.RepoArchive{
			Repo: testRepo1, Files: map[string]string{
				"README.md": "# Welcome to the README\n",
			},
		}

		cachedDiff := []byte(`diff --git README.md README.md
index 02a19af..c9644dd 100644
--- README.md
+++ README.md
@@ -1 +1,2 @@
 # Welcome to the README
+foobar
`)

		task := &Task{
			BatchChangeAttributes: &BatchChangeAttributes{},
			Steps: []batches.Step{
				{Run: `echo -e "foobar\n" >> README.md`},
			},
			CachedResultFound: true,
			CachedResult: stepExecutionResult{
				StepIndex:          0,
				Diff:               cachedDiff,
				Outputs:            map[string]interface{}{},
				PreviousStepResult: StepResult{},
			},
			Repository: testRepo1,
		}

		results, err := testExecuteTasks(t, []*Task{task}, archive)
		if err != nil {
			t.Fatalf("execution failed: %s", err)
		}

		if have, want := len(results), 1; have != want {
			t.Fatalf("wrong number of execution results. want=%d, have=%d", want, have)
		}

		// We want the diff to be the same as the cached one, since we only had to
		// execute a single step
		executionResult := results[0].result
		if diff := cmp.Diff(executionResult.Diff, string(cachedDiff)); diff != "" {
			t.Fatalf("wrong diff: %s", diff)
		}

		if have, want := len(results[0].stepResults), 1; have != want {
			t.Fatalf("wrong length of step results. have=%d, want=%d", have, want)
		}

		stepResult := results[0].stepResults[0]
		if diff := cmp.Diff(stepResult, task.CachedResult); diff != "" {
			t.Fatalf("wrong stepResult: %s", diff)
		}
	})

	t.Run("one of multiple steps cached", func(t *testing.T) {
		archive := mock.RepoArchive{
			Repo: testRepo1,
			Files: map[string]string{
				"README.md": `# automation-testing
This repository is used to test opening and closing pull request with Automation

(c) Copyright Sourcegraph 2013-2020.
(c) Copyright Sourcegraph 2013-2020.
(c) Copyright Sourcegraph 2013-2020.`,
			},
		}

		cachedDiff := []byte(`diff --git README.md README.md
index 1914491..cd2ccbf 100644
--- README.md
+++ README.md
@@ -3,4 +3,5 @@ This repository is used to test opening and closing pull request with Automation

 (c) Copyright Sourcegraph 2013-2020.
 (c) Copyright Sourcegraph 2013-2020.
-(c) Copyright Sourcegraph 2013-2020.
\ No newline at end of file
+(c) Copyright Sourcegraph 2013-2020.this is step 2
+this is step 3
diff --git README.txt README.txt
new file mode 100644
index 0000000..888e1ec
--- /dev/null
+++ README.txt
@@ -0,0 +1 @@
+this is step 1
`)

		wantFinalDiff := `diff --git README.md README.md
index 1914491..d6782d3 100644
--- README.md
+++ README.md
@@ -3,4 +3,7 @@ This repository is used to test opening and closing pull request with Automation
 
 (c) Copyright Sourcegraph 2013-2020.
 (c) Copyright Sourcegraph 2013-2020.
-(c) Copyright Sourcegraph 2013-2020.
\ No newline at end of file
+(c) Copyright Sourcegraph 2013-2020.this is step 2
+this is step 3
+this is step 4
+previous_step.modified_files=[README.md]
diff --git README.txt README.txt
new file mode 100644
index 0000000..888e1ec
--- /dev/null
+++ README.txt
@@ -0,0 +1 @@
+this is step 1
diff --git my-output.txt my-output.txt
new file mode 100644
index 0000000..257ae8e
--- /dev/null
+++ my-output.txt
@@ -0,0 +1 @@
+this is step 5
`

		task := &Task{
			Repository:            testRepo1,
			BatchChangeAttributes: &BatchChangeAttributes{},
			Steps: []batches.Step{
				{Run: `echo "this is step 1" >> README.txt`},
				{Run: `echo "this is step 2" >> README.md`},
				{Run: `echo "this is step 3" >> README.md`, Outputs: batches.Outputs{
					"myOutput": batches.Output{
						Value: "my-output.txt",
					},
				}},
				{Run: `echo "this is step 4" >> README.md
echo "previous_step.modified_files=${{ previous_step.modified_files }}" >> README.md
`},
				{Run: `echo "this is step 5" >> ${{ outputs.myOutput }}`},
			},
			CachedResultFound: true,
			CachedResult: stepExecutionResult{
				StepIndex: 2,
				Diff:      cachedDiff,
				Outputs: map[string]interface{}{
					"myOutput": "my-output.txt",
				},
				PreviousStepResult: StepResult{
					Files: &git.Changes{
						Modified: []string{"README.md"},
						Added:    []string{"README.txt"},
					},
					Stdout: nil,
					Stderr: nil,
				},
			},
		}

		results, err := testExecuteTasks(t, []*Task{task}, archive)
		if err != nil {
			t.Fatalf("execution failed: %s", err)
		}

		if have, want := len(results), 1; have != want {
			t.Fatalf("wrong number of execution results. want=%d, have=%d", want, have)
		}

		executionResult := results[0].result
		if diff := cmp.Diff(executionResult.Diff, wantFinalDiff); diff != "" {
			t.Fatalf("wrong diff: %s", diff)
		}

		if diff := cmp.Diff(executionResult.Outputs, task.CachedResult.Outputs); diff != "" {
			t.Fatalf("wrong execution result outputs: %s", diff)
		}

		// Only two steps should've been executed
		if have, want := len(results[0].stepResults), 2; have != want {
			t.Fatalf("wrong length of step results. have=%d, want=%d", have, want)
		}

		lastStepResult := results[0].stepResults[1]
		if have, want := lastStepResult.StepIndex, 4; have != want {
			t.Fatalf("wrong stepIndex. have=%d, want=%d", have, want)
		}

		if diff := cmp.Diff(lastStepResult.Outputs, task.CachedResult.Outputs); diff != "" {
			t.Fatalf("wrong step result outputs: %s", diff)
		}
	})
}

func testExecuteTasks(t *testing.T, tasks []*Task, archives ...mock.RepoArchive) ([]taskResult, error) {
	if runtime.GOOS == "windows" {
		t.Skip("Test doesn't work on Windows because dummydocker is written in bash")
	}

	testTempDir, err := ioutil.TempDir("", "executor-integration-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(testTempDir) })

	// Setup dummydocker
	addToPath(t, "testdata/dummydocker")

	// Setup mock test server & client
	mux := mock.NewZipArchivesMux(t, nil, archives...)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	var clientBuffer bytes.Buffer
	client := api.NewClient(api.ClientOpts{Endpoint: ts.URL, Out: &clientBuffer})

	// Prepare images
	//
	images := make(map[string]docker.Image)
	for _, t := range tasks {
		for _, step := range t.Steps {
			images[step.Container] = &mock.Image{RawDigest: step.Container}
		}
	}

	// Setup executor
	executor := newExecutor(newExecutorOpts{
		Creator:     workspace.NewCreator(context.Background(), "bind", testTempDir, testTempDir, images),
		Fetcher:     batches.NewRepoFetcher(client, testTempDir, false),
		Logger:      mock.LogNoOpManager{},
		EnsureImage: imageMapEnsurer(images),

		TempDir:     testTempDir,
		Parallelism: runtime.GOMAXPROCS(0),
		Timeout:     30 * time.Second,
	})

	executor.Start(context.Background(), tasks, newDummyTaskExecutionUI())
	return executor.Wait(context.Background())
}

func imageMapEnsurer(m map[string]docker.Image) imageEnsurer {
	return func(_ context.Context, container string) (docker.Image, error) {
		if i, ok := m[container]; ok {
			return i, nil
		}
		return nil, errors.New(fmt.Sprintf("image for %s not found", container))