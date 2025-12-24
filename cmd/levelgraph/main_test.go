// Copyright (c) 2024 LevelGraph Go Contributors
//
// Permission is hereby granted, free of charge, to any person
// obtaining a copy of this software and associated documentation
// files (the "Software"), to deal in the Software without
// restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following
// conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI_Help(t *testing.T) {
	var out, errOut bytes.Buffer
	cli := &CLI{Out: &out, Err: &errOut}

	exitCode := cli.Run([]string{"help"})
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	if !strings.Contains(out.String(), "LevelGraph CLI") {
		t.Error("expected help output to contain 'LevelGraph CLI'")
	}
}

func TestCLI_NoArgs(t *testing.T) {
	var out, errOut bytes.Buffer
	cli := &CLI{Out: &out, Err: &errOut}

	exitCode := cli.Run([]string{})
	if exitCode != 1 {
		t.Errorf("expected exit code 1 for no args, got %d", exitCode)
	}
}

func TestCLI_UnknownCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	cli := &CLI{Out: &out, Err: &errOut}

	exitCode := cli.Run([]string{"unknown"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1 for unknown command, got %d", exitCode)
	}

	if !strings.Contains(errOut.String(), "Unknown command: unknown") {
		t.Errorf("expected error message about unknown command, got: %s", errOut.String())
	}
}

func TestCLI_PutGetDump(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "levelgraph-cli-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Test put
	t.Run("put", func(t *testing.T) {
		var out, errOut bytes.Buffer
		cli := &CLI{Out: &out, Err: &errOut}

		exitCode := cli.Run([]string{"put", "-db", dbPath, "alice", "knows", "bob"})
		if exitCode != 0 {
			t.Errorf("put failed with exit code %d, stderr: %s", exitCode, errOut.String())
		}

		if !strings.Contains(out.String(), "Triple added") {
			t.Errorf("expected 'Triple added' in output, got: %s", out.String())
		}
	})

	// Test get with exact match
	t.Run("get exact", func(t *testing.T) {
		var out, errOut bytes.Buffer
		cli := &CLI{Out: &out, Err: &errOut}

		exitCode := cli.Run([]string{"get", "-db", dbPath, "alice", "knows", "bob"})
		if exitCode != 0 {
			t.Errorf("get failed with exit code %d, stderr: %s", exitCode, errOut.String())
		}

		if !strings.Contains(out.String(), "alice knows bob") {
			t.Errorf("expected 'alice knows bob' in output, got: %s", out.String())
		}
	})

	// Test get with wildcard
	t.Run("get wildcard", func(t *testing.T) {
		var out, errOut bytes.Buffer
		cli := &CLI{Out: &out, Err: &errOut}

		exitCode := cli.Run([]string{"get", "-db", dbPath, "alice", "*", "*"})
		if exitCode != 0 {
			t.Errorf("get wildcard failed with exit code %d, stderr: %s", exitCode, errOut.String())
		}

		if !strings.Contains(out.String(), "alice knows bob") {
			t.Errorf("expected 'alice knows bob' in output, got: %s", out.String())
		}
	})

	// Add another triple
	t.Run("put second", func(t *testing.T) {
		var out, errOut bytes.Buffer
		cli := &CLI{Out: &out, Err: &errOut}

		exitCode := cli.Run([]string{"put", "-db", dbPath, "bob", "knows", "charlie"})
		if exitCode != 0 {
			t.Errorf("second put failed with exit code %d, stderr: %s", exitCode, errOut.String())
		}
	})

	// Test dump
	t.Run("dump", func(t *testing.T) {
		var out, errOut bytes.Buffer
		cli := &CLI{Out: &out, Err: &errOut}

		exitCode := cli.Run([]string{"dump", "-db", dbPath})
		if exitCode != 0 {
			t.Errorf("dump failed with exit code %d, stderr: %s", exitCode, errOut.String())
		}

		output := out.String()
		if !strings.Contains(output, "alice knows bob") {
			t.Errorf("dump missing 'alice knows bob', got: %s", output)
		}
		if !strings.Contains(output, "bob knows charlie") {
			t.Errorf("dump missing 'bob knows charlie', got: %s", output)
		}
	})
}

