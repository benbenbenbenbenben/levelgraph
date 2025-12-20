// LevelGraph Playground JavaScript

// State
let wasmReady = false;
let currentBuild = 'tinygo'; // default to smaller TinyGo build

// Graph visualization state
let graphData = { nodes: [], links: [] };
let graphSvg = null;
let graphSimulation = null;
let graphZoom = null;
let showEdgeLabels = true;
let highlightedNodes = new Set();
let highlightedLinks = new Set();

// WASM build configurations
const wasmBuilds = {
    tinygo: {
        wasm: 'levelgraph-tinygo.wasm',
        exec: 'wasm_exec_tinygo.js'
    },
    standard: {
        wasm: 'levelgraph.wasm',
        exec: 'wasm_exec.js'
    }
};

// Example code snippets
const examples = {
    basic: `// Basic CRUD Operations

// Insert triples
put([
    { subject: "alice", predicate: "name", object: "Alice Smith" },
    { subject: "alice", predicate: "age", object: "30" },
    { subject: "bob", predicate: "name", object: "Bob Jones" },
    { subject: "bob", predicate: "age", object: "25" }
]);

// Get all triples about alice
log("All data about alice:");
get({ subject: "alice" });

// Get all name predicates
log("\\nAll names:");
get({ predicate: "name" });

// Delete a triple
del([{ subject: "alice", predicate: "age", object: "30" }]);

// Verify deletion
log("\\nAlice after deleting age:");
get({ subject: "alice" });
`,

    social: `// Social Network Example

// Build a social graph
put([
    // People
    { subject: "alice", predicate: "type", object: "Person" },
    { subject: "bob", predicate: "type", object: "Person" },
    { subject: "charlie", predicate: "type", object: "Person" },
    { subject: "diana", predicate: "type", object: "Person" },
    
    // Friendships (symmetric, so add both directions)
    { subject: "alice", predicate: "friend", object: "bob" },
    { subject: "bob", predicate: "friend", object: "alice" },
    { subject: "alice", predicate: "friend", object: "charlie" },
    { subject: "charlie", predicate: "friend", object: "alice" },
    { subject: "bob", predicate: "friend", object: "diana" },
    { subject: "diana", predicate: "friend", object: "bob" },
    
    // Interests
    { subject: "alice", predicate: "likes", object: "hiking" },
    { subject: "alice", predicate: "likes", object: "photography" },
    { subject: "bob", predicate: "likes", object: "hiking" },
    { subject: "charlie", predicate: "likes", object: "music" },
    { subject: "diana", predicate: "likes", object: "photography" }
]);

// Find alice's friends
log("Alice's friends:");
get({ subject: "alice", predicate: "friend" });

// Find who likes hiking
log("\\nPeople who like hiking:");
search([
    { subject: "?person", predicate: "likes", object: "hiking" }
]);

// Find people who share ANY interest with alice (excluding alice herself)
log("\\nPeople who share interests with alice:");
search([
    { subject: "alice", predicate: "likes", object: "?interest" },
    { subject: "?other", predicate: "likes", object: "?interest" }
], { notEqual: [{ var: "other", value: "alice" }] });
`,

    search: `// Pattern Search with Variables

// Create a knowledge graph
put([
    // Categories
    { subject: "programming", predicate: "type", object: "Category" },
    { subject: "databases", predicate: "type", object: "Category" },
    
    // Programming languages
    { subject: "go", predicate: "category", object: "programming" },
    { subject: "javascript", predicate: "category", object: "programming" },
    { subject: "python", predicate: "category", object: "programming" },
    
    // Databases
    { subject: "leveldb", predicate: "category", object: "databases" },
    { subject: "postgres", predicate: "category", object: "databases" },
    
    // Relationships
    { subject: "levelgraph", predicate: "uses", object: "go" },
    { subject: "levelgraph", predicate: "uses", object: "leveldb" },
    { subject: "levelgraph", predicate: "type", object: "GraphDatabase" }
]);

// Find all programming languages
log("Programming languages:");
search([
    { subject: "?lang", predicate: "category", object: "programming" }
]);

// Find what levelgraph uses
log("\\nLevelGraph dependencies:");
search([
    { subject: "levelgraph", predicate: "uses", object: "?dep" }
]);

// Find all items and their categories
log("\\nAll categorized items:");
search([
    { subject: "?item", predicate: "category", object: "?cat" }
]);

// Multi-hop: What categories do levelgraph's deps belong to?
log("\\nCategories of levelgraph dependencies:");
search([
    { subject: "levelgraph", predicate: "uses", object: "?dep" },
    { subject: "?dep", predicate: "category", object: "?category" }
]);
`,

    navigation: `// Graph Navigation API

// Create a hierarchical graph
put([
    // Organization structure
    { subject: "company", predicate: "has_dept", object: "engineering" },
    { subject: "company", predicate: "has_dept", object: "marketing" },
    { subject: "engineering", predicate: "has_team", object: "backend" },
    { subject: "engineering", predicate: "has_team", object: "frontend" },
    { subject: "marketing", predicate: "has_team", object: "content" },
    
    // People in teams
    { subject: "backend", predicate: "has_member", object: "alice" },
    { subject: "backend", predicate: "has_member", object: "bob" },
    { subject: "frontend", predicate: "has_member", object: "charlie" },
    { subject: "content", predicate: "has_member", object: "diana" },
    
    // Skills
    { subject: "alice", predicate: "skill", object: "go" },
    { subject: "alice", predicate: "skill", object: "databases" },
    { subject: "bob", predicate: "skill", object: "go" },
    { subject: "charlie", predicate: "skill", object: "javascript" },
    { subject: "diana", predicate: "skill", object: "writing" }
]);

// Navigate: company -> departments
log("Company departments:");
nav({ start: "company", steps: [
    { type: "out", predicate: "has_dept" }
]});

// Navigate: company -> engineering -> teams
log("\\nEngineering teams:");
nav({ start: "company", steps: [
    { type: "out", predicate: "has_dept" }
]});
nav({ start: "engineering", steps: [
    { type: "out", predicate: "has_team" }
]});

// Navigate: company -> dept -> team -> members
log("\\nAll employees (via navigation):");
nav({ start: "company", steps: [
    { type: "out", predicate: "has_dept" },
    { type: "out", predicate: "has_team" },
    { type: "out", predicate: "has_member" }
]});

// Find alice's skills
log("\\nAlice's skills:");
nav({ start: "alice", steps: [
    { type: "out", predicate: "skill" }
]});

// Reverse navigation: who works on backend?
log("\\nBackend team members (reverse nav):");
nav({ start: "backend", steps: [
    { type: "out", predicate: "has_member" }
]});
`
};

