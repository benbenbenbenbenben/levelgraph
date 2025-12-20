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

package vector

import (
	"container/heap"
	"math"
	"math/rand"
	"sync"
)

// HNSWIndex implements Hierarchical Navigable Small World graphs for
// approximate nearest neighbor search. It provides O(log n) search time
// with high recall rates, making it ideal for large-scale similarity search.
//
// # When to Use HNSW vs Flat Index
//
//   - Use FlatIndex for datasets < 10,000 vectors or when 100% recall is required
//   - Use HNSWIndex for datasets > 10,000 vectors or when speed is critical
//
// # Parameter Tuning Guide
//
// The three main parameters control the trade-off between speed, recall, and memory:
//
// M (connections per layer):
//   - Controls how many connections each node maintains in the graph
//   - Higher M = better recall, but more memory and slower construction
//   - Recommended: 12-48 (default: 16)
//   - Rule of thumb: M=16 works well for most cases; increase to 32-48 for
//     high-dimensional vectors (>256 dims) or when recall must be >99%
//
// efConstruction (construction quality):
//   - Controls the size of the candidate list during index building
//   - Higher values = better index quality, but slower construction
//   - Recommended: 100-500 (default: 200)
//   - Rule of thumb: efConstruction >= 2*M; increase for better recall
//
// efSearch (search quality):
//   - Controls the size of the candidate list during search
//   - Higher values = better recall, but slower search
//   - Recommended: 50-500 (default: 50)
//   - Can be changed per-query via SearchWithEf
//   - Rule of thumb: Start with efSearch = 2*M, increase until recall is acceptable
//
// # Memory Usage
//
// Approximate memory per vector: ~(4*dims + 8*M*avgLevel) bytes
// For 192-dim vectors with M=16: ~1KB per vector
//
// # Example Configurations
//
//	// High-speed, lower recall (~95%)
//	index := vector.NewHNSWIndex(192,
//	    vector.WithM(12),
//	    vector.WithEfConstruction(100),
//	    vector.WithEfSearch(30),
//	)
//
//	// Balanced (default, ~98% recall)
//	index := vector.NewHNSWIndex(192,
//	    vector.WithM(16),
//	    vector.WithEfConstruction(200),
//	    vector.WithEfSearch(50),
//	)
//
//	// High-recall (~99.5%)
//	index := vector.NewHNSWIndex(192,
//	    vector.WithM(32),
//	    vector.WithEfConstruction(400),
//	    vector.WithEfSearch(200),
//	)
//
// # Persistence
//
// HNSW graphs can be exported and imported for persistence:
//
//	// Export graph structure
//	data := index.Export()
//	bytes, _ := json.Marshal(data)
//
//	// Later, import into a new index
//	newIndex := vector.NewHNSWIndex(192)
//	newIndex.Import(data)
//
// # Thread Safety
//
// HNSWIndex is safe for concurrent use. Multiple goroutines can search
// simultaneously. Write operations (Add, Delete) acquire exclusive locks.
//
// Reference: "Efficient and robust approximate nearest neighbor search
// using Hierarchical Navigable Small World graphs" by Malkov & Yashunin
type HNSWIndex struct {
	dimensions int
	distance   DistanceFunc

	// HNSW parameters
	m              int     // Number of connections per layer
	mMax           int     // Maximum connections for non-zero layers
	mMax0          int     // Maximum connections for layer 0
	efConstruction int     // Size of dynamic candidate list during construction
	efSearch       int     // Size of dynamic candidate list during search
	levelMult      float64 // Level generation multiplier

	// Graph structure
	nodes      map[string]*hnswNode
	entryPoint *hnswNode
	maxLevel   int

	mu    sync.RWMutex
	rng   *rand.Rand
	rngMu sync.Mutex
}

// hnswNode represents a node in the HNSW graph.
type hnswNode struct {
	id      string
	vector  []float32
	level   int
	friends []map[string]*hnswNode // friends[level] = connected nodes at that level
}

