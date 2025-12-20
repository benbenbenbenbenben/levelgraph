# HNSW Demo: Semantic Search for Knowledge Graphs

This demo showcases how HNSW (Hierarchical Navigable Small World) vector search can heal disconnected knowledge in a graph database. By using embeddings, we can find semantically related content even when there are no explicit graph links.

## Getting Started

You'll need to use `nolij` from the `example/nolij` directory:

```bash
# Build nolij (if not already built)
cd ../nolij
go build .

# Return to hnsw-demo and run nolij
cd ../hnsw-demo

# Sync all markdown files into the graph
../nolij/nolij sync

# Check what was indexed
../nolij/nolij stats
```

## The HNSW Advantage

Traditional graph databases only find connections through explicit links. But what if your knowledge base has gaps? Consider:

- A document about **Serena Williams** mentions "tennis" and "Grand Slam victories"
- A document about **Venus Williams** mentions "tennis" and "Olympic gold"  
- A document about **Ada Lovelace** mentions "first programmer" and "Charles Babbage"
- A document about **Grace Hopper** mentions "COBOL" and "compiler"

Even without explicit links between these documents, HNSW can find that:
- Serena and Venus are semantically similar (both tennis champions)
- Ada and Grace are semantically similar (both computing pioneers)

This is the power of **healing disconnected knowledge** with embeddings.

## Demo Data Structure

This demo contains a deliberately discontinuous knowledge base:

| File | Description |
|------|-------------|
| [PEOPLE.md](./PEOPLE.md) | 20 notable people: 10 sports figures + 10 CS pioneers |
| [SPORTS.md](./SPORTS.md) | 25+ sports and athletic activities |
| [HOBBIES.md](./HOBBIES.md) | 25+ hobbies and pastimes |
| [CAREERS.md](./CAREERS.md) | 40+ career paths and professions |

Each person has their own detailed markdown file in `PEOPLE/` with varying levels of detail and formatting - intentionally inconsistent to demonstrate how semantic search can still find relevant connections.

## Example Commands

```bash
# Sync the knowledge base
../nolij/nolij sync

# Find all indexed files
../nolij/nolij find nolij:root contains:file ?

# Find links from PEOPLE.md
../nolij/nolij find file:PEOPLE.md 'text:links:*' ?

# Database statistics
../nolij/nolij stats

# Dump all triples
../nolij/nolij dump
```

## How HNSW Works

HNSW builds a multi-layer graph where:
1. Each document gets converted to a dense vector (embedding)
2. Similar documents are connected in a navigable graph structure
3. Queries traverse this graph to find approximate nearest neighbors in O(log n) time

Key parameters:
- **M**: Maximum connections per node (more = better recall, slower build)
- **efConstruction**: Build-time search depth (higher = better graph quality)
- **efSearch**: Query-time search depth (higher = better recall, slower queries)

## Why This Matters

In real-world knowledge bases:
- Documents are created by different people at different times
- Connections between related concepts are often missing
- Traditional keyword search misses semantic relationships

HNSW + embeddings solve this by understanding **meaning**, not just matching words.