// Initialize WASM
async function initWasm(buildType = 'tinygo') {
    currentBuild = buildType;
    wasmReady = false;
    setStatus(false, 'Loading WASM...');
    
    try {
        const build = wasmBuilds[buildType];
        const go = new Go();
        const result = await WebAssembly.instantiateStreaming(
            fetch(build.wasm),
            go.importObject
        );
        go.run(result.instance);
        
        // Wait for the ready event or check isReady
        const checkReady = () => {
            if (window.levelgraph && window.levelgraph.isReady()) {
                setStatus(true);
                wasmReady = true;
            } else {
                setTimeout(checkReady, 50);
            }
        };
        checkReady();
    } catch (err) {
        console.error("Failed to load WASM:", err);
        setStatus(false, "Failed to load: " + err.message);
    }
}

// Switch WASM build (requires page reload to load different wasm_exec.js)
function switchWasmBuild() {
    const select = document.getElementById('wasmBuild');
    const newBuild = select.value;
    if (newBuild !== currentBuild) {
        // Store preference and reload to load correct wasm_exec.js
        localStorage.setItem('levelgraph-wasm-build', newBuild);
        window.location.reload();
    }
}

// Set status indicator
function setStatus(ready, message) {
    const dot = document.getElementById("statusDot");
    const text = document.getElementById("statusText");
    const btn = document.getElementById("runBtn");
    
    if (ready) {
        dot.className = "status-dot ready";
        text.textContent = "Ready";
        btn.disabled = false;
    } else {
        dot.className = "status-dot";
        text.textContent = message || "Not Ready";
        btn.disabled = true;
    }
}

