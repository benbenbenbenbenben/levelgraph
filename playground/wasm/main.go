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

//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"syscall/js"

	"github.com/levelgraph/levelgraph"
)

var db *levelgraph.DB

func main() {
	// Create the in-memory database
	store := levelgraph.NewMemStore()
	db = levelgraph.OpenWithStore(store)

	// Register functions for JavaScript
	js.Global().Set("levelgraph", js.ValueOf(map[string]interface{}{
		"put":     js.FuncOf(put),
		"del":     js.FuncOf(del),
		"get":     js.FuncOf(get),
		"search":  js.FuncOf(search),
		"nav":     js.FuncOf(nav),
		"reset":   js.FuncOf(reset),
		"isReady": js.FuncOf(isReady),
	}))

	// Signal that WASM is ready
	js.Global().Call("dispatchEvent", js.Global().Get("CustomEvent").New("levelgraph-ready"))

	// Keep the Go runtime alive
	select {}
}

// isReady returns true if the database is ready.
func isReady(this js.Value, args []js.Value) interface{} {
	return db != nil && db.IsOpen()
}

// reset clears the database and creates a fresh one.
func reset(this js.Value, args []js.Value) interface{} {
	if db != nil {
		db.Close()
	}
	store := levelgraph.NewMemStore()
	db = levelgraph.OpenWithStore(store)
	return nil
}

// put inserts triples into the database.
// Args: triplesJSON (array of {subject, predicate, object})
// Returns: {error?: string}
func put(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{"error": "put requires a triples argument"}
	}

	triplesJSON := args[0].String()
	var triplesData []struct {
		Subject   string `json:"subject"`
		Predicate string `json:"predicate"`
		Object    string `json:"object"`
	}

	if err := json.Unmarshal([]byte(triplesJSON), &triplesData); err != nil {
		return map[string]interface{}{"error": "invalid JSON: " + err.Error()}
	}

	triples := make([]*levelgraph.Triple, len(triplesData))
	for i, t := range triplesData {
		triples[i] = levelgraph.NewTripleFromStrings(t.Subject, t.Predicate, t.Object)
	}

	ctx := context.Background()
	if err := db.Put(ctx, triples...); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{"count": len(triples)}
}

// del deletes triples from the database.
// Args: triplesJSON (array of {subject, predicate, object})
// Returns: {error?: string}
func del(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{"error": "del requires a triples argument"}
	}

	triplesJSON := args[0].String()
	var triplesData []struct {
		Subject   string `json:"subject"`
		Predicate string `json:"predicate"`
		Object    string `json:"object"`
	}

	if err := json.Unmarshal([]byte(triplesJSON), &triplesData); err != nil {
		return map[string]interface{}{"error": "invalid JSON: " + err.Error()}
	}

	triples := make([]*levelgraph.Triple, len(triplesData))
	for i, t := range triplesData {
		triples[i] = levelgraph.NewTripleFromStrings(t.Subject, t.Predicate, t.Object)
	}

	ctx := context.Background()
	if err := db.Del(ctx, triples...); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{"count": len(triples)}
}

// get retrieves triples matching a pattern.
// Args: patternJSON ({subject?, predicate?, object?, limit?, offset?})
// Returns: {triples: [{subject, predicate, object}], error?: string}
func get(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{"error": "get requires a pattern argument"}
	}

	patternJSON := args[0].String()
	var patternData struct {
		Subject   string `json:"subject,omitempty"`
		Predicate string `json:"predicate,omitempty"`
		Object    string `json:"object,omitempty"`
		Limit     int    `json:"limit,omitempty"`
		Offset    int    `json:"offset,omitempty"`
	}

	if err := json.Unmarshal([]byte(patternJSON), &patternData); err != nil {
		return map[string]interface{}{"error": "invalid JSON: " + err.Error()}
	}

	pattern := &levelgraph.Pattern{
		Limit:  patternData.Limit,
		Offset: patternData.Offset,
	}

	if patternData.Subject != "" {
		pattern.Subject = []byte(patternData.Subject)
	}
	if patternData.Predicate != "" {
		pattern.Predicate = []byte(patternData.Predicate)
	}
	if patternData.Object != "" {
		pattern.Object = []byte(patternData.Object)
	}

	ctx := context.Background()
	triples, err := db.Get(ctx, pattern)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	results := make([]interface{}, len(triples))
	for i, t := range triples {
		results[i] = map[string]interface{}{
			"subject":   string(t.Subject),
			"predicate": string(t.Predicate),
			"object":    string(t.Object),
		}
	}

	return map[string]interface{}{"triples": results}
}