// HNSWOption configures an HNSWIndex.
type HNSWOption func(*HNSWIndex)

// WithM sets the number of bi-directional links created for each element.
// Higher values lead to better recall but slower construction and larger memory.
// Default: 16, recommended range: 12-48
func WithM(m int) HNSWOption {
	return func(h *HNSWIndex) {
		h.m = m
		h.mMax = m
		h.mMax0 = m * 2
	}
}

// WithEfConstruction sets the size of the dynamic candidate list during construction.
// Higher values lead to better index quality but slower construction.
// Default: 200, should be >= 2*M
func WithEfConstruction(ef int) HNSWOption {
	return func(h *HNSWIndex) {
		h.efConstruction = ef
	}
}

// WithEfSearch sets the size of the dynamic candidate list during search.
// Higher values lead to better recall but slower search.
// Default: 50, can be changed per-query
func WithEfSearch(ef int) HNSWOption {
	return func(h *HNSWIndex) {
		h.efSearch = ef
	}
}

// WithHNSWDistance sets the distance function for the HNSW index.
// Default is Cosine distance.
func WithHNSWDistance(fn DistanceFunc) HNSWOption {
	return func(h *HNSWIndex) {
		h.distance = fn
	}
}

// WithSeed sets the random seed for reproducible level generation.
func WithSeed(seed int64) HNSWOption {
	return func(h *HNSWIndex) {
		h.rng = rand.New(rand.NewSource(seed))
	}
}

// NewHNSWIndex creates a new HNSW index for approximate nearest neighbor search.
func NewHNSWIndex(dimensions int, opts ...HNSWOption) *HNSWIndex {
	h := &HNSWIndex{
		dimensions:     dimensions,
		distance:       Cosine,
		m:              16,
		mMax:           16,
		mMax0:          32,
		efConstruction: 200,
		efSearch:       50,
		nodes:          make(map[string]*hnswNode),
		maxLevel:       -1,
		rng:            rand.New(rand.NewSource(rand.Int63())),
	}

	for _, opt := range opts {
		opt(h)
	}

	// Calculate level multiplier: 1/ln(M)
	h.levelMult = 1.0 / math.Log(float64(h.m))

	return h
}

// Add adds or updates a vector with the given ID.
func (h *HNSWIndex) Add(id []byte, vector []float32) error {
	if len(vector) == 0 {
		return ErrEmptyVector
	}
	if len(vector) != h.dimensions {
		return ErrDimensionMismatch
	}

	// Make a copy
	v := make([]float32, len(vector))
	copy(v, vector)

	idStr := string(id)

	h.mu.Lock()
	defer h.mu.Unlock()

	// Check if updating existing node
	if existing, exists := h.nodes[idStr]; exists {
		// If the vector hasn't changed significantly, just update in place
		// Otherwise, delete and re-add to rebuild connections
		oldDist := h.distance(existing.vector, v)
		if oldDist > 0.1 { // Threshold for significant change (cosine distance > 0.1)
			// Significant change - delete and re-add to rebuild connections
			h.deleteUnlocked(idStr)
		} else {
			// Minor change - just update vector in place
			existing.vector = v
			return nil
		}
	}

	// Generate random level
	level := h.randomLevel()

	// Create new node
	node := &hnswNode{
		id:      idStr,
		vector:  v,
		level:   level,
		friends: make([]map[string]*hnswNode, level+1),
	}
	for i := 0; i <= level; i++ {
		node.friends[i] = make(map[string]*hnswNode)
	}

	h.nodes[idStr] = node

	// If this is the first node, set as entry point
	if h.entryPoint == nil {
		h.entryPoint = node
		h.maxLevel = level
		return nil
	}

	// Find entry point for insertion
	ep := h.entryPoint
	currentMaxLevel := h.maxLevel

	// Traverse from top level to node's level + 1, finding closest entry point
	for lc := currentMaxLevel; lc > level; lc-- {
		ep = h.searchLayerClosest(v, ep, lc)
	}

	// For each level from min(level, maxLevel) down to 0, find and connect neighbors
	for lc := min(level, currentMaxLevel); lc >= 0; lc-- {
		neighbors := h.searchLayer(v, ep, h.efConstruction, lc)

		// Select M best neighbors
		mMax := h.mMax
		if lc == 0 {
			mMax = h.mMax0
		}
		selectedNeighbors := h.selectNeighborsSimple(neighbors, mMax)

		// Connect node to neighbors (bidirectional)
		for _, neighbor := range selectedNeighbors {
			node.friends[lc][neighbor.id] = neighbor
			neighbor.friends[lc][node.id] = node

			// Shrink neighbor connections if needed
			if len(neighbor.friends[lc]) > mMax {
				h.shrinkConnections(neighbor, lc, mMax)
			}
		}

		if len(neighbors) > 0 {
			ep = neighbors[0]
		}
	}

	// Update entry point if new node has higher level
	if level > h.maxLevel {
		h.entryPoint = node
		h.maxLevel = level
	}

	return nil
}