// Output functions
function appendOutput(content, type = "result") {
    const output = document.getElementById("output");
    const entry = document.createElement("div");
    entry.className = "output-entry";
    
    if (typeof content === "object") {
        entry.innerHTML = formatObject(content, type);
    } else {
        entry.innerHTML = `<div class="output-${type}">${escapeHtml(String(content))}</div>`;
    }
    
    output.appendChild(entry);
    output.scrollTop = output.scrollHeight;
}

function formatObject(obj, type) {
    if (obj.triples && Array.isArray(obj.triples)) {
        if (obj.triples.length === 0) {
            return '<div class="output-info">(no results)</div>';
        }
        return formatTriples(obj.triples);
    }
    if (obj.solutions && Array.isArray(obj.solutions)) {
        if (obj.solutions.length === 0) {
            return '<div class="output-info">(no solutions)</div>';
        }
        return formatSolutions(obj.solutions);
    }
    if (obj.values && Array.isArray(obj.values)) {
        if (obj.values.length === 0) {
            return '<div class="output-info">(no values)</div>';
        }
        return formatValues(obj.values);
    }
    if (obj.error) {
        return `<div class="output-error">Error: ${escapeHtml(obj.error)}</div>`;
    }
    if (obj.count !== undefined) {
        return `<div class="output-info">${obj.count} triple(s) affected</div>`;
    }
    return `<pre class="output-${type}">${escapeHtml(JSON.stringify(obj, null, 2))}</pre>`;
}

function formatTriples(triples) {
    let html = '<div class="triple-list">';
    for (const t of triples) {
        html += `<div class="triple">
            <span class="triple-part triple-subject">${escapeHtml(t.subject)}</span>
            <span class="triple-arrow">→</span>
            <span class="triple-part triple-predicate">${escapeHtml(t.predicate)}</span>
            <span class="triple-arrow">→</span>
            <span class="triple-part triple-object">${escapeHtml(t.object)}</span>
        </div>`;
    }
    html += '</div>';
    return html;
}

function formatSolutions(solutions) {
    let html = '<div class="triple-list">';
    for (const sol of solutions) {
        const pairs = Object.entries(sol).map(([k, v]) => 
            `<span class="triple-part triple-subject">?${escapeHtml(k)}</span>=<span class="triple-part triple-object">${escapeHtml(v)}</span>`
        ).join(' ');
        html += `<div class="triple">${pairs}</div>`;
    }
    html += '</div>';
    return html;
}

function formatValues(values) {
    let html = '<div class="triple-list">';
    for (const v of values) {
        html += `<div class="triple"><span class="triple-part triple-object">${escapeHtml(v)}</span></div>`;
    }
    html += '</div>';
    return html;
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function clearOutput() {
    document.getElementById("output").innerHTML = "";
}

// Database wrapper functions
function put(triples) {
    const result = window.levelgraph.put(JSON.stringify(triples));
    appendOutput(result, result.error ? "error" : "info");
    
    // Update graph visualization
    if (!result.error) {
        addTriplesToGraph(triples);
        updateGraph();
    }
    
    return result;
}

function del(triples) {
    const result = window.levelgraph.del(JSON.stringify(triples));
    appendOutput(result, result.error ? "error" : "info");
    
    // Update graph visualization
    if (!result.error) {
        removeTriplesFromGraph(triples);
        updateGraph();
    }
    
    return result;
}

function get(pattern) {
    const result = window.levelgraph.get(JSON.stringify(pattern));
    appendOutput(result, result.error ? "error" : "result");
    
    // Highlight matching results in graph
    if (!result.error) {
        highlightQueryResults(result);
    }
    
    return result;
}

function search(patterns, options) {
    const result = options 
        ? window.levelgraph.search(JSON.stringify(patterns), JSON.stringify(options))
        : window.levelgraph.search(JSON.stringify(patterns));
    appendOutput(result, result.error ? "error" : "result");
    
    // Highlight matching results in graph
    if (!result.error) {
        highlightQueryResults(result);
    }
    
    return result;
}

function nav(navConfig) {
    const result = window.levelgraph.nav(JSON.stringify(navConfig));
    appendOutput(result, result.error ? "error" : "result");
    
    // Highlight matching results in graph
    if (!result.error) {
        highlightQueryResults(result);
    }
    
    return result;
}

function log(message) {
    appendOutput(message, "info");
}

function resetDB() {
    window.levelgraph.reset();
    clearOutput();
    clearGraph();
    appendOutput("Database reset.", "info");
}

// Run code from editor
function runCode() {
    if (!wasmReady) {
        appendOutput("WASM not ready yet", "error");
        return;
    }
    
    const code = document.getElementById("editor").value;
    
    try {
        // Reset DB and graph before running
        window.levelgraph.reset();
        clearOutput();
        clearGraph();
        
        // Execute the code
        eval(code);
    } catch (err) {
        appendOutput("JavaScript Error: " + err.message, "error");
        console.error(err);
    }
}

// Handle example selection
document.getElementById("examples").addEventListener("change", function(e) {
    if (e.target.value && examples[e.target.value]) {
        document.getElementById("editor").value = examples[e.target.value];
        e.target.value = "";
    }
});

// Keyboard shortcut: Ctrl/Cmd + Enter to run
document.getElementById("editor").addEventListener("keydown", function(e) {
    if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
        e.preventDefault();
        runCode();
    }
});

