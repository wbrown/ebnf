package parse

import (
	"os"
	"testing"

	"github.com/wbrown/ebnf"
)

// Helper to check if we can access choicescript
func hasChoiceScriptAccess() bool {
	_, err := os.Stat("../../choicescript/resources/choicescript_lexer.ebnf")
	return err == nil
}

// Benchmark data - JSON grammar for baseline (built-in, no external deps)
var jsonGrammar = `
json = <ws>* value <ws>* ;
<value> = object | array | string | number | boolean | null ;
boolean = true | false ;
true = <"true"> ;
false = <"false"> ;
null = <"null"> ;
object = <"{"> <ws>* <"}"> | <"{"> <ws>* members <ws>* <"}"> ;
members = pair ( <ws>* <","> <ws>* pair )* ;
pair = string <ws>* <":"> <ws>* value ;
array = <"["> <ws>* <"]"> | <"["> <ws>* elements <ws>* <"]"> ;
elements = value ( <ws>* <","> <ws>* value )* ;
string = <'"'> <string_content> <'"'> ;
<string_content> = #"([^\"\\]|\\[\"\\\/bfnrt]|\\u[0-9a-fA-F]{4})*" ;
number = int frac? exp? ;
int = "-"? ( "0" | [1-9] digit* ) ;
<digit> = [0-9] ;
frac = <"."> digit+ ;
exp = ( <"e"> | <"E"> ) ( <"+"> | <"-"> )? digit+ ;
ws = " " | "\t" | "\n" | "\r" ;
`

var benchGrammar *ebnf.Grammar
var benchSmallJSON string
var benchMediumJSON string
var benchLargeJSON string

func init() {
	var err error
	benchGrammar, err = ebnf.ParseString(jsonGrammar)
	if err != nil {
		panic(err)
	}

	// Small JSON (~100 bytes)
	benchSmallJSON = `{"name":"John","age":30,"city":"New York"}`

	// Medium JSON (~500 bytes)
	benchMediumJSON = `{
		"users": [
			{"id":1,"name":"Alice","email":"alice@example.com","active":true},
			{"id":2,"name":"Bob","email":"bob@example.com","active":false},
			{"id":3,"name":"Charlie","email":"charlie@example.com","active":true}
		],
		"metadata": {"total":3,"page":1,"limit":10}
	}`

	// Large JSON (~2KB)
	benchLargeJSON = `{
		"company": "TechCorp",
		"employees": [
			{"id":1,"name":"Alice Johnson","role":"Engineer","salary":95000,"projects":["ProjectA","ProjectB"]},
			{"id":2,"name":"Bob Smith","role":"Manager","salary":110000,"projects":["ProjectC"]},
			{"id":3,"name":"Charlie Brown","role":"Designer","salary":85000,"projects":["ProjectA","ProjectD"]},
			{"id":4,"name":"Diana Prince","role":"Engineer","salary":98000,"projects":["ProjectB","ProjectC"]},
			{"id":5,"name":"Eve Adams","role":"Engineer","salary":92000,"projects":["ProjectD"]},
			{"id":6,"name":"Frank Castle","role":"Manager","salary":115000,"projects":["ProjectE"]},
			{"id":7,"name":"Grace Hopper","role":"Engineer","salary":105000,"projects":["ProjectA","ProjectE"]},
			{"id":8,"name":"Henry Ford","role":"Designer","salary":88000,"projects":["ProjectC","ProjectD"]}
		],
		"departments": {
			"engineering": {"budget":500000,"headcount":5},
			"design": {"budget":200000,"headcount":2},
			"management": {"budget":300000,"headcount":2}
		}
	}`
}

// BenchmarkParseSmallJSON benchmarks parsing small JSON (~100 bytes)
func BenchmarkParseSmallJSON(b *testing.B) {
	b.SetBytes(int64(len(benchSmallJSON)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := New(benchGrammar)
		_, err := p.Parse(benchSmallJSON, "value")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseMediumJSON benchmarks parsing medium JSON (~500 bytes)
func BenchmarkParseMediumJSON(b *testing.B) {
	b.SetBytes(int64(len(benchMediumJSON)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := New(benchGrammar)
		_, err := p.Parse(benchMediumJSON, "value")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseLargeJSON benchmarks parsing large JSON (~2KB)
func BenchmarkParseLargeJSON(b *testing.B) {
	b.SetBytes(int64(len(benchLargeJSON)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := New(benchGrammar)
		_, err := p.Parse(benchLargeJSON, "value")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseSmallJSONAllocs measures allocations for small JSON
func BenchmarkParseSmallJSONAllocs(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := New(benchGrammar)
		_, err := p.Parse(benchSmallJSON, "value")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseMediumJSONAllocs measures allocations for medium JSON
func BenchmarkParseMediumJSONAllocs(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := New(benchGrammar)
		_, err := p.Parse(benchMediumJSON, "value")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseLargeJSONAllocs measures allocations for large JSON
func BenchmarkParseLargeJSONAllocs(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := New(benchGrammar)
		_, err := p.Parse(benchLargeJSON, "value")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Simple arithmetic grammar for baseline comparison
var simpleGrammar = `
expr = add_expr ;
add_expr = mult_expr ( ( <"+"> | <"-"> ) mult_expr )* ;
mult_expr = primary ( ( <"*"> | <"/"> ) primary )* ;
primary = number | <"("> expr <")"> ;
number = #"[0-9]+" ;
`

// BenchmarkParseSimpleExpr benchmarks a simple expression parser
func BenchmarkParseSimpleExpr(b *testing.B) {
	g, err := ebnf.ParseString(simpleGrammar)
	if err != nil {
		b.Fatal(err)
	}

	input := "1+2*3+(4-5)*6/7"
	b.SetBytes(int64(len(input)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := New(g)
		_, err := p.Parse(input, "expr")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseSimpleExprAllocs measures allocations for simple expression
func BenchmarkParseSimpleExprAllocs(b *testing.B) {
	g, err := ebnf.ParseString(simpleGrammar)
	if err != nil {
		b.Fatal(err)
	}

	input := "1+2*3+(4-5)*6/7"
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := New(g)
		_, err := p.Parse(input, "expr")
		if err != nil {
			b.Fatal(err)
		}
	}
}