// Delete removes a vector by ID.
func (h *HNSWIndex) Delete(id []byte) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.deleteUnlocked(string(id))
}

// deleteUnlocked is the internal delete implementation without locking.
// Caller must hold h.mu.Lock().
func (h *HNSWIndex) deleteUnlocked(idStr string) error {
	node, exists := h.nodes[idStr]
	if !exists {
		return ErrNotFound
	}

	// For each level, collect the orphaned neighbors and reconnect them
	for lc := 0; lc <= node.level; lc++ {
		orphanedNeighbors := make([]*hnswNode, 0, len(node.friends[lc]))
		for _, friend := range node.friends[lc] {
			// Remove the connection to the deleted node
			delete(friend.friends[lc], idStr)
			orphanedNeighbors = append(orphanedNeighbors, friend)
		}

		// Reconnect orphaned neighbors to maintain graph connectivity
		// For each orphaned neighbor, try to connect it to other orphaned neighbors
		// that it's not already connected to
		h.repairConnections(orphanedNeighbors, lc)
	}

	delete(h.nodes, idStr)

	// Update entry point if needed
	if h.entryPoint == node {
		h.entryPoint = nil
		h.maxLevel = -1

		// Find new entry point (node with highest level)
		for _, n := range h.nodes {
			if h.entryPoint == nil || n.level > h.maxLevel {
				h.entryPoint = n
				h.maxLevel = n.level
			}
		}
	}

	return nil
}

// repairConnections reconnects orphaned neighbors after a node deletion.
// For each orphan, it tries to find new connections among the other orphans
// or by searching for nearby nodes.
func (h *HNSWIndex) repairConnections(orphans []*hnswNode, level int) {
	if len(orphans) < 2 {
		return
	}

	mMax := h.mMax
	if level == 0 {
		mMax = h.mMax0
	}

	// For each orphan, try to connect to other orphans that are close
	for i, orphan := range orphans {
		// Skip if this orphan already has enough connections
		if len(orphan.friends[level]) >= mMax {
			continue
		}

		// Find the best candidates among other orphans
		type candidate struct {
			node *hnswNode
			dist float32
		}
		candidates := make([]candidate, 0, len(orphans)-1)

		for j, other := range orphans {
			if i == j {
				continue
			}
			// Skip if already connected
			if _, connected := orphan.friends[level][other.id]; connected {
				continue
			}
			// Skip if other already has max connections
			if len(other.friends[level]) >= mMax {
				continue
			}

			dist := h.distance(orphan.vector, other.vector)
			candidates = append(candidates, candidate{other, dist})
		}

		// Sort candidates by distance
		for x := 0; x < len(candidates)-1; x++ {
			for y := x + 1; y < len(candidates); y++ {
				if candidates[y].dist < candidates[x].dist {
					candidates[x], candidates[y] = candidates[y], candidates[x]
				}
			}
		}

		// Add connections to closest candidates
		slotsAvailable := mMax - len(orphan.friends[level])
		for _, c := range candidates {
			if slotsAvailable <= 0 {
				break
			}
			// Check if other still has room
			if len(c.node.friends[level]) >= mMax {
				continue
			}

			// Create bidirectional connection
			orphan.friends[level][c.node.id] = c.node
			c.node.friends[level][orphan.id] = orphan
			slotsAvailable--
		}
	}
}