// Initialize
document.addEventListener("DOMContentLoaded", function() {
    // Get saved preference or default to tinygo
    const savedBuild = localStorage.getItem('levelgraph-wasm-build') || 'tinygo';
    currentBuild = savedBuild;
    
    // Update dropdown to match
    const select = document.getElementById('wasmBuild');
    if (select) {
        select.value = savedBuild;
    }
    
    // Initialize graph visualization
    initGraph();
    
    // Initialize with the selected build
    initWasm(savedBuild);
});

// ============================================
// Graph Visualization (D3.js Force-Directed)
// ============================================

// Color scheme matching triple visualization
const nodeColors = {
    subject: { fill: 'rgba(78, 204, 163, 0.8)', stroke: '#4ecca3' },
    object: { fill: 'rgba(100, 149, 237, 0.8)', stroke: '#6495ED' },
    both: { fill: 'rgba(186, 156, 200, 0.8)', stroke: '#ba9cc8' } // appears as both subject and object
};

const linkColor = '#e94560';

// Initialize the graph SVG and simulation
function initGraph() {
    const container = document.getElementById('graph-container');
    if (!container) return;
    
    // Clear container
    container.innerHTML = '';
    
    const width = container.clientWidth || 400;
    const height = container.clientHeight || 400;
    
    // Create SVG
    graphSvg = d3.select(container)
        .append('svg')
        .attr('width', '100%')
        .attr('height', '100%')
        .attr('viewBox', [0, 0, width, height]);
    
    // Add arrow marker for directed edges
    graphSvg.append('defs').append('marker')
        .attr('id', 'arrowhead')
        .attr('viewBox', '-0 -5 10 10')
        .attr('refX', 20)
        .attr('refY', 0)
        .attr('orient', 'auto')
        .attr('markerWidth', 6)
        .attr('markerHeight', 6)
        .append('path')
        .attr('d', 'M 0,-5 L 10,0 L 0,5')
        .attr('fill', linkColor);
    
    // Add highlighted arrow marker
    graphSvg.select('defs').append('marker')
        .attr('id', 'arrowhead-highlighted')
        .attr('viewBox', '-0 -5 10 10')
        .attr('refX', 20)
        .attr('refY', 0)
        .attr('orient', 'auto')
        .attr('markerWidth', 6)
        .attr('markerHeight', 6)
        .append('path')
        .attr('d', 'M 0,-5 L 10,0 L 0,5')
        .attr('fill', '#ff6b6b');
    
    // Create container group for zoom
    const g = graphSvg.append('g').attr('class', 'graph-content');
    
    // Add zoom behavior
    graphZoom = d3.zoom()
        .scaleExtent([0.1, 4])
        .on('zoom', (event) => {
            g.attr('transform', event.transform);
        });
    
    graphSvg.call(graphZoom);
    
    // Create groups for links and nodes
    g.append('g').attr('class', 'links');
    g.append('g').attr('class', 'link-labels');
    g.append('g').attr('class', 'nodes');
    
    // Initialize force simulation
    graphSimulation = d3.forceSimulation()
        .force('link', d3.forceLink().id(d => d.id).distance(100))
        .force('charge', d3.forceManyBody().strength(-300))
        .force('center', d3.forceCenter(width / 2, height / 2))
        .force('collision', d3.forceCollide().radius(30));
    
    graphSimulation.on('tick', updateGraphPositions);
}

