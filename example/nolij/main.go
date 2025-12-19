// Package main implements nolij, a knowledge graph CLI built on LevelGraph.
package main

import (
	"context"
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/levelgraph/levelgraph"
)

const dbPath = ".nolij.db"

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "help", "-h", "--help":
		printHelp()
	case "add":
		cmdAdd(args)
	case "del", "rm":
		cmdDel(args)
	case "find":
		cmdFind(args)
	case "from":
		cmdFrom(args)
	case "path":
		cmdPath(args)
	case "join":
		cmdJoin(args)
	case "sync":
		cmdSync(args)
	case "stats":
		cmdStats()
	case "dump":
		cmdDump()
	case "nuke":
		cmdNuke()
// TODO: case "install": // install to the users bin path (any OS!) - needs docs etc too
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`nolij - A knowledge graph CLI

Usage: nolij <command> [arguments]

Commands:
  add <subject> <predicate> <object>   Add a triple to the graph
  del <subject> <predicate> <object>   Delete a triple from the graph
  find <s> <p> <o>                     Search (use ? or * as wildcard)
  from <node>                          Follow all edges from a node
  path <start> <end>                   Find path between two nodes (BFS)
  join <s1> <p1> <o1> <s2> <p2> <o2>   Join two patterns (use $var for variables)
  sync                                 Index markdown files in current directory
  stats                                Show database statistics
  dump                                 Print all triples
  nuke                                 Delete the database (with confirmation)
  help                                 Show this help message

Examples:
  nolij add alice knows bob
  nolij add bob works_at acme
  nolij find alice ? ?                 # Find all facts about alice
  nolij find ? knows ?                 # Find all "knows" relationships
  nolij from alice                     # Follow edges from alice
  nolij path alice london              # Find path from alice to london
  nolij join file:README.md '$p' '$b' '$b' "codeblock:has meta:raw" bash
  nolij sync                           # Index .md files

The database is stored in .nolij.db/ in the current directory.`)
}

func openDB() (*levelgraph.DB, error) {
	return levelgraph.Open(dbPath)
}

