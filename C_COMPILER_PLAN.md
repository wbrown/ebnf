# Project Plan: C Compiler on the EBNF Transform Toolkit

## For AnotherClaude (or a team of them)

### What You're Building

A C11 compiler targeting x86-64 Linux, built entirely on the `github.com/wbrown/ebnf` parser and transform toolkit. Every compiler pass -- from preprocessing through code emission -- is a `TransformMap`. The architecture is not a design decision. It's a given.

### What Already Exists

**The EBNF toolkit** (~4,800 LOC) provides:
- `ebnf.LoadGrammar()` -- parse `.ebnf` files into grammar definitions
- `parse.New(grammar)` -- create a parser from a grammar
- `parser.Parse(input, startRule)` -- parse input into a `*ParseTree`
- `parse.Transform(tree, transforms)` -- apply a `TransformMap` bottom-up
- `parse.TransformMap` -- map of grammar rule names → transform functions
- `parse.TransformContext` -- parent, siblings, tree, shared `State` map
- Multi-pass, bottom-up, top-down, context-aware, node-aware transforms
- Source position tracking (line, column, start, end) on every node

**The ChoiceScript project** proves this works for a real language:
- ~400 line EBNF grammar handling indentation-sensitivity, 9-level operator precedence, 49 commands
- Separate `TransformMap` files merged in `init()` (linker pattern)
- `transformBinarySequence()` factory handles all precedence levels identically
- Complete interpreter: grammar → parse → transform → S-expression AST → polymorphic execution
- Production-tested on commercial game files

You are not inventing architecture. You are filling in `TransformMap` entries with C-specific domain knowledge.

---

## Project Structure

```
cc/
├── c.ebnf                      # Unified C + preprocessor grammar
├── ast/
│   ├── types.go                # AST node type definitions
│   ├── expr.go                 # Expression node types
│   ├── stmt.go                 # Statement node types
│   ├── decl.go                 # Declaration node types
│   └── print.go                # S-expression printer (debug/testing)
├── transforms/
│   ├── merged.go               # MergedTransforms + init()
│   ├── expressions.go          # Expression transforms (precedence, operators)
│   ├── statements.go           # Statement transforms (if, while, for, switch, etc.)
│   ├── declarations.go         # Declaration/type transforms
│   ├── preprocessor.go         # #define, #include, #ifdef transforms
│   ├── symbols.go              # Scope/symbol table pass
│   ├── typecheck.go            # Type checking/coercion pass
│   └── emit.go                 # AST → x86-64 assembly text
├── cmd/
│   └── cc/main.go              # CLI entry point
└── testdata/
    ├── *.c                     # Test programs
    └── expect/                 # Expected assembly output
```

---

## Phase 0: Study the Toolkit and ChoiceScript

**Do this before writing any code.**

Read and understand these files from the EBNF toolkit:
- `parse/transform.go` -- the transform engine
- `parse/parser.go` -- the parsing engine
- `ebnf_parser.go` -- how `.ebnf` grammars are parsed

If you can access the ChoiceScript project, study:
- `resources/choicescript_lexer.ebnf` -- a real grammar of comparable complexity to C
- `transforms/expressions.go` -- how `transformBinarySequence()` works
- `transforms/conditionals.go` -- how block structures become AST nodes
- `transforms/merged.go` -- the linker pattern

Key insight from `transformBinarySequence()`: the grammar encodes operator precedence in the rule chain (`or_expr → and_expr → eq_expr → ... → primary_expr`). The transform function is identical at every level -- it just builds left-associative `BinaryOp` nodes from whatever the parser structured. One function, all precedence levels. This transfers directly to C with 6 more lines for the additional levels.

---

## Phase 1: The Grammar (`c.ebnf`)

**Estimated: ~500 lines**

This is a single `.ebnf` file containing the unified C and preprocessor grammar. The preprocessor is NOT a separate text phase. It's part of the same grammar.

### Key design decisions

