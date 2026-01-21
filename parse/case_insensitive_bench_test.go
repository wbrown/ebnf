package parse

import (
	"testing"

	"github.com/wbrown/ebnf"
)

// BenchmarkCaseInsensitiveVsRegex compares performance of case-insensitive
// literal matching using EqualFold vs regex (?i) pattern matching.

func BenchmarkCaseInsensitiveLiteral(b *testing.B) {
	grammar, _ := ebnf.ParseString(`keyword = 'SELECT'i ;`)
	p := New(grammar)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.Parse("select", "keyword")
	}
}

func BenchmarkCaseSensitiveLiteral(b *testing.B) {
	grammar, _ := ebnf.ParseString(`keyword = 'select' ;`)
	p := New(grammar)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.Parse("select", "keyword")
	}
}

func BenchmarkRegexCaseInsensitive(b *testing.B) {
	grammar, _ := ebnf.ParseString(`keyword = #"(?i)select" ;`)
	p := New(grammar)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.Parse("select", "keyword")
	}
}

func BenchmarkCaseInsensitiveSQLQuery(b *testing.B) {
	grammar, _ := ebnf.ParseString(`
		query = 'SELECT'i <' '> column <' '> 'FROM'i <' '> table ;
		column = 'name' ;
		table = 'users' ;
	`)
	p := New(grammar)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.Parse("SELECT name FROM users", "query")
	}
}

func BenchmarkRegexSQLQuery(b *testing.B) {
	grammar, _ := ebnf.ParseString(`
		query = #"(?i)SELECT" <' '> column <' '> #"(?i)FROM" <' '> table ;
		column = 'name' ;
		table = 'users' ;
	`)
	p := New(grammar)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.Parse("SELECT name FROM users", "query")
	}
}

func BenchmarkGlobalCaseInsensitive(b *testing.B) {
	grammar, _ := ebnf.ParseString(`keyword = 'select' ;`)
	p := New(grammar, WithCaseInsensitive(true))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.Parse("SELECT", "keyword")
	}
}
