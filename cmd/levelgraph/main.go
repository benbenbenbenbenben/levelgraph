package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/benbenbenbenbenben/levelgraph"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "put":
		runPut(args)
	case "get":
		runGet(args)
	case "dump":
		runDump(args)
	case "load":
		runLoad(args)
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`LevelGraph CLI

Usage:
  levelgraph <command> [arguments]

Commands:
  put <subject> <predicate> <object>   Add a triple
  get <subject> <predicate> <object>   Get triples (use '*' as wildcard)
  dump                                 Dump all triples
  load <file>                          Load triples from a file (N-Triples format)

Global Flags:
  -db <path>                           Path to database (default: levelgraph.db)
`)
}

func parseFlags(args []string) (*levelgraph.DB, []string) {
	fs := flag.NewFlagSet("levelgraph", flag.ExitOnError)
	dbPath := fs.String("db", "levelgraph.db", "Path to database")
	fs.Parse(args)

	db, err := levelgraph.Open(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	return db, fs.Args()
}

func runPut(args []string) {
	db, remaining := parseFlags(args)
	defer db.Close()

	if len(remaining) != 3 {
		log.Fatal("Usage: levelgraph put <subject> <predicate> <object>")
	}

	err := db.Put(context.Background(), levelgraph.NewTripleFromStrings(remaining[0], remaining[1], remaining[2]))
	if err != nil {
		log.Fatalf("Failed to put triple: %v", err)
	}
	fmt.Println("Triple added.")
}

func runGet(args []string) {
	db, remaining := parseFlags(args)
	defer db.Close()

	if len(remaining) != 3 {
		log.Fatal("Usage: levelgraph get <subject> <predicate> <object> (use '*' for wildcard)")
	}

	parsePart := func(s string) []byte {
		if s == "*" {
			return nil
		}
		return []byte(s)
	}

	pattern := &levelgraph.Pattern{
		Subject:   parsePart(remaining[0]),
		Predicate: parsePart(remaining[1]),
		Object:    parsePart(remaining[2]),
	}

	triples, err := db.Get(context.Background(), pattern)
	if err != nil {
		log.Fatalf("Failed to get triples: %v", err)
	}

	for _, t := range triples {
		fmt.Printf("%s %s %s\n", t.Subject, t.Predicate, t.Object)
	}
}

func runDump(args []string) {
	db, _ := parseFlags(args)
	defer db.Close()

	triples, err := db.Get(context.Background(), &levelgraph.Pattern{})
	if err != nil {
		log.Fatalf("Failed to dump triples: %v", err)
	}

	for _, t := range triples {
		fmt.Printf("%s %s %s\n", t.Subject, t.Predicate, t.Object)
	}
}

func runLoad(args []string) {
	db, remaining := parseFlags(args)
	defer db.Close()

	if len(remaining) != 1 {
		log.Fatal("Usage: levelgraph load <file>")
	}

	filePath := remaining[0]
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
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
				log.Printf("Failed to put triple '%s': %v", line, err)
			} else {
				count++
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	fmt.Printf("Loaded %d triples.\n", count)
}