**Preprocessor directives are grammar rules:**
```ebnf
preprocessing_directive = define_directive
                        | include_directive
                        | ifdef_directive
                        | ifndef_directive
                        | if_directive
                        | elif_directive
                        | else_directive
                        | endif_directive
                        | undef_directive
                        | line_directive
                        | error_directive
                        | pragma_directive ;

define_directive = <"#"> <ws>* <"define"> <ws>+ identifier replacement_list? <newline> ;
define_func_directive = <"#"> <ws>* <"define"> <ws>+ identifier <"("> param_list? <")"> replacement_list? <newline> ;
include_directive = <"#"> <ws>* <"include"> <ws>+ ( angle_path | string_path ) <newline> ;
ifdef_directive = <"#"> <ws>* <"ifdef"> <ws>+ identifier <newline> ;
endif_directive = <"#"> <ws>* <"endif"> <newline>? ;
```

They appear inline with C code. The parser produces a tree containing both preprocessor nodes and C nodes. Transform passes handle them in order.

**Expression precedence uses the same chain pattern as ChoiceScript:**
```ebnf
expr = assignment_expr ;
assignment_expr = conditional_expr ( assignment_op assignment_expr )? ;  (* right-assoc *)
conditional_expr = logical_or_expr ( <"?"> expr <":"> conditional_expr )? ;
logical_or_expr = logical_and_expr ( <"||"> logical_and_expr )* ;
logical_and_expr = bitwise_or_expr ( <"&&"> bitwise_or_expr )* ;
bitwise_or_expr = bitwise_xor_expr ( <"|"> bitwise_xor_expr )* ;
bitwise_xor_expr = bitwise_and_expr ( <"^"> bitwise_and_expr )* ;
bitwise_and_expr = equality_expr ( <"&"> equality_expr )* ;
equality_expr = relational_expr ( ( <"=="> | <"!=" > ) relational_expr )* ;
relational_expr = shift_expr ( ( <"<"> | <">"> | <"<="> | <">=" > ) shift_expr )* ;
shift_expr = additive_expr ( ( <"<<"> | <">>" > ) additive_expr )* ;
additive_expr = multiplicative_expr ( ( <"+"> | <"-" > ) multiplicative_expr )* ;
multiplicative_expr = cast_expr ( ( <"*"> | <"/"> | <"%"> ) cast_expr )* ;
cast_expr = unary_expr | <"("> type_name <")"> cast_expr ;
unary_expr = postfix_expr
           | <"++"> unary_expr | <"--"> unary_expr
           | unary_op cast_expr
           | <"sizeof"> ( <"("> type_name <")"> | unary_expr ) ;
postfix_expr = primary_expr postfix_suffix* ;
postfix_suffix = array_index | func_call | member_access | ptr_member_access | postfix_inc | postfix_dec ;
primary_expr = identifier | constant | string_literal | <"("> expr <")"> ;
```