// Get retrieves a vector by ID.
func (h *HNSWIndex) Get(id []byte) ([]float32, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	node, exists := h.nodes[string(id)]
	if !exists {
		return nil, ErrNotFound
	}

	result := make([]float32, len(node.vector))
	copy(result, node.vector)
	return result, nil
}

// Search finds the k nearest vectors to the query.
func (h *HNSWIndex) Search(query []float32, k int) ([]Match, error) {
	return h.SearchWithEf(query, k, h.efSearch)
}

// SearchWithEf finds the k nearest vectors with a custom ef parameter.
func (h *HNSWIndex) SearchWithEf(query []float32, k int, ef int) ([]Match, error) {
	if k <= 0 {
		return nil, ErrInvalidK
	}
	if len(query) != h.dimensions {
		return nil, ErrDimensionMismatch
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.entryPoint == nil {
		return []Match{}, nil
	}

	// Start from entry point
	ep := h.entryPoint

	// Traverse from top level to level 1
	for lc := h.maxLevel; lc > 0; lc-- {
		ep = h.searchLayerClosest(query, ep, lc)
	}

	// Search layer 0 with ef candidates
	candidates := h.searchLayer(query, ep, max(ef, k), 0)

	// Return top k
	results := make([]Match, 0, min(k, len(candidates)))
	for i := 0; i < k && i < len(candidates); i++ {
		dist := h.distance(query, candidates[i].vector)
		results = append(results, Match{
			ID:       []byte(candidates[i].id),
			Distance: dist,
			Score:    NormalizeScore(dist),
		})
	}

	return results, nil
}

// Len returns the number of vectors in the index.
func (h *HNSWIndex) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.nodes)
}

// Dimensions returns the vector dimensionality.
func (h *HNSWIndex) Dimensions() int {
	return h.dimensions
}

// randomLevel generates a random level for a new node.
func (h *HNSWIndex) randomLevel() int {
	h.rngMu.Lock()
	r := h.rng.Float64()
	h.rngMu.Unlock()

	return int(math.Floor(-math.Log(r) * h.levelMult))
}