// Update the graph with new data
function updateGraph() {
    if (!graphSvg) {
        initGraph();
    }
    
    const g = graphSvg.select('.graph-content');
    const container = document.getElementById('graph-container');
    const width = container.clientWidth || 400;
    const height = container.clientHeight || 400;
    
    // Update links
    const linkGroup = g.select('.links');
    const links = linkGroup.selectAll('.graph-link')
        .data(graphData.links, d => `${d.source.id || d.source}-${d.predicate}-${d.target.id || d.target}`);
    
    links.exit().remove();
    
    const linksEnter = links.enter()
        .append('path')
        .attr('class', 'graph-link')
        .attr('stroke', linkColor)
        .attr('marker-end', 'url(#arrowhead)');
    
    // Update link labels
    const labelGroup = g.select('.link-labels');
    const labels = labelGroup.selectAll('.graph-link-label')
        .data(graphData.links, d => `${d.source.id || d.source}-${d.predicate}-${d.target.id || d.target}`);
    
    labels.exit().remove();
    
    labels.enter()
        .append('text')
        .attr('class', 'graph-link-label')
        .attr('text-anchor', 'middle')
        .text(d => d.predicate)
        .style('display', showEdgeLabels ? 'block' : 'none');
    
    // Update nodes
    const nodeGroup = g.select('.nodes');
    const nodes = nodeGroup.selectAll('.graph-node')
        .data(graphData.nodes, d => d.id);
    
    nodes.exit().remove();
    
    const nodesEnter = nodes.enter()
        .append('g')
        .attr('class', 'graph-node')
        .call(d3.drag()
            .on('start', dragStarted)
            .on('drag', dragged)
            .on('end', dragEnded));
    
    nodesEnter.append('circle')
        .attr('r', 15)
        .attr('fill', d => getNodeColor(d).fill)
        .attr('stroke', d => getNodeColor(d).stroke);
    
    nodesEnter.append('text')
        .attr('dy', 25)
        .text(d => truncateLabel(d.id, 12));
    
    // Add tooltip on hover
    nodesEnter.append('title')
        .text(d => d.id);
    
    // Update simulation
    graphSimulation.nodes(graphData.nodes);
    graphSimulation.force('link').links(graphData.links);
    graphSimulation.force('center', d3.forceCenter(width / 2, height / 2));
    graphSimulation.alpha(1).restart();
    
    // Apply highlighting
    applyHighlighting();
}

// Get node color based on whether it's a subject, object, or both
function getNodeColor(node) {
    if (node.isSubject && node.isObject) {
        return nodeColors.both;
    } else if (node.isSubject) {
        return nodeColors.subject;
    } else {
        return nodeColors.object;
    }
}

// Truncate long labels
function truncateLabel(text, maxLen) {
    if (text.length <= maxLen) return text;
    return text.substring(0, maxLen - 1) + '…';
}

// Update positions on simulation tick
function updateGraphPositions() {
    if (!graphSvg) return;
    
    const g = graphSvg.select('.graph-content');
    
    // Update link paths (curved for multiple edges between same nodes)
    g.select('.links').selectAll('.graph-link')
        .attr('d', d => {
            const dx = d.target.x - d.source.x;
            const dy = d.target.y - d.source.y;
            const dr = Math.sqrt(dx * dx + dy * dy) * 1.5;
            return `M${d.source.x},${d.source.y}A${dr},${dr} 0 0,1 ${d.target.x},${d.target.y}`;
        });
    
    // Update link label positions
    g.select('.link-labels').selectAll('.graph-link-label')
        .attr('x', d => (d.source.x + d.target.x) / 2)
        .attr('y', d => (d.source.y + d.target.y) / 2 - 5);
    
    // Update node positions
    g.select('.nodes').selectAll('.graph-node')
        .attr('transform', d => `translate(${d.x},${d.y})`);
}

// Drag handlers
function dragStarted(event, d) {
    if (!event.active) graphSimulation.alphaTarget(0.3).restart();
    d.fx = d.x;
    d.fy = d.y;
}

function dragged(event, d) {
    d.fx = event.x;
    d.fy = event.y;
}

function dragEnded(event, d) {
    if (!event.active) graphSimulation.alphaTarget(0);
    d.fx = null;
    d.fy = null;
}