func cmdAdd(args []string) {
	if len(args) != 3 {
		fmt.Println("Usage: nolij add <subject> <predicate> <object>")
		os.Exit(1)
	}

	db, err := openDB()
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	triple := levelgraph.NewTripleFromStrings(args[0], args[1], args[2])
	if err := db.Put(context.Background(), triple); err != nil {
		fmt.Printf("Error adding triple: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Added: %s → %s → %s\n", args[0], args[1], args[2])
}

func cmdDel(args []string) {
	if len(args) != 3 {
		fmt.Println("Usage: nolij del <subject> <predicate> <object>")
		os.Exit(1)
	}

	db, err := openDB()
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	triple := levelgraph.NewTripleFromStrings(args[0], args[1], args[2])
	if err := db.Del(context.Background(), triple); err != nil {
		fmt.Printf("Error deleting triple: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Deleted: %s → %s → %s\n", args[0], args[1], args[2])
}

func cmdFind(args []string) {
	if len(args) != 3 {
		fmt.Println("Usage: nolij find <subject> <predicate> <object>")
		fmt.Println("Use ? or * as wildcard")
		os.Exit(1)
	}

	db, err := openDB()
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	pattern := &levelgraph.Pattern{}
	if args[0] != "?" && args[0] != "*" {
		pattern.Subject = []byte(args[0])
	}
	if args[1] != "?" && args[1] != "*" {
		pattern.Predicate = []byte(args[1])
	}
	if args[2] != "?" && args[2] != "*" {
		pattern.Object = []byte(args[2])
	}

	results, err := db.Get(context.Background(), pattern)
	if err != nil {
		fmt.Printf("Error searching: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println("No results found.")
		return
	}

	for _, t := range results {
		fmt.Printf("%s → %s → %s\n", t.Subject, t.Predicate, t.Object)
	}
	fmt.Printf("\n(%d results)\n", len(results))
}

func cmdFrom(args []string) {
	if len(args) != 1 {
		fmt.Println("Usage: nolij from <node>")
		os.Exit(1)
	}

	db, err := openDB()
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	results, err := db.Get(context.Background(), &levelgraph.Pattern{
		Subject: []byte(args[0]),
	})
	if err != nil {
		fmt.Printf("Error searching: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Printf("No edges from '%s'\n", args[0])
		return
	}

	fmt.Printf("Edges from '%s':\n", args[0])
	for _, t := range results {
		fmt.Printf("  ─%s→ %s\n", t.Predicate, t.Object)
	}
}

func cmdPath(args []string) {
	if len(args) != 2 {
		fmt.Println("Usage: nolij path <start> <end>")
		os.Exit(1)
	}

	db, err := openDB()
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	start, end := args[0], args[1]

	// BFS path finding
	type pathNode struct {
		node string
		pred string
		prev *pathNode
	}

	visited := make(map[string]bool)
	queue := []*pathNode{{node: start}}
	visited[start] = true

	var found *pathNode
	for len(queue) > 0 && found == nil {
		current := queue[0]
		queue = queue[1:]

		results, err := db.Get(context.Background(), &levelgraph.Pattern{
			Subject: []byte(current.node),
		})
		if err != nil {
			continue
		}

		for _, t := range results {
			next := string(t.Object)
			if visited[next] {
				continue
			}
			visited[next] = true

			nextNode := &pathNode{
				node: next,
				pred: string(t.Predicate),
				prev: current,
			}

			if next == end {
				found = nextNode
				break
			}
			queue = append(queue, nextNode)
		}
	}

	if found == nil {
		fmt.Printf("No path found from '%s' to '%s'\n", start, end)
		return
	}

	// Reconstruct path
	var path []string
	for n := found; n != nil; n = n.prev {
		if n.prev != nil {
			path = append([]string{fmt.Sprintf("─%s→ [%s]", n.pred, n.node)}, path...)
		} else {
			path = append([]string{fmt.Sprintf("[%s]", n.node)}, path...)
		}
	}

	fmt.Println(strings.Join(path, " "))
}

func cmdJoin(args []string) {
	if len(args) != 6 {
		fmt.Println("Usage: nolij join <s1> <p1> <o1> <s2> <p2> <o2>")
		fmt.Println("Use $varname for variables to join on")
		fmt.Println("Example: nolij join file:README.md '$p' '$block' '$block' \"codeblock:has meta:raw\" bash")
		os.Exit(1)
	}

	db, err := openDB()
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Parse patterns - $var becomes a Variable, else concrete value or wildcard
	parseValue := func(s string) interface{} {
		if strings.HasPrefix(s, "$") {
			return levelgraph.V(s[1:])
		}
		if s == "?" || s == "*" {
			return levelgraph.V("_wild_" + s)
		}
		return []byte(s)
	}

	pattern1 := &levelgraph.Pattern{
		Subject:   parseValue(args[0]),
		Predicate: parseValue(args[1]),
		Object:    parseValue(args[2]),
	}
	pattern2 := &levelgraph.Pattern{
		Subject:   parseValue(args[3]),
		Predicate: parseValue(args[4]),
		Object:    parseValue(args[5]),
	}

	results, err := db.Search(context.Background(), []*levelgraph.Pattern{pattern1, pattern2}, nil)
	if err != nil {
		fmt.Printf("Error searching: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println("No results found.")
		return
	}

	// Print results
	for _, sol := range results {
		var parts []string
		for k, v := range sol {
			if !strings.HasPrefix(k, "_wild_") {
				parts = append(parts, fmt.Sprintf("%s=%s", k, v))
			}
		}
		fmt.Println(strings.Join(parts, ", "))
	}
	fmt.Printf("\n(%d results)\n", len(results))
}

func cmdSync(args []string) {
	fmt.Println("📁 Syncing markdown files...")

	db, err := openDB()
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Collect markdown files
	var mdFiles []string
	err = filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip hidden directories (except current dir)
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && path != "." {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(path), ".md") {
			mdFiles = append(mdFiles, path)
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
		os.Exit(1)
	}

	if len(mdFiles) == 0 {
		fmt.Println("No markdown files found.")
		return
	}

	fmt.Printf("Found %d markdown file(s)\n\n", len(mdFiles))

	// Track file status
	type fileStatus struct {
		path    string
		status  string // "new", "synced", "desynced"
		hash    string
		oldHash string
	}
	var statuses []fileStatus

	// PASS 1: File discovery and hashing
	fmt.Println("Pass 1: File discovery and hashing...")
	for _, path := range mdFiles {
		fileKey := "file:" + path

		// Compute SHA256
		hash, err := hashFile(path)
		if err != nil {
			fmt.Printf("  ⚠ Error hashing %s: %v\n", path, err)
			continue
		}

		// Check existing hash
		existingResults, err := db.Get(context.Background(), &levelgraph.Pattern{
			Subject:   []byte(fileKey),
			Predicate: []byte("has:sha256"),
		})
		if err != nil {
			fmt.Printf("  ⚠ Error querying %s: %v\n", path, err)
			continue
		}

		status := fileStatus{path: path, hash: hash}
		if len(existingResults) == 0 {
			status.status = "new"
		} else {
			oldHash := string(existingResults[0].Object)
			status.oldHash = oldHash
			if oldHash == hash {
				status.status = "synced"
			} else {
				status.status = "desynced"
			}
		}

		statuses = append(statuses, status)

		// Add/update triples
		db.Put(context.Background(), levelgraph.NewTripleFromStrings("nolij:root", "contains:file", fileKey))

		// Delete old hash if exists
		if status.oldHash != "" {
			db.Del(context.Background(), levelgraph.NewTripleFromStrings(fileKey, "has:sha256", status.oldHash))
		}
		db.Put(context.Background(), levelgraph.NewTripleFromStrings(fileKey, "has:sha256", hash))

		statusIcon := map[string]string{"new": "✚", "synced": "✓", "desynced": "↻"}[status.status]
		fmt.Printf("  %s %s\n", statusIcon, path)
	}

	// Process files that need syncing (new or desynced)
	for _, status := range statuses {
		if status.status == "synced" {
			continue
		}

		fileKey := "file:" + status.path
		content, err := os.ReadFile(status.path)
		if err != nil {
			fmt.Printf("  ⚠ Error reading %s: %v\n", status.path, err)
			continue
		}

		// PASS 2: Link extraction
		syncLinks(db, fileKey, string(content))

		// PASS 3: Code block extraction
		syncCodeBlocks(db, fileKey, string(content))
	}

	// Print summary
	var newCount, syncedCount, desyncedCount int
	for _, s := range statuses {
		switch s.status {
		case "new":
			newCount++
		case "synced":
			syncedCount++
		case "desynced":
			desyncedCount++
		}
	}

	fmt.Printf("\n✅ Sync complete: %d new, %d unchanged, %d updated\n", newCount, syncedCount, desyncedCount)
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func syncLinks(db *levelgraph.DB, fileKey, content string) {
	// Remove old link predicates
	results, _ := db.Get(context.Background(), &levelgraph.Pattern{Subject: []byte(fileKey)})
	for _, t := range results {
		pred := string(t.Predicate)
		if strings.HasPrefix(pred, "text:links:") {
			db.Del(context.Background(), t)
		}
	}

	// Find markdown links: [text](url) or [text][ref]
	linkRegex := regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		matches := linkRegex.FindAllStringSubmatchIndex(line, -1)
		for _, match := range matches {
			if len(match) >= 6 {
				col := match[0]
				url := line[match[4]:match[5]]
				predicate := fmt.Sprintf("text:links:%d:%d", lineNum+1, col+1)
				db.Put(context.Background(), levelgraph.NewTripleFromStrings(fileKey, predicate, url))
			}
		}
	}
}

func syncCodeBlocks(db *levelgraph.DB, fileKey, content string) {
	// Remove old codeblock predicates
	results, _ := db.Get(context.Background(), &levelgraph.Pattern{Subject: []byte(fileKey)})
	for _, t := range results {
		pred := string(t.Predicate)
		if strings.HasPrefix(pred, "text:includes:") {
			db.Del(context.Background(), t)
		}
	}

	// Find fenced code blocks
	lines := strings.Split(content, "\n")
	inBlock := false
	var blockStart int
	var blockInfo string
	var blockContent strings.Builder

	for i, line := range lines {
		if strings.HasPrefix(line, "```") {
			if !inBlock {
				// Start of block
				inBlock = true
				blockStart = i + 1
				blockInfo = strings.TrimPrefix(line, "```")
				blockInfo = strings.TrimSpace(blockInfo)
				blockContent.Reset()
			} else {
				// End of block
				inBlock = false
				blockEnd := i + 1

				contentHash := hashString(blockContent.String())
				codeblockKey := "codeblock:" + contentHash

				// Add file -> codeblock relationship
				predicate := fmt.Sprintf("text:includes:%d:%d", blockStart, blockEnd)
				db.Put(context.Background(), levelgraph.NewTripleFromStrings(fileKey, predicate, codeblockKey))

				// Add codeblock metadata if present
				if blockInfo != "" {
					db.Put(context.Background(), levelgraph.NewTripleFromStrings(codeblockKey, "codeblock:has meta:raw", blockInfo))
				}
			}
		} else if inBlock {
			if blockContent.Len() > 0 {
				blockContent.WriteString("\n")
			}
			blockContent.WriteString(line)
		}
	}
}

func cmdStats() {
	db, err := openDB()
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	results, err := db.Get(context.Background(), &levelgraph.Pattern{})
	if err != nil {
		fmt.Printf("Error querying: %v\n", err)
		os.Exit(1)
	}

	// Count unique subjects, predicates, objects
	subjects := make(map[string]bool)
	predicates := make(map[string]bool)
	objects := make(map[string]bool)

	for _, t := range results {
		subjects[string(t.Subject)] = true
		predicates[string(t.Predicate)] = true
		objects[string(t.Object)] = true
	}

	fmt.Printf("Database: %s\n", dbPath)
	fmt.Printf("Triples:    %d\n", len(results))
	fmt.Printf("Subjects:   %d unique\n", len(subjects))
	fmt.Printf("Predicates: %d unique\n", len(predicates))
	fmt.Printf("Objects:    %d unique\n", len(objects))
}

func cmdDump() {
	db, err := openDB()
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	results, err := db.Get(context.Background(), &levelgraph.Pattern{})
	if err != nil {
		fmt.Printf("Error querying: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println("Database is empty.")
		return
	}

	for _, t := range results {
		fmt.Printf("%s → %s → %s\n", t.Subject, t.Predicate, t.Object)
	}
	fmt.Printf("\n(%d triples)\n", len(results))
}

func cmdNuke() {
	fmt.Printf("This will delete the database at %s\n", dbPath)
	fmt.Print("Are you sure? Type 'yes' to confirm: ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(response)

	if response != "yes" {
		fmt.Println("Aborted.")
		return
	}

	if err := os.RemoveAll(dbPath); err != nil {
		fmt.Printf("Error deleting database: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("💥 Database deleted.")
}
