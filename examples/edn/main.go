// Package main demonstrates parsing EDN (Extensible Data Notation) using the
// EBNF parser framework, with transforms that produce typed Go values.
//
// This serves as both an example and a benchmark comparison point against the
// hand-rolled EDN parser in janus-datalog/datalog/edn.
package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/wbrown/ebnf"
	"github.com/wbrown/ebnf/parse"
)

// EDN value types — mirrors the types in janus-datalog/datalog/edn.Node
// but uses native Go types instead of a union struct.

type EDNValue interface{}

type EDNList []EDNValue
type EDNVector []EDNValue
type EDNMap []EDNMapEntry
type EDNSet []EDNValue
type EDNTagged struct {
	Tag   string
	Value EDNValue
}
type EDNKeyword string
type EDNSymbol string

type EDNMapEntry struct {
	Key, Val EDNValue
}

func buildTransforms() parse.TransformMap {
	return parse.TransformMap{
		"nil_lit": func(s string) EDNValue { return nil },
		"boolean": func(s string) EDNValue { return s == "true" },
		"integer": func(s string) EDNValue {
			v, _ := strconv.ParseInt(s, 10, 64)
			return v
		},
		"float": func(parts ...string) EDNValue {
			s := strings.Join(parts, "")
			v, _ := strconv.ParseFloat(s, 64)
			return v
		},
		"string": func(s string) EDNValue {
			// Strip quotes and unescape
			if len(s) >= 2 {
				s = s[1 : len(s)-1]
			}
			s = strings.ReplaceAll(s, `\"`, `"`)
			s = strings.ReplaceAll(s, `\\`, `\`)
			s = strings.ReplaceAll(s, `\n`, "\n")
			s = strings.ReplaceAll(s, `\t`, "\t")
			s = strings.ReplaceAll(s, `\r`, "\r")
			return s
		},
		"character": func(s string) EDNValue {
			// Strip leading backslash
			s = strings.TrimPrefix(s, `\`)
			switch s {
			case "newline":
				return '\n'
			case "space":
				return ' '
			case "tab":
				return '\t'
			case "return":
				return '\r'
			default:
				runes := []rune(s)
				if len(runes) == 1 {
					return runes[0]
				}
				return rune(0)
			}
		},
		"keyword": func(s string) EDNValue { return EDNKeyword(s) },
		"symbol":  func(s string) EDNValue { return EDNSymbol(s) },
		"list": func(items ...EDNValue) EDNValue {
			return EDNList(items)
		},
		"vector": func(items ...EDNValue) EDNValue {
			return EDNVector(items)
		},
		"hash_map": func(items ...EDNValue) EDNValue {
			entries := make(EDNMap, 0, len(items)/2)
			for i := 0; i+1 < len(items); i += 2 {
				entries = append(entries, EDNMapEntry{Key: items[i], Val: items[i+1]})
			}
			return entries
		},
		"set": func(items ...EDNValue) EDNValue {
			return EDNSet(items)
		},
		"tagged": func(tag string, value EDNValue) EDNValue {
			return EDNTagged{Tag: tag, Value: value}
		},
		"tag_name": func(s string) string { return s },
	}
}

func formatValue(v EDNValue, indent int) string {
	prefix := strings.Repeat("  ", indent)
	switch val := v.(type) {
	case nil:
		return prefix + "nil"
	case bool:
		return prefix + fmt.Sprintf("%v", val)
	case int64:
		return prefix + fmt.Sprintf("%d", val)
	case float64:
		return prefix + fmt.Sprintf("%g", val)
	case string:
		return prefix + fmt.Sprintf("%q", val)
	case rune:
		return prefix + fmt.Sprintf("\\%c", val)
	case EDNKeyword:
		return prefix + string(val)
	case EDNSymbol:
		return prefix + string(val)
	case EDNList:
		if len(val) == 0 {
			return prefix + "()"
		}
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = formatValue(item, 0)
		}
		return prefix + "(" + strings.Join(parts, " ") + ")"
	case EDNVector:
		if len(val) == 0 {
			return prefix + "[]"
		}
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = formatValue(item, 0)
		}
		return prefix + "[" + strings.Join(parts, " ") + "]"
	case EDNMap:
		if len(val) == 0 {
			return prefix + "{}"
		}
		parts := make([]string, len(val))
		for i, entry := range val {
			parts[i] = formatValue(entry.Key, 0) + " " + formatValue(entry.Val, 0)
		}
		return prefix + "{" + strings.Join(parts, ", ") + "}"
	case EDNSet:
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = formatValue(item, 0)
		}
		return prefix + "#{" + strings.Join(parts, " ") + "}"
	case EDNTagged:
		return prefix + "#" + val.Tag + " " + formatValue(val.Value, 0)
	default:
		return prefix + fmt.Sprintf("<%T>", v)
	}
}

func main() {
	grammar, err := ebnf.LoadGrammar("examples/edn.ebnf")
	if err != nil {
		// Try relative path for running from examples/edn/
		grammar, err = ebnf.LoadGrammar("../edn.ebnf")
		if err != nil {
			log.Fatal(err)
		}
	}

	p := parse.New(grammar)
	transforms := buildTransforms()

	// Example EDN values — including a Datalog query
	examples := []string{
		// Simple values
		`42`,
		`:keyword`,
		`"hello world"`,
		`true`,
		`nil`,

		// Collections
		`[1 2 3]`,
		`{:name "Alice" :age 30}`,
		`#{:a :b :c}`,

		// Tagged literal
		`#inst "2025-01-01T00:00:00Z"`,

		// Nested structure
		`{:person {:name "Bob" :scores [95 87 92]} :active true}`,

		// Datalog query (the real use case)
		`[:find ?e ?name
		  :where [?e :person/name ?name]
		         [?e :person/age ?age]
		         [(> ?age 21)]]`,

		// Complex query with subqueries and OR
		`[:find ?scenario ?title ?count
		  :where [?scenario :entity/type :entity.type/scenario]
		         [(get-else $ ?scenario :scenario/title "") ?title]
		         (or [(q [:find (count ?t)
		                  :in $ ?s
		                  :where [?t :task/root ?s]
		                         [?t :task/status :status/complete]]
		                 $ ?scenario) [[?count]]]
		             [(ground 0) ?count])]`,
	}

	for _, input := range examples {
		tree, err := p.Parse(input, "edn")
		if err != nil {
			fmt.Printf("PARSE ERROR: %v\n  input: %s\n\n", err, input)
			continue
		}

		result, err := parse.Transform(tree, transforms)
		if err != nil {
			fmt.Printf("TRANSFORM ERROR: %v\n  input: %s\n\n", err, input)
			continue
		}

		// Transform returns the top-level value(s)
		switch v := result.(type) {
		case []interface{}:
			// Multiple top-level values
			for _, item := range v {
				fmt.Println(formatValue(item.(EDNValue), 0))
			}
		case EDNValue:
			fmt.Println(formatValue(v, 0))
		default:
			fmt.Printf("%v\n", result)
		}
		fmt.Println()
	}

	// Quick parse timing
	queryStr := `[:find ?scenario ?title ?createdAt ?taskCount ?totalTokens
	  :where [?scenario :entity/type :entity.type/scenario]
	         [?scenario :scenario/title ?title]
	         [?scenario :scenario/created-at ?createdAt]
	         (or [(q [:find (count ?t) (sum ?tok)
	                  :in $ ?s
	                  :where [?t :task/root ?s]
	                         [?t :task/status :status/complete]
	                         [(get-else $ ?t :task/token-count 0) ?tok]]
	                 $ ?scenario) [[?taskCount ?totalTokens]]]
	             [(ground [0 0]) [[?taskCount ?totalTokens]]])]`

	const iterations = 1000
	start := time.Now()
	for i := 0; i < iterations; i++ {
		tree, _ := p.Parse(queryStr, "edn")
		parse.Transform(tree, transforms)
	}
	elapsed := time.Since(start)
	fmt.Printf("EBNF parse+transform: %d iterations in %s (%.1f µs/op)\n",
		iterations, elapsed, float64(elapsed.Microseconds())/float64(iterations))
}
