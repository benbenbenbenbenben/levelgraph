// LevelGraph Playground JavaScript

// State
let wasmReady = false;
let currentBuild = 'tinygo'; // default to smaller TinyGo build

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
    return result;
}

function del(triples) {
    const result = window.levelgraph.del(JSON.stringify(triples));
    appendOutput(result, result.error ? "error" : "info");
    return result;
}

function get(pattern) {
    const result = window.levelgraph.get(JSON.stringify(pattern));
    appendOutput(result, result.error ? "error" : "result");
    return result;
}

function search(patterns, options) {
    const result = options 
        ? window.levelgraph.search(JSON.stringify(patterns), JSON.stringify(options))
        : window.levelgraph.search(JSON.stringify(patterns));
    appendOutput(result, result.error ? "error" : "result");
    return result;
}

function nav(navConfig) {
    const result = window.levelgraph.nav(JSON.stringify(navConfig));
    appendOutput(result, result.error ? "error" : "result");
    return result;
}

function log(message) {
    appendOutput(message, "info");
}

function resetDB() {
    window.levelgraph.reset();
    clearOutput();
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
        // Reset DB before running
        window.levelgraph.reset();
        clearOutput();
        
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
    
    // Initialize with the selected build
    initWasm(savedBuild);
});
