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
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/benbenbenbenbenben/levelgraph"
)

func main() {
	cli := &CLI{
		Out: os.Stdout,
		Err: os.Stderr,
	}
	os.Exit(cli.Run(os.Args[1:]))
}

// CLI encapsulates the command-line interface for LevelGraph.
type CLI struct {
	Out io.Writer // Output writer (default: os.Stdout)
	Err io.Writer // Error writer (default: os.Stderr)
}

// Run executes the CLI with the given arguments and returns an exit code.
func (c *CLI) Run(args []string) int {
	if len(args) < 1 {
		c.printUsage()
		return 1
	}

	cmd := args[0]
	cmdArgs := args[1:]

	var err error
	switch cmd {
	case "put":
		err = c.runPut(cmdArgs)
	case "get":
		err = c.runGet(cmdArgs)
	case "dump":
		err = c.runDump(cmdArgs)
	case "load":
		err = c.runLoad(cmdArgs)
	case "help", "-h", "--help":
		c.printUsage()
		return 0
	default:
		fmt.Fprintf(c.Err, "Unknown command: %s\n", cmd)
		c.printUsage()
		return 1
	}

	if err != nil {
		fmt.Fprintf(c.Err, "Error: %v\n", err)
		return 1
	}
	return 0
}

func (c *CLI) printUsage() {
	fmt.Fprint(c.Out, `LevelGraph CLI

Usage:
  levelgraph <command> [arguments]

Commands:
  put <subject> <predicate> <object>   Add a triple
  get <subject> <predicate> <object>   Get triples (use '*' as wildcard)
  dump                                 Dump all triples
  load <file>                          Load triples from a file (N-Triples format)
  help                                 Show this help message

Global Flags:
  -db <path>                           Path to database (default: levelgraph.db)
`)
}

// parseFlags parses command-line flags and opens the database.
func (c *CLI) parseFlags(args []string) (*levelgraph.DB, []string, error) {
	fs := flag.NewFlagSet("levelgraph", flag.ContinueOnError)
	fs.SetOutput(c.Err)
	dbPath := fs.String("db", "levelgraph.db", "Path to database")

	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	db, err := levelgraph.Open(*dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}
	return db, fs.Args(), nil
}

func (c *CLI) runPut(args []string) error {
	db, remaining, err := c.parseFlags(args)
	if err != nil {
		return err
	}
	defer db.Close()

	if len(remaining) != 3 {
		return fmt.Errorf("usage: levelgraph put <subject> <predicate> <object>")
	}

	err = db.Put(context.Background(), levelgraph.NewTripleFromStrings(remaining[0], remaining[1], remaining[2]))
	if err != nil {
		return fmt.Errorf("failed to put triple: %w", err)
	}
	fmt.Fprintln(c.Out, "Triple added.")
	return nil
}

func (c *CLI) runGet(args []string) error {
	db, remaining, err := c.parseFlags(args)
	if err != nil {
		return err
	}
	defer db.Close()

	if len(remaining) != 3 {
		return fmt.Errorf("usage: levelgraph get <subject> <predicate> <object> (use '*' for wildcard)")
	}

	parsePart := func(s string) []byte {
		if s == "*" {
			return nil
		}
		return []byte(s)
	}

	pattern := levelgraph.NewPattern(parsePart(remaining[0]), parsePart(remaining[1]), parsePart(remaining[2]))

	triples, err := db.Get(context.Background(), pattern)
	if err != nil {
		return fmt.Errorf("failed to get triples: %w", err)
	}

	for _, t := range triples {
		fmt.Fprintf(c.Out, "%s %s %s\n", t.Subject, t.Predicate, t.Object)
	}
	return nil
}

func (c *CLI) runDump(args []string) error {
	db, _, err := c.parseFlags(args)
	if err != nil {
		return err
	}
	defer db.Close()

	triples, err := db.Get(context.Background(), &levelgraph.Pattern{})
	if err != nil {
		return fmt.Errorf("failed to dump triples: %w", err)
	}

	for _, t := range triples {
		fmt.Fprintf(c.Out, "%s %s %s\n", t.Subject, t.Predicate, t.Object)
	}
	return nil
}

func (c *CLI) runLoad(args []string) error {
	db, remaining, err := c.parseFlags(args)
	if err != nil {
		return err
	}
	defer db.Close()

	if len(remaining) != 1 {
		return fmt.Errorf("usage: levelgraph load <file>")
	}

	filePath := remaining[0]
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	count, err := c.loadTriples(db, file)
	if err != nil {
		return err
	}

	fmt.Fprintf(c.Out, "Loaded %d triples.\n", count)
	return nil
}

// loadTriples loads triples from an N-Triples format reader into the database.
func (c *CLI) loadTriples(db *levelgraph.DB, r io.Reader) (int, error) {
	scanner := bufio.NewScanner(r)
	count := 0
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 3 {
			// Basic N-Triples parsing (simplified)
			sub := parts[0]
			pred := parts[1]
			obj := strings.Join(parts[2:], " ")
			obj = strings.TrimSuffix(obj, " .")

			err := db.Put(context.Background(), levelgraph.NewTripleFromStrings(sub, pred, obj))
			if err != nil {
				fmt.Fprintf(c.Err, "Warning: line %d: failed to put triple: %v\n", lineNum, err)
			} else {
				count++
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return count, fmt.Errorf("error reading input: %w", err)
	}

	return count, nil
}