// search executes a search query with multiple patterns (join).
// Args: patternsJSON (array of patterns), optionsJSON (optional)
// Returns: {solutions: [{varName: value}], error?: string}
func search(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{"error": "search requires a patterns argument"}
	}

	patternsJSON := args[0].String()
	var patternsData []struct {
		Subject   interface{} `json:"subject,omitempty"`
		Predicate interface{} `json:"predicate,omitempty"`
		Object    interface{} `json:"object,omitempty"`
	}

	if err := json.Unmarshal([]byte(patternsJSON), &patternsData); err != nil {
		return map[string]interface{}{"error": "invalid JSON: " + err.Error()}
	}

	patterns := make([]*levelgraph.Pattern, len(patternsData))
	for i, p := range patternsData {
		pattern := &levelgraph.Pattern{}
		pattern.Subject = parsePatternField(p.Subject)
		pattern.Predicate = parsePatternField(p.Predicate)
		pattern.Object = parsePatternField(p.Object)
		patterns[i] = pattern
	}

	var opts *levelgraph.SearchOptions
	var filterNotEqual []struct {
		Var   string `json:"var"`   // Variable name (without ?)
		Value string `json:"value"` // Constant value to compare against
		Var2  string `json:"var2"`  // Or another variable name (without ?)
	}
	if len(args) > 1 {
		optsJSON := args[1].String()
		var optsData struct {
			Limit    int `json:"limit,omitempty"`
			Offset   int `json:"offset,omitempty"`
			NotEqual []struct {
				Var   string `json:"var"`
				Value string `json:"value"`
				Var2  string `json:"var2"`
			} `json:"notEqual,omitempty"`
		}
		if err := json.Unmarshal([]byte(optsJSON), &optsData); err == nil {
			opts = &levelgraph.SearchOptions{
				Limit:  optsData.Limit,
				Offset: optsData.Offset,
			}
			filterNotEqual = optsData.NotEqual
		}
	}

	// Add filter for notEqual constraints
	if len(filterNotEqual) > 0 && opts == nil {
		opts = &levelgraph.SearchOptions{}
	}
	if len(filterNotEqual) > 0 {
		opts.Filter = func(sol levelgraph.Solution) bool {
			for _, ne := range filterNotEqual {
				varVal, ok := sol[ne.Var]
				if !ok {
					continue
				}
				// Compare to constant value
				if ne.Value != "" {
					if string(varVal) == ne.Value {
						return false
					}
				}
				// Compare to another variable
				if ne.Var2 != "" {
					if var2Val, ok2 := sol[ne.Var2]; ok2 {
						if string(varVal) == string(var2Val) {
							return false
						}
					}
				}
			}
			return true
		}
	}

	ctx := context.Background()
	solutions, err := db.Search(ctx, patterns, opts)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	results := make([]interface{}, len(solutions))
	for i, sol := range solutions {
		solMap := make(map[string]interface{})
		for k, v := range sol {
			solMap[k] = string(v)
		}
		results[i] = solMap
	}

	return map[string]interface{}{"solutions": results}
}

// parsePatternField parses a pattern field value.
// If it's a string starting with "?", it's a variable.
// Otherwise, it's a concrete value.
func parsePatternField(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	str, ok := v.(string)
	if !ok {
		return nil
	}
	if len(str) > 1 && str[0] == '?' {
		return levelgraph.V(str[1:])
	}
	return []byte(str)
}

// nav executes a navigation query.
// Args: navJSON ({start, steps: [{type: "out"|"in", predicate}]})
// Returns: {values: [string], error?: string}
func nav(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{"error": "nav requires a navigation argument"}
	}

	navJSON := args[0].String()
	var navData struct {
		Start string `json:"start"`
		Steps []struct {
			Type      string `json:"type"`      // "out" or "in"
			Predicate string `json:"predicate"` // the edge predicate
		} `json:"steps"`
	}

	if err := json.Unmarshal([]byte(navJSON), &navData); err != nil {
		return map[string]interface{}{"error": "invalid JSON: " + err.Error()}
	}

	ctx := context.Background()
	navigator := db.Nav(ctx, navData.Start)

	for _, step := range navData.Steps {
		switch step.Type {
		case "out":
			navigator = navigator.ArchOut(step.Predicate)
		case "in":
			navigator = navigator.ArchIn(step.Predicate)
		default:
			return map[string]interface{}{"error": "unknown step type: " + step.Type}
		}
	}

	values, err := navigator.Values()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	results := make([]interface{}, len(values))
	for i, v := range values {
		results[i] = string(v)
	}

	return map[string]interface{}{"values": results}
}