**Declarations (the hard part of C's grammar):**
```ebnf
declaration = declaration_specifiers init_declarator_list? <";"> ;
declaration_specifiers = ( storage_class | type_specifier | type_qualifier )+ ;
type_specifier = <"void"> | <"char"> | <"short"> | <"int"> | <"long">
               | <"float"> | <"double"> | <"signed"> | <"unsigned">
               | struct_or_union_specifier | enum_specifier | typedef_name ;
declarator = pointer? direct_declarator ;
direct_declarator = ( identifier | <"("> declarator <")"> )
                    ( <"["> constant_expr? <"]"> | <"("> parameter_list <")"> )* ;
pointer = ( <"*"> type_qualifier* )+ ;
```

The typedef/identifier ambiguity (is `foo` a type name or a variable?) is resolved in the symbol table transform pass, not in the grammar. Parse it as identifier everywhere, then the symbol pass annotates which identifiers are typedef names. This is a standard approach and avoids context-sensitive grammar hacks.

**Statements are straightforward:**
```ebnf
statement = labeled_stmt | compound_stmt | expression_stmt
          | if_stmt | switch_stmt | while_stmt | do_stmt | for_stmt
          | goto_stmt | continue_stmt | break_stmt | return_stmt ;
compound_stmt = <"{"> block_item* <"}"> ;
if_stmt = <"if"> <"("> expr <")"> statement ( <"else"> statement )? ;
while_stmt = <"while"> <"("> expr <")"> statement ;
for_stmt = <"for"> <"("> for_init? <";"> expr? <";"> expr? <")"> statement ;
```

Simpler than ChoiceScript's indentation-sensitive blocks.

### Testing Phase 1

```bash
# Parse and print the tree -- verify it produces expected structure
go run ./cmd/cc -parse-only test.c
```

Write 10-15 small `.c` files covering: expressions, declarations, function definitions, struct/union/enum, pointer declarations, control flow, preprocessor directives mixed with code. Verify they all parse without error.

---

## Phase 2: AST Node Types (`ast/`)

**Estimated: ~1,000 lines**

Define Go types for every C construct. Follow the ChoiceScript pattern: each node type is a struct implementing a common interface.

```go
type Node interface {
    Pos() SourcePos  // source position for error reporting
}

type Expression interface {
    Node
    exprNode()
}

type Statement interface {
    Node
    stmtNode()
}

type Declaration interface {
    Node
    declNode()
}
```

**Expression nodes:**
```go
type BinaryExpr struct {
    Op    string
    Left  Expression
    Right Expression
    pos   SourcePos
}

type UnaryExpr struct {
    Op      string
    Operand Expression
    Prefix  bool        // true for prefix ++/-- and unary ops, false for postfix
    pos     SourcePos
}

type CallExpr struct {
    Func Expression
    Args []Expression
    pos  SourcePos
}

type CastExpr struct {
    Type    TypeName
    Operand Expression
    pos     SourcePos
}

// ... ~15 expression node types total
```

**Statement nodes:**
```go
type IfStmt struct {
    Cond Expression
    Then Statement
    Else Statement  // nil if no else
    pos  SourcePos
}

type ForStmt struct {
    Init Statement    // declaration or expression
    Cond Expression
    Post Expression
    Body Statement
    pos  SourcePos
}

type CompoundStmt struct {
    Items []BlockItem  // declarations and statements
    pos   SourcePos
}

// ... ~12 statement node types
```

**Declaration nodes:**
```go
type VarDecl struct {
    Name       string
    Type       Type
    Init       Expression  // nil if no initializer
    Storage    StorageClass
    pos        SourcePos
}

type FuncDecl struct {
    Name       string
    ReturnType Type
    Params     []ParamDecl
    Body       *CompoundStmt  // nil for prototypes
    pos        SourcePos
}

type StructDecl struct {
    Tag     string  // "" for anonymous
    Fields  []FieldDecl
    pos     SourcePos
}

// ... ~10 declaration node types
```

**Type representation:**
```go
type Type interface {
    typeNode()
    Size() int     // sizeof
    Align() int    // alignment
}

type BasicType struct {
    Kind BasicKind  // Void, Char, Short, Int, Long, Float, Double
    Signed bool
}

type PointerType struct {
    Base Type
}

type ArrayType struct {
    Base Type
    Len  Expression  // nil for unsized
}

type FuncType struct {
    Return Type
    Params []Type
    Variadic bool
}

type StructType struct {
    Decl *StructDecl
}
```

Also write an S-expression printer (`ast/print.go`) for debugging, exactly like ChoiceScript's `ast_printer.go`. Being able to see `(func main (params) (compound (return (+ 1 2))))` is invaluable for debugging transforms.

---

## Phase 3: Transform Maps

Each file is an independent work unit. Each is a separate `TransformMap` that gets merged in `init()`. **These can be developed in parallel by separate agents.**

### 3a. `transforms/expressions.go`

**Estimated: ~400 lines**

This is the most direct transfer from ChoiceScript. The pattern is nearly identical.

```go
var ExpressionTransforms = parse.TransformMap{
    // Precedence levels -- same factory function for all
    "logical_or_expr":      transformBinarySequence(),
    "logical_and_expr":     transformBinarySequence(),
    "bitwise_or_expr":      transformBinarySequence(),
    "bitwise_xor_expr":     transformBinarySequence(),
    "bitwise_and_expr":     transformBinarySequence(),
    "equality_expr":        transformBinarySequence(),
    "relational_expr":      transformBinarySequence(),
    "shift_expr":           transformBinarySequence(),
    "additive_expr":        transformBinarySequence(),
    "multiplicative_expr":  transformBinarySequence(),

    // Special cases
    "assignment_expr":      transformAssignment,       // right-associative
    "conditional_expr":     transformTernary,           // ? :
    "cast_expr":            transformCast,              // (type)expr
    "unary_expr":           transformUnary,             // prefix ops, sizeof
    "postfix_expr":         transformPostfix,           // a[i], a.b, a->b, a++, f()
    "primary_expr":         passThrough,

    // Literals
    "constant":             transformConstant,          // int, float, char literals
    "string_literal":       transformStringLiteral,
    "identifier":           transformIdentifier,

    // Operators -- extract the operator string
    "assignment_op":        extractOperator,
    "unary_op":             extractOperator,
}
```

`transformBinarySequence()` transfers from ChoiceScript with zero modifications. It builds left-associative `*ast.BinaryExpr` nodes. The grammar's rule chain handles precedence.

New work: `transformTernary` (~20 lines), `transformCast` (~15 lines), `transformPostfix` (~40 lines for array/member/call/inc/dec suffixes).

### 3b. `transforms/statements.go`

**Estimated: ~400 lines**

```go
var StatementTransforms = parse.TransformMap{
    "statement":        passThrough,
    "compound_stmt":    transformCompound,
    "if_stmt":          transformIf,
    "while_stmt":       transformWhile,
    "do_stmt":          transformDoWhile,
    "for_stmt":         transformFor,
    "switch_stmt":      transformSwitch,
    "case_label":       transformCase,
    "default_label":    transformDefault,
    "return_stmt":      transformReturn,
    "break_stmt":       func() *ast.BreakStmt { return &ast.BreakStmt{} },
    "continue_stmt":    func() *ast.ContinueStmt { return &ast.ContinueStmt{} },
    "goto_stmt":        transformGoto,
    "labeled_stmt":     transformLabeled,
    "expression_stmt":  transformExprStmt,
    "block_item":       passThrough,
}
```

Each transform is small and mechanical. `transformIf` extracts condition, then-body, optional else-body, returns `*ast.IfStmt`. Same pattern as ChoiceScript's `transformIfBlock`, but simpler (braces, not indentation).

### 3c. `transforms/declarations.go`

**Estimated: ~800 lines**

This is the hardest transform file because C declaration syntax is genuinely complex. But it's still mechanical: extract specifiers, extract declarators, build typed AST nodes.

```go
var DeclarationTransforms = parse.TransformMap{
    "declaration":              transformDeclaration,
    "declaration_specifiers":   transformDeclSpecifiers,
    "init_declarator_list":     transformInitDeclList,
    "declarator":               transformDeclarator,
    "direct_declarator":        transformDirectDeclarator,
    "pointer":                  transformPointer,
    "parameter_list":           transformParamList,
    "parameter_declaration":    transformParamDecl,
    "type_name":                transformTypeName,
    "struct_or_union_specifier": transformStructOrUnion,
    "enum_specifier":           transformEnum,
    "function_definition":      transformFuncDef,
    "translation_unit":         transformTranslationUnit,
    // ... ~20 more entries for declaration sub-productions
}
```

The key complexity: `declarator` is recursive (pointers to arrays of functions returning pointers to...). Build the type inside-out. This is ~100 lines of careful but mechanical code.

### 3d. `transforms/preprocessor.go`

**Estimated: ~1,200 lines**

The preprocessor is grammar rules + transform passes. NOT a separate text phase.

```go
var PreprocessorTransforms = parse.TransformMap{
    "define_directive":      transformDefine,
    "define_func_directive": transformDefineFunc,
    "undef_directive":       transformUndef,
    "include_directive":     transformInclude,
    "ifdef_directive":       transformIfdef,
    "ifndef_directive":      transformIfndef,
    "if_directive":          transformPPIf,
    "elif_directive":        transformPPElif,
    "else_directive":        transformPPElse,
    "endif_directive":       transformPPEndif,
    "line_directive":        transformLine,
    "error_directive":       transformPPError,
    "pragma_directive":      transformPragma,
}
```

**How it works in passes:**

1. **Pass 1 (top-down):** Evaluate `#ifdef`/`#if`/`#else`/`#endif`. Prune dead branches from the tree. Use `TransformContext.State` to track defined macros. This is structurally identical to ChoiceScript's `transformOrphanElseBlock` -- conditional pruning of tree branches.

2. **Pass 2 (recursive):** Process `#include`. Parse the included file (recursive call to the parser), graft its tree into the current tree at the include point. The toolkit's `parse.New(grammar).Parse()` is reentrant -- you can call it from inside a transform.

3. **Pass 3 (bottom-up, iterative):** Expand macros. Walk the tree, find identifier nodes matching `#define` entries in state, replace with expansion. Repeat until no more expansions occur (handles macros that expand to other macros). This is the most complex single transform, but it's still a tree walk with substitution.

**Macro expansion detail:**

```go
func transformMacroExpansion(ctx *parse.TransformContext, node *parse.Node, parts ...interface{}) (interface{}, error) {
    macros := ctx.State["macros"].(map[string]*MacroDef)

    // Check if this identifier is a defined macro
    if ident, ok := parts[0].(string); ok {
        if macro, exists := macros[ident]; exists {
            // Object-like macro: substitute the replacement token list
            // Function-like macro: bind arguments, substitute
            // Handle stringification (#) and token pasting (##)
            return expandMacro(macro, parts, ctx)
        }
    }
    return parts[0], nil  // not a macro, pass through
}
```

Stringification (`#arg`) and token pasting (`a##b`) add complexity but are well-specified operations on token sequences. ~200 lines for complete macro expansion including edge cases.

### 3e. `transforms/symbols.go`

**Estimated: ~600 lines**

Symbol table construction as a transform pass using `TransformContext.State`.

```go
var SymbolTransforms = parse.TransformMap{
    "compound_stmt":    symbolEnterScope,   // push scope on {
    "function_definition": symbolEnterFunc, // push function scope
    "declaration":      symbolRegisterDecl, // register name → type
    "identifier":       symbolResolve,      // look up name in scope chain
}
```

The scope stack lives in `TransformContext.State["scopes"]`:

```go
type Scope struct {
    Parent  *Scope
    Symbols map[string]*Symbol
}

type Symbol struct {
    Name    string
    Type    ast.Type
    Kind    SymbolKind  // Variable, Function, TypeDef, EnumConst, Label
    Defined bool        // forward declaration vs definition
}
```

Push scope on `{`, pop on `}`. Register declarations as they're encountered. Resolve identifiers by walking the scope chain. This is where the typedef/identifier ambiguity gets resolved: if an identifier resolves to a `TypeDef` symbol, annotate the AST node.

### 3f. `transforms/typecheck.go`

**Estimated: ~1,500 lines**

Type checking and implicit coercion as a transform pass. This is where most of C's semantic complexity lives.

```go
var TypeCheckTransforms = parse.TransformMap{
    "binary_expr":      typecheckBinary,
    "assignment_expr":  typecheckAssignment,
    "call_expr":        typecheckCall,
    "cast_expr":        typecheckCast,
    "return_stmt":      typecheckReturn,
    "array_index":      typecheckIndex,
    "member_access":    typecheckMember,
    "declaration":      typecheckInit,      // initializer matches declared type
}
```

The bulk of this file is implementing C's type promotion and conversion rules:
- Integer promotion (char/short → int)
- Usual arithmetic conversions (int + float → float)
- Pointer arithmetic (ptr + int → ptr)
- Assignment compatibility (implicit conversions)
- Function argument coercion
- Lvalue checking (can't assign to `a + b`)

This is dense, mechanical, well-specified domain knowledge. The C11 standard defines every rule precisely. It's tedious but there are no design decisions -- just implement the spec.

When a coercion is needed, the transform inserts an explicit `*ast.CastExpr` node into the tree. After this pass, all conversions are explicit in the AST.

### 3g. `transforms/emit.go`

**Estimated: ~1,500 lines**

The final transform: AST → x86-64 assembly text.

**C was designed for this to be shallow.** Each AST node maps to a small, obvious sequence of instructions. This is not coincidence -- it's the entire design purpose of the language.

```go
var EmitTransforms = parse.TransformMap{
    // Expressions → register-based codegen
    "binary_expr":     emitBinary,
    "unary_expr":      emitUnary,
    "call_expr":       emitCall,
    "constant":        emitConstant,
    "identifier":      emitLoad,
    "assign_expr":     emitStore,
    "cast_expr":       emitCast,
    "string_literal":  emitStringLit,

    // Statements → control flow
    "if_stmt":         emitIf,
    "while_stmt":      emitWhile,
    "for_stmt":        emitFor,
    "do_stmt":         emitDoWhile,
    "switch_stmt":     emitSwitch,
    "return_stmt":     emitReturn,
    "compound_stmt":   emitCompound,
    "goto_stmt":       emitGoto,
    "labeled_stmt":    emitLabel,

    // Declarations → stack allocation
    "var_decl":        emitVarDecl,
    "func_decl":       emitFuncDecl,

    // Top level → sections
    "translation_unit": emitTranslationUnit,
}
```

**Register allocation strategy:** Start simple. Use a stack-based approach:
- Each expression evaluates to `%rax`
- Binary ops: evaluate left → push → evaluate right → pop left → op
- Function args: evaluate into ABI registers (`%rdi`, `%rsi`, `%rdx`, `%rcx`, `%r8`, `%r9`), spill to stack beyond 6
- Local variables: fixed stack slots, assigned during `emitFuncDecl`

This is not optimal but it's correct, and it's ~800 lines. Optimization is a later pass (also a TransformMap on the assembly IR, if desired).

**Example:**
```go
func emitBinary(ctx *parse.TransformContext, op string, left, right string) string {
    // left and right are already emitted code strings
    // that leave their results in %rax
    reg := ctx.State["temp_reg"].(int)
    ctx.State["temp_reg"] = reg + 1

    var instr string
    switch op {
    case "+": instr = "addq"
    case "-": instr = "subq"
    case "*": instr = "imulq"
    case "/": instr = "idivq"  // needs special handling for %rdx:%rax
    case "==", "!=", "<", ">", "<=", ">=": return emitComparison(op, left, right)
    }

    return left +
        fmt.Sprintf("    movq %%rax, %%r%d\n", 10+reg) +
        right +
        fmt.Sprintf("    %s %%r%d, %%rax\n", instr, 10+reg)
}
```

The transforms produce AT&T syntax x86-64 assembly text. Feed it to `as` and `ld`. You do not need your own assembler or linker for v1.

### 3h. `transforms/merged.go`

**Estimated: ~60 lines**

The linker. Identical in structure to ChoiceScript's.

```go
package transforms

import "github.com/wbrown/ebnf/parse"

var CTransforms = parse.TransformMap{}

func init() {
    for k, v := range ExpressionTransforms    { CTransforms[k] = v }
    for k, v := range StatementTransforms     { CTransforms[k] = v }
    for k, v := range DeclarationTransforms   { CTransforms[k] = v }
    for k, v := range PreprocessorTransforms  { CTransforms[k] = v }
    for k, v := range SymbolTransforms        { CTransforms[k] = v }
    for k, v := range TypeCheckTransforms     { CTransforms[k] = v }
    for k, v := range EmitTransforms          { CTransforms[k] = v }

    // Structural pass-throughs
    CTransforms["ws"] = func(...interface{}) string { return "" }
    CTransforms["newline"] = func(...interface{}) string { return "" }
}
```

Each transform map occupies its own file, its own key space. They compose without conflict.

---

## Phase 4: Multi-Pass Pipeline

The `TransformMap` entries above are organized by *domain* (expressions, statements, etc.) for development, but they execute in *passes* for correctness. Some passes need to run before others.

```go
func Compile(source string, filename string) (string, error) {
    grammar, _ := ebnf.LoadGrammar("c.ebnf")
    parser := parse.New(grammar)

    // Parse: source → parse tree
    tree, err := parser.Parse(source, "translation_unit")
    if err != nil {
        return "", err
    }

    // Pass 1: Preprocess (conditional compilation, macro expansion, includes)
    tree, err = preprocessPass(tree, PreprocessorTransforms)
    if err != nil {
        return "", err
    }

    // Pass 2: AST construction (parse tree → typed AST nodes)
    // Uses ExpressionTransforms + StatementTransforms + DeclarationTransforms
    astResult, err := parse.Transform(tree, ASTTransforms)
    if err != nil {
        return "", err
    }

    // Pass 3: Symbol resolution
    astResult, err = parse.Transform(astResult, SymbolTransforms)
    if err != nil {
        return "", err
    }

    // Pass 4: Type checking and coercion insertion
    astResult, err = parse.Transform(astResult, TypeCheckTransforms)
    if err != nil {
        return "", err
    }

    // Pass 5: Code emission (AST → assembly text)
    asm, err := parse.Transform(astResult, EmitTransforms)
    if err != nil {
        return "", err
    }

    return asm.(string), nil
}
```

Each pass is a `Transform()` call with a different map. The output of one pass is the input to the next. The toolkit handles the tree walking. You just write the per-node functions.

Note: Passes 2-4 could be merged into a single `CTransforms` map if bottom-up ordering is sufficient. Separate passes are needed when a later transform depends on information gathered by an earlier one (e.g., type checking needs the symbol table built first). The multi-pass support in the toolkit handles this.

---

## Phase 5: Testing Strategy

### Unit tests per transform file

Each transform file gets its own `_test.go` with small, focused tests:

```go
func TestBinaryExprAdd(t *testing.T) {
    tree := parseExpr("2 + 3")
    result, err := parse.Transform(tree, ExpressionTransforms)
    require.NoError(t, err)
    binop := result.(*ast.BinaryExpr)
    assert.Equal(t, "+", binop.Op)
    // ... check left and right
}
```

### Integration tests: small C programs

```c
// testdata/add.c
int main() { return 2 + 3; }
```

```bash
./cc testdata/add.c -o /tmp/add && /tmp/add; echo $?
# Expected: 5
```

### Progressive test suite

Build up in order of complexity:
1. **Return constants:** `int main() { return 42; }`
2. **Binary arithmetic:** `return 2 + 3;`
3. **Local variables:** `int x = 5; return x;`
4. **If/else:** `if (x > 0) return 1; else return 0;`
5. **Loops:** `while`, `for`, `do-while`
6. **Functions:** calls, parameters, recursion
7. **Pointers:** `*p`, `&x`, `p->field`
8. **Structs:** definition, access, nested
9. **Arrays:** declaration, indexing, pointer decay
10. **Preprocessor:** `#define`, `#include`, `#ifdef`
11. **Standard library:** link against libc, call `printf`
12. **Real programs:** small open-source projects

### Regression suite

Once a test passes, it must never break. Run the full suite before every commit. Use the ChoiceScript pattern: integration tests on real files, not just synthetic examples.

### External test suites

When the basics work:
- [c-testsuite](https://github.com/c-testsuite/c-testsuite) -- community C compiler test suite
- GCC torture tests -- edge cases and corner cases
- Compile small real programs (a libc-free hello world, then small utilities)

---

## Agent Work Allocation

If using parallel agents, each agent owns one transform file. They cannot conflict because each file defines a separate `TransformMap` with unique keys.

| Agent | File | Dependencies |
|---|---|---|
| Agent 1 | `c.ebnf` + `ast/` | None -- do this first or in parallel with transforms using stub AST types |
| Agent 2 | `transforms/expressions.go` | AST types |
| Agent 3 | `transforms/statements.go` | AST types |
| Agent 4 | `transforms/declarations.go` | AST types |
| Agent 5 | `transforms/preprocessor.go` | AST types |
| Agent 6 | `transforms/symbols.go` | AST types + declaration transforms |
| Agent 7 | `transforms/typecheck.go` | AST types + symbol transforms |
| Agent 8 | `transforms/emit.go` | AST types |

Agents 2, 3, 4, 5, 8 can work fully in parallel once AST types exist.
Agent 6 needs declaration transforms to be mostly done.
Agent 7 needs symbol transforms to be mostly done.

The `merged.go` file is trivial and can be written by any agent.

---

## What NOT To Do

1. **Do not build a separate preprocessor.** It's grammar rules and transforms. Same `.ebnf`, same pipeline.

2. **Do not build your own assembler or linker for v1.** Use `as` and `ld`. The compiler's job is to produce correct assembly text. Assembler/linker are separate projects.

3. **Do not add an optimization pass for v1.** Get correctness first. Optimization is a later `TransformMap` on the IR. It can be added without changing any existing code.

4. **Do not invent infrastructure.** The toolkit provides the tree walker, the transform engine, the multi-pass support, the context/state system. Write transform *functions*, not frameworks.

5. **Do not try to handle all of C11 immediately.** Start with the subset needed to compile `int main() { return 42; }` and expand from there. Each new feature is a few more `TransformMap` entries.

6. **Do not add abstraction layers.** No adapters, no wrappers, no "flexibility." Each grammar rule maps to one transform function. That's the architecture. It's enough.

---

## Milestone Targets

**M1: "return 42"**
- Grammar: expressions, function definitions, return statement
- Transforms: expression, statement, emit (minimal)
- Output: working x86-64 assembly for `int main() { return 42; }`
- Assemble with `as`, link with `ld`, run, verify exit code is 42

**M2: Arithmetic**
- All binary and unary operators
- Correct precedence and associativity
- `int main() { return (2 + 3) * 4 - 1; }` → 19

**M3: Variables and control flow**
- Local variable declarations
- if/else, while, for, do-while
- Fibonacci, factorial, fizzbuzz compile and run correctly

**M4: Functions**
- Function declarations and definitions
- Parameter passing (System V AMD64 ABI)
- Recursion works
- Link against libc, call `printf`

**M5: Pointers and structs**
- Pointer arithmetic, dereference, address-of
- Struct definition, member access, `->` operator
- Dynamic memory via libc `malloc`/`free`

**M6: Preprocessor**
- `#define` object and function macros
- `#include` with file loading
- `#ifdef`/`#ifndef`/`#if`/`#else`/`#endif`
- Can compile programs that use standard headers

**M7: Real programs**
- Compile small open-source C projects
- Pass c-testsuite
- Fix edge cases as discovered

---

## Key Insight

The Anthropic blog describes 16 agents producing 100,000 lines of Rust in two weeks for $20K. They solved two problems simultaneously: *how to build a compiler* and *what C means*.

You only need to solve *what C means*. The "how" is the EBNF toolkit. Every pass is a `TransformMap`. The parse tree walks itself. The transforms compose by construction.

Write the grammar. Define the AST types. Fill in the transform functions. That's it. That's the whole compiler.
