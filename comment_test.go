package ebnf

import (
	"testing"
)

func TestCommentsInGrammar(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "comment_between_rules",
			input: `
rule1 = 'a' ;
(* This is a comment *)
rule2 = 'b' ;`,
			wantErr: false,
		},
		{
			name: "comment_in_choice",
			input: `rule = [ 
    'a' |  (* option a *)
    'b' |  (* option b *)  
    'c'    (* option c *)
] ;`,
			wantErr: false,
		},
		{
			name: "comment_in_sequence",
			input: `rule = 'a' (* first *) 'b' (* second *) 'c' ;`,
			wantErr: false,
		},
		{
			name: "comment_after_terminal",
			input: `rule = 'test' (* a test *) ;`,
			wantErr: false,
		},
		{
			name: "multiple_comments",
			input: `
(* First comment *)
rule1 = 'a' ;
(* Second comment *)
(* Third comment *)
rule2 = 'b' ;`,
			wantErr: false,
		},
		{
			name: "comment_in_nested_groups",
			input: `rule = [ ( 'a' (* inner comment *) ) | (* outer comment *) 'b' ] ;`,
			wantErr: false,
		},
		{
			name: "comment_after_hidden_terminal",
			input: `rule = <'hidden'> (* after hidden *) 'visible' ;`,
			wantErr: false,
		},
		{
			name: "comment_in_repetition",
			input: `rule = ( 'item' (* each item *) )* ;`,
			wantErr: false,
		},
		{
			name: "comment_in_optional",
			input: `rule = ( 'optional' (* maybe *) )? ;`,
			wantErr: false,
		},
		{
			name: "comment_with_parentheses_in_brackets",
			input: `rule = [ ( 'grouped' ) | (* comment forces choice interpretation *) 'ungrouped' ] ;`,
			wantErr: false,
		},
		{
			name: "comment_between_choice_alternatives",
			input: `rule = 'a' | (* between *) 'b' | (* choices *) 'c' ;`,
			wantErr: false,
		},
		{
			name: "comment_in_char_class_makes_it_choice",
			input: `rule = [ a (* this makes it a choice, not char class *) | b ] ;`,
			wantErr: false,
		},
		{
			name: "multi_rule_grammar_with_comments",
			input: `(* Test grammar with comments in various places *)

rule1 = [ 
    option_a |  (* This is option A *)
    option_b |  (* This is option B *)
    option_c    (* This is option C *)
] ;

rule2 = terminal1 (* after terminal *) terminal2 ;

option_a = 'a' ;
option_b = 'b' ;
option_c = 'c' ;
terminal1 = 'x' ;
terminal2 = 'y' ;`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grammar, err := ParseString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			// Verify we actually parsed rules
			if !tt.wantErr && grammar != nil {
				if len(grammar.Rules) == 0 {
					t.Error("Expected at least one rule but got none")
				}
			}
		})
	}
}

func TestCommentEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "unclosed_comment",
			input: `rule = 'a' (* unclosed comment ;`,
			wantErr: true,
		},
		{
			name: "nested_comments_not_supported",
			input: `rule = 'a' (* outer (* inner *) comment *) ;`,
			wantErr: true, // Standard Pascal comments don't support nesting
		},
		{
			name: "comment_at_eof",
			input: `rule = 'a' ; (* final comment`,
			wantErr: true, // Unclosed comment at EOF
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}