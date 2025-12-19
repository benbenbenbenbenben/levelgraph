# Nolij

A knowledge graph CLI built on LevelGraph for indexing and querying relationships in your codebase.

## Installation

```bash
cd example/nolij
go build .
```

## Usage

```
nolij <command> [arguments]
```

### Commands

| Command | Description |
|---------|-------------|
| `add <s> <p> <o>` | Add a triple to the graph |
| `del <s> <p> <o>` | Delete a triple from the graph |
| `find <s> <p> <o>` | Search for triples (use `?` or `*` as wildcard) |
| `from <node>` | Follow all edges from a node |
| `path <start> <end>` | Find shortest path between two nodes (BFS) |
| `join <s1> <p1> <o1> <s2> <p2> <o2>` | Join two patterns using variables |
| `sync` | Index markdown files in current directory |
| `stats` | Show database statistics |
| `dump` | Print all triples in the database |
| `nuke` | Delete the database (with confirmation) |

## Examples

### Basic Triple Operations

```bash
# Add facts about people
nolij add alice knows bob
nolij add bob works_at acme
nolij add acme located_in london

# Query relationships
nolij find alice ? ?          # All facts about alice
nolij find ? knows ?          # All "knows" relationships
nolij find ? ? london         # Everything related to london

# Follow edges from a node
nolij from alice
# Output:
#   -knows-> bob

# Find path between nodes
nolij path alice london
# Output:
#   [alice] -knows-> [bob] -works_at-> [acme] -located_in-> [london]
```

### Join Queries with Variables

Use `$varname` to create join variables that connect patterns:

```bash
# Find files that contain bash code blocks
nolij join 'file:README.md' '$pred' '$block' '$block' 'codeblock:has meta:raw' 'bash'

# Find all files and their code block languages
nolij join '$file' '$pred' '$block' '$block' 'codeblock:has meta:raw' '$lang'
```

### Markdown Sync

The `sync` command indexes markdown files, extracting:
- File metadata (SHA256 hash for change detection)
- Markdown links with line/column positions
- Fenced code blocks with language info

```bash
# Index all .md files in current directory (recursive)
nolij sync

# Output:
# Found 5 markdown file(s)
#
# Pass 1: File discovery and hashing...
#   + README.md
#   + docs/guide.md
#   ✓ docs/api.md (unchanged)
#   ↻ CHANGELOG.md (updated)
#
# Sync complete: 2 new, 1 unchanged, 1 updated
```

#### Sync Schema

Files are indexed with these triple patterns:

```
nolij:root           contains:file    file:<path>
file:<path>          has:sha256       <hash>
file:<path>          text:links:<line>:<col>    <url>
file:<path>          text:includes:<start>:<end>    codeblock:<hash>
codeblock:<hash>     codeblock:has meta:raw    <language>
```

#### Querying Synced Data

```bash
# List all indexed files
nolij find nolij:root contains:file ?

# Find all links in a file
nolij find file:README.md 'text:links:*' ?

# Find files containing Go code
nolij join '$file' '$p' '$block' '$block' 'codeblock:has meta:raw' 'go'

# Database overview
nolij stats
```

### Database Management

```bash
# View statistics
nolij stats
# Output:
#   Database: .nolij.db
#   Triples:    142
#   Subjects:   45 unique
#   Predicates: 12 unique
#   Objects:    89 unique

# Dump all triples
nolij dump

# Delete database (asks for confirmation)
nolij nuke
```

## Database Location

The database is stored in `.nolij.db/` in the current working directory. Add this to your `.gitignore`:

```
.nolij.db/
```

## Use Cases

- **Code documentation graphs**: Link concepts, functions, and documentation
- **Dependency tracking**: Model relationships between components
- **Knowledge bases**: Build queryable fact stores
- **Markdown wikis**: Index and query interconnected documents