func TestCLI_Load(t *testing.T) {
	// Create temp directory for test database and input file
	tmpDir, err := os.MkdirTemp("", "levelgraph-cli-load-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	inputFile := filepath.Join(tmpDir, "triples.nt")

	// Create input file with N-Triples format
	inputContent := `# Comment line
alice knows bob .
bob knows charlie .
charlie likes programming .

# Another comment
dave follows alice .
`
	if err := os.WriteFile(inputFile, []byte(inputContent), 0644); err != nil {
		t.Fatalf("failed to write input file: %v", err)
	}

	// Test load
	t.Run("load", func(t *testing.T) {
		var out, errOut bytes.Buffer
		cli := &CLI{Out: &out, Err: &errOut}

		exitCode := cli.Run([]string{"load", "-db", dbPath, inputFile})
		if exitCode != 0 {
			t.Errorf("load failed with exit code %d, stderr: %s", exitCode, errOut.String())
		}

		if !strings.Contains(out.String(), "Loaded 4 triples") {
			t.Errorf("expected 'Loaded 4 triples' in output, got: %s", out.String())
		}
	})

	// Verify loaded data with dump
	t.Run("verify", func(t *testing.T) {
		var out, errOut bytes.Buffer
		cli := &CLI{Out: &out, Err: &errOut}

		exitCode := cli.Run([]string{"dump", "-db", dbPath})
		if exitCode != 0 {
			t.Errorf("dump failed with exit code %d, stderr: %s", exitCode, errOut.String())
		}

		output := out.String()
		expectedTriples := []string{
			"alice knows bob",
			"bob knows charlie",
			"charlie likes programming",
			"dave follows alice",
		}
		for _, expected := range expectedTriples {
			if !strings.Contains(output, expected) {
				t.Errorf("dump missing '%s', got: %s", expected, output)
			}
		}
	})
}

func TestCLI_PutMissingArgs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "levelgraph-cli-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	var out, errOut bytes.Buffer
	cli := &CLI{Out: &out, Err: &errOut}

	exitCode := cli.Run([]string{"put", "-db", dbPath, "alice", "knows"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1 for missing args, got %d", exitCode)
	}

	if !strings.Contains(errOut.String(), "usage") {
		t.Errorf("expected usage message in error, got: %s", errOut.String())
	}
}

func TestCLI_GetMissingArgs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "levelgraph-cli-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	var out, errOut bytes.Buffer
	cli := &CLI{Out: &out, Err: &errOut}

	exitCode := cli.Run([]string{"get", "-db", dbPath, "alice"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1 for missing args, got %d", exitCode)
	}
}

func TestCLI_LoadMissingFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "levelgraph-cli-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	var out, errOut bytes.Buffer
	cli := &CLI{Out: &out, Err: &errOut}

	exitCode := cli.Run([]string{"load", "-db", dbPath, "/nonexistent/file.nt"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1 for missing file, got %d", exitCode)
	}

	if !strings.Contains(errOut.String(), "failed to open file") {
		t.Errorf("expected 'failed to open file' in error, got: %s", errOut.String())
	}
}

func TestCLI_LoadMissingArgs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "levelgraph-cli-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	var out, errOut bytes.Buffer
	cli := &CLI{Out: &out, Err: &errOut}

	exitCode := cli.Run([]string{"load", "-db", dbPath})
	if exitCode != 1 {
		t.Errorf("expected exit code 1 for missing file arg, got %d", exitCode)
	}
}

func TestCLI_InvalidDbPath(t *testing.T) {
	var out, errOut bytes.Buffer
	cli := &CLI{Out: &out, Err: &errOut}

	// Try to open a database in a non-existent directory that can't be created
	exitCode := cli.Run([]string{"dump", "-db", "/nonexistent/deeply/nested/path/db"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1 for invalid db path, got %d", exitCode)
	}

	if !strings.Contains(errOut.String(), "failed to open database") {
		t.Errorf("expected 'failed to open database' in error, got: %s", errOut.String())
	}
}

func TestCLI_InvalidFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	cli := &CLI{Out: &out, Err: &errOut}

	exitCode := cli.Run([]string{"dump", "-invalid-flag"})
	if exitCode != 1 {
		t.Errorf("expected exit code 1 for invalid flag, got %d", exitCode)
	}
}

func TestCLI_HelpVariants(t *testing.T) {
	for _, helpCmd := range []string{"-h", "--help"} {
		t.Run(helpCmd, func(t *testing.T) {
			var out, errOut bytes.Buffer
			cli := &CLI{Out: &out, Err: &errOut}

			exitCode := cli.Run([]string{helpCmd})
			if exitCode != 0 {
				t.Errorf("expected exit code 0 for %s, got %d", helpCmd, exitCode)
			}

			if !strings.Contains(out.String(), "LevelGraph CLI") {
				t.Errorf("expected help output for %s", helpCmd)
			}
		})
	}
}

func TestCLI_LoadEmptyLines(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "levelgraph-cli-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	inputFile := filepath.Join(tmpDir, "sparse.nt")

	// File with lots of empty lines and comments, few actual triples
	inputContent := `

# Start of file

alice knows bob .

   # Indented comment
   
bob knows charlie .

# End
`
	if err := os.WriteFile(inputFile, []byte(inputContent), 0644); err != nil {
		t.Fatalf("failed to write input file: %v", err)
	}

	var out, errOut bytes.Buffer
	cli := &CLI{Out: &out, Err: &errOut}

	exitCode := cli.Run([]string{"load", "-db", dbPath, inputFile})
	if exitCode != 0 {
		t.Errorf("load failed with exit code %d, stderr: %s", exitCode, errOut.String())
	}

	if !strings.Contains(out.String(), "Loaded 2 triples") {
		t.Errorf("expected 'Loaded 2 triples', got: %s", out.String())
	}
}

func TestCLI_LoadIncompleteLines(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "levelgraph-cli-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	inputFile := filepath.Join(tmpDir, "incomplete.nt")

	// Lines with fewer than 3 fields should be skipped
	inputContent := `alice knows bob .
incomplete
ab
valid subject predicate object .
`
	if err := os.WriteFile(inputFile, []byte(inputContent), 0644); err != nil {
		t.Fatalf("failed to write input file: %v", err)
	}

	var out, errOut bytes.Buffer
	cli := &CLI{Out: &out, Err: &errOut}

	exitCode := cli.Run([]string{"load", "-db", dbPath, inputFile})
	if exitCode != 0 {
		t.Errorf("load failed with exit code %d, stderr: %s", exitCode, errOut.String())
	}

	// Should only load 2 valid triples (lines with <3 fields skipped)
	if !strings.Contains(out.String(), "Loaded 2 triples") {
		t.Errorf("expected 'Loaded 2 triples', got: %s", out.String())
	}
}