// Add triples to the graph data structure
function addTriplesToGraph(triples) {
    const nodeMap = new Map();
    
    // Preserve existing nodes
    for (const node of graphData.nodes) {
        nodeMap.set(node.id, node);
    }
    
    // Process new triples
    for (const triple of triples) {
        const { subject, predicate, object } = triple;
        
        // Add/update subject node
        if (!nodeMap.has(subject)) {
            nodeMap.set(subject, { id: subject, isSubject: true, isObject: false });
        } else {
            nodeMap.get(subject).isSubject = true;
        }
        
        // Add/update object node
        if (!nodeMap.has(object)) {
            nodeMap.set(object, { id: object, isSubject: false, isObject: true });
        } else {
            nodeMap.get(object).isObject = true;
        }
        
        // Add link if not already present
        const linkExists = graphData.links.some(
            l => (l.source.id || l.source) === subject && 
                 (l.target.id || l.target) === object && 
                 l.predicate === predicate
        );
        
        if (!linkExists) {
            graphData.links.push({
                source: subject,
                target: object,
                predicate: predicate
            });
        }
    }
    
    graphData.nodes = Array.from(nodeMap.values());
}

// Remove triples from graph
function removeTriplesFromGraph(triples) {
    for (const triple of triples) {
        const { subject, predicate, object } = triple;
        
        // Remove matching links
        graphData.links = graphData.links.filter(
            l => !((l.source.id || l.source) === subject && 
                   (l.target.id || l.target) === object && 
                   l.predicate === predicate)
        );
    }
    
    // Remove orphan nodes (nodes with no links)
    const connectedNodes = new Set();
    for (const link of graphData.links) {
        connectedNodes.add(link.source.id || link.source);
        connectedNodes.add(link.target.id || link.target);
    }
    
    graphData.nodes = graphData.nodes.filter(n => connectedNodes.has(n.id));
}

// Clear the graph
function clearGraph() {
    graphData = { nodes: [], links: [] };
    highlightedNodes.clear();
    highlightedLinks.clear();
    if (graphSvg) {
        updateGraph();
    }
}

// Highlight nodes and links based on query results
function highlightQueryResults(results) {
    highlightedNodes.clear();
    highlightedLinks.clear();
    
    if (results.triples && Array.isArray(results.triples)) {
        for (const t of results.triples) {
            highlightedNodes.add(t.subject);
            highlightedNodes.add(t.object);
            highlightedLinks.add(`${t.subject}-${t.predicate}-${t.object}`);
        }
    }
    
    if (results.solutions && Array.isArray(results.solutions)) {
        for (const sol of results.solutions) {
            for (const value of Object.values(sol)) {
                highlightedNodes.add(value);
            }
        }
    }
    
    if (results.values && Array.isArray(results.values)) {
        for (const v of results.values) {
            highlightedNodes.add(v);
        }
    }
    
    applyHighlighting();
}

// Apply visual highlighting to nodes and links
function applyHighlighting() {
    if (!graphSvg) return;
    
    const g = graphSvg.select('.graph-content');
    
    g.select('.nodes').selectAll('.graph-node')
        .classed('highlighted', d => highlightedNodes.has(d.id));
    
    g.select('.links').selectAll('.graph-link')
        .classed('highlighted', d => {
            const key = `${d.source.id || d.source}-${d.predicate}-${d.target.id || d.target}`;
            return highlightedLinks.has(key);
        })
        .attr('marker-end', d => {
            const key = `${d.source.id || d.source}-${d.predicate}-${d.target.id || d.target}`;
            return highlightedLinks.has(key) ? 'url(#arrowhead-highlighted)' : 'url(#arrowhead)';
        });
}

// Reset graph zoom
function resetGraphZoom() {
    if (graphSvg && graphZoom) {
        graphSvg.transition().duration(500).call(
            graphZoom.transform,
            d3.zoomIdentity
        );
    }
}

// Toggle edge labels
function toggleGraphLabels() {
    showEdgeLabels = !showEdgeLabels;
    if (graphSvg) {
        graphSvg.select('.graph-content')
            .select('.link-labels')
            .selectAll('.graph-link-label')
            .style('display', showEdgeLabels ? 'block' : 'none');
    }
}