// searchLayerClosest finds the single closest node in a layer.
func (h *HNSWIndex) searchLayerClosest(query []float32, entry *hnswNode, level int) *hnswNode {
	current := entry
	currentDist := h.distance(query, current.vector)

	for {
		changed := false
		for _, friend := range current.friends[level] {
			dist := h.distance(query, friend.vector)
			if dist < currentDist {
				current = friend
				currentDist = dist
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	return current
}

// searchLayer performs a beam search in a layer, returning ef closest nodes.
func (h *HNSWIndex) searchLayer(query []float32, entry *hnswNode, ef int, level int) []*hnswNode {
	visited := make(map[string]bool)
	visited[entry.id] = true

	// candidates is a min-heap (closest first)
	candidates := &nodeHeap{
		nodes:   []*hnswNode{entry},
		dists:   []float32{h.distance(query, entry.vector)},
		maxHeap: false,
	}
	heap.Init(candidates)

	// results is a max-heap (farthest first, for easy removal)
	results := &nodeHeap{
		nodes:   []*hnswNode{entry},
		dists:   []float32{h.distance(query, entry.vector)},
		maxHeap: true,
	}
	heap.Init(results)

	for candidates.Len() > 0 {
		// Get closest candidate
		closestIdx := 0
		closest := candidates.nodes[closestIdx]
		closestDist := candidates.dists[closestIdx]
		heap.Remove(candidates, closestIdx)

		// Get farthest result
		farthestDist := results.dists[0]

		// If closest candidate is farther than farthest result, stop
		if closestDist > farthestDist {
			break
		}

		// Explore neighbors
		for _, neighbor := range closest.friends[level] {
			if visited[neighbor.id] {
				continue
			}
			visited[neighbor.id] = true

			dist := h.distance(query, neighbor.vector)
			farthestDist = results.dists[0]

			if dist < farthestDist || results.Len() < ef {
				heap.Push(candidates, nodeEntry{neighbor, dist})
				heap.Push(results, nodeEntry{neighbor, dist})

				if results.Len() > ef {
					heap.Pop(results)
				}
			}
		}
	}

	// Extract results sorted by distance (ascending)
	result := make([]*hnswNode, results.Len())
	for i := len(result) - 1; i >= 0; i-- {
		entry := heap.Pop(results).(nodeEntry)
		result[i] = entry.node
	}

	return result
}

// selectNeighborsSimple selects the M closest neighbors.
func (h *HNSWIndex) selectNeighborsSimple(candidates []*hnswNode, m int) []*hnswNode {
	if len(candidates) <= m {
		return candidates
	}
	return candidates[:m]
}

// shrinkConnections reduces a node's connections to the maximum allowed.
func (h *HNSWIndex) shrinkConnections(node *hnswNode, level int, maxConn int) {
	if len(node.friends[level]) <= maxConn {
		return
	}

	// Collect all friends with distances
	type friendDist struct {
		friend *hnswNode
		dist   float32
	}
	friends := make([]friendDist, 0, len(node.friends[level]))
	for _, friend := range node.friends[level] {
		friends = append(friends, friendDist{
			friend: friend,
			dist:   h.distance(node.vector, friend.vector),
		})
	}

	// Sort by distance
	for i := 0; i < len(friends)-1; i++ {
		for j := i + 1; j < len(friends); j++ {
			if friends[j].dist < friends[i].dist {
				friends[i], friends[j] = friends[j], friends[i]
			}
		}
	}

	// Keep only the closest maxConn
	newFriends := make(map[string]*hnswNode)
	for i := 0; i < maxConn && i < len(friends); i++ {
		newFriends[friends[i].friend.id] = friends[i].friend
	}

	// Remove connections that were dropped
	for id, friend := range node.friends[level] {
		if _, kept := newFriends[id]; !kept {
			delete(friend.friends[level], node.id)
		}
	}

	node.friends[level] = newFriends
}

// nodeEntry pairs a node with its distance for heap operations.
type nodeEntry struct {
	node *hnswNode
	dist float32
}

// nodeHeap is a heap of nodes, configurable as min or max heap.
type nodeHeap struct {
	nodes   []*hnswNode
	dists   []float32
	maxHeap bool
}

func (h *nodeHeap) Len() int { return len(h.nodes) }

func (h *nodeHeap) Less(i, j int) bool {
	if h.maxHeap {
		return h.dists[i] > h.dists[j]
	}
	return h.dists[i] < h.dists[j]
}

func (h *nodeHeap) Swap(i, j int) {
	h.nodes[i], h.nodes[j] = h.nodes[j], h.nodes[i]
	h.dists[i], h.dists[j] = h.dists[j], h.dists[i]
}

func (h *nodeHeap) Push(x any) {
	entry := x.(nodeEntry)
	h.nodes = append(h.nodes, entry.node)
	h.dists = append(h.dists, entry.dist)
}

func (h *nodeHeap) Pop() any {
	n := len(h.nodes)
	node := h.nodes[n-1]
	dist := h.dists[n-1]
	h.nodes = h.nodes[:n-1]
	h.dists = h.dists[:n-1]
	return nodeEntry{node, dist}
}

// ============================================================================
// HNSW Persistence
// ============================================================================

// HNSWData represents the serializable state of an HNSW index.
// This can be used to save and restore the graph structure without
// rebuilding from vectors.
type HNSWData struct {
	// Parameters
	Dimensions     int
	M              int
	MMax           int
	MMax0          int
	EfConstruction int
	EfSearch       int

	// Graph state
	MaxLevel     int
	EntryPointID string
	Nodes        []HNSWNodeData
}

// HNSWNodeData represents a serializable node in the HNSW graph.
type HNSWNodeData struct {
	ID      string
	Vector  []float32
	Level   int
	Friends [][]string // Friends[level] = list of connected node IDs
}

// Export exports the HNSW index state for persistence.
// The returned HNSWData can be serialized (e.g., with encoding/gob or JSON)
// and later restored with Import.
func (h *HNSWIndex) Export() *HNSWData {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data := &HNSWData{
		Dimensions:     h.dimensions,
		M:              h.m,
		MMax:           h.mMax,
		MMax0:          h.mMax0,
		EfConstruction: h.efConstruction,
		EfSearch:       h.efSearch,
		MaxLevel:       h.maxLevel,
		Nodes:          make([]HNSWNodeData, 0, len(h.nodes)),
	}

	if h.entryPoint != nil {
		data.EntryPointID = h.entryPoint.id
	}

	// Export each node
	for _, node := range h.nodes {
		nodeData := HNSWNodeData{
			ID:      node.id,
			Vector:  make([]float32, len(node.vector)),
			Level:   node.level,
			Friends: make([][]string, len(node.friends)),
		}
		copy(nodeData.Vector, node.vector)

		// Export friends at each level
		for level, friends := range node.friends {
			nodeData.Friends[level] = make([]string, 0, len(friends))
			for friendID := range friends {
				nodeData.Friends[level] = append(nodeData.Friends[level], friendID)
			}
		}

		data.Nodes = append(data.Nodes, nodeData)
	}

	return data
}

// Import restores the HNSW index state from previously exported data.
// This is much faster than rebuilding the index by re-inserting vectors,
// as it directly restores the graph structure.
//
// Note: The index should be empty before calling Import. Any existing
// nodes will be replaced.
func (h *HNSWIndex) Import(data *HNSWData) error {
	if data == nil {
		return ErrEmptyVector
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Validate dimensions match
	if data.Dimensions != h.dimensions {
		return ErrDimensionMismatch
	}

	// Restore parameters (optional - could validate they match instead)
	h.m = data.M
	h.mMax = data.MMax
	h.mMax0 = data.MMax0
	h.efConstruction = data.EfConstruction
	h.efSearch = data.EfSearch
	h.maxLevel = data.MaxLevel
	h.levelMult = 1.0 / math.Log(float64(h.m))

	// Clear existing nodes
	h.nodes = make(map[string]*hnswNode, len(data.Nodes))
	h.entryPoint = nil

	// First pass: create all nodes without connections
	for _, nodeData := range data.Nodes {
		node := &hnswNode{
			id:      nodeData.ID,
			vector:  make([]float32, len(nodeData.Vector)),
			level:   nodeData.Level,
			friends: make([]map[string]*hnswNode, nodeData.Level+1),
		}
		copy(node.vector, nodeData.Vector)

		// Initialize friend maps
		for level := 0; level <= nodeData.Level; level++ {
			node.friends[level] = make(map[string]*hnswNode)
		}

		h.nodes[nodeData.ID] = node
	}

	// Set entry point
	if data.EntryPointID != "" {
		h.entryPoint = h.nodes[data.EntryPointID]
	}

	// Second pass: restore connections
	for _, nodeData := range data.Nodes {
		node := h.nodes[nodeData.ID]
		for level, friendIDs := range nodeData.Friends {
			for _, friendID := range friendIDs {
				if friend, exists := h.nodes[friendID]; exists {
					node.friends[level][friendID] = friend
				}
			}
		}
	}

	return nil
}

// Ensure HNSWIndex implements Index.
var _ Index = (*HNSWIndex)(nil)
