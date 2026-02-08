# Building a C Compiler on the EBNF Transform Toolkit

## For a Single Developer (Human or Claude)

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

## Before You Start: Study the Toolkit

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

## Build Order

This is a sequential plan. Each step produces something you can test before moving on. Do not skip ahead. Each milestone builds on the last.

---

### M1: "return 42"

**Goal:** `int main() { return 42; }` compiles to x86-64 assembly, assembles with `as`, links with `ld`, runs, exit code is 42.

This milestone is deliberately tiny. It proves the entire pipeline works end-to-end: grammar → parse → transform → emit → assemble → link → run. Everything after this is adding more grammar rules and more transform entries.

#### M1a: Minimal grammar

Write the smallest `.ebnf` that parses `int main() { return 42; }`:

```ebnf
translation_unit = function_definition+ ;

function_definition = type_specifier declarator compound_stmt ;

type_specifier = "void" | "char" | "short" | "int" | "long"
               | "float" | "double" | "signed" | "unsigned" ;

declarator = identifier <"("> parameter_list? <")"> ;
parameter_list = "void" | parameter ( <","> parameter )* ;
parameter = type_specifier identifier? ;

compound_stmt = <"{"> block_item* <"}"> ;
block_item = statement ;

statement = return_stmt ;
return_stmt = <"return"> expr <";"> ;

expr = constant ;
constant = #"[0-9]+" ;

identifier = #"[a-zA-Z_][a-zA-Z0-9_]*" ;

ws = #"\\s+" ;
```

That's maybe 20 lines. It parses exactly one shape of program. Test it:

```bash
go run ./cmd/cc -parse-only testdata/return42.c
```

Verify you get a parse tree. Print it. Look at it.

#### M1b: Minimal AST types

You need exactly 5 types for M1:

```go
type TranslationUnit struct { Decls []Declaration }
type FuncDecl struct { Name string; ReturnType string; Body *CompoundStmt }
type CompoundStmt struct { Items []Statement }
type ReturnStmt struct { Value Expression }
type IntLiteral struct { Value int }
```

And an S-expression printer so you can see `(func main int (compound (return 42)))`.

#### M1c: Minimal transforms

```go
var M1Transforms = parse.TransformMap{
    "translation_unit":    transformTranslationUnit,
    "function_definition": transformFuncDef,
    "type_specifier":      extractText,
    "declarator":          extractIdentifier,
    "compound_stmt":       transformCompound,
    "return_stmt":         transformReturn,
    "constant":            transformIntLiteral,
    "identifier":          extractText,
}
```

~8 entries. Each function is 5-15 lines. Total: ~100 lines.

#### M1d: Minimal emit

```go
var M1Emit = parse.TransformMap{
    "translation_unit": func(decls ...string) string {
        return strings.Join(decls, "\n")
    },
    "func_decl": func(name string, body string) string {
        return fmt.Sprintf("    .globl %s\n%s:\n    pushq %%rbp\n    movq %%rsp, %%rbp\n%s    popq %%rbp\n    ret\n", name, name, body)
    },
    "return_stmt": func(val string) string {
        return fmt.Sprintf("    movl $%s, %%eax\n", val)
    },
    "int_literal": func(val int) string {
        return strconv.Itoa(val)
    },
}
```

~4 entries. The emit transform returns assembly text. The text goes to `as`, then `ld`.

#### M1e: Wire it up

```go
func main() {
    source := readFile(os.Args[1])
    grammar, _ := ebnf.LoadGrammar("c.ebnf")
    parser := parse.New(grammar)
    tree, _ := parser.Parse(source, "translation_unit")
    ast, _ := parse.Transform(tree, M1Transforms)
    asm, _ := parse.Transform(ast, M1Emit)
    writeFile("/tmp/out.s", asm.(string))
    exec.Command("as", "-o", "/tmp/out.o", "/tmp/out.s").Run()
    exec.Command("ld", "-o", os.Args[2], "/tmp/out.o").Run()
}
```

Test:

```bash
go run ./cmd/cc testdata/return42.c -o /tmp/return42
/tmp/return42; echo $?
# Must print: 42
```

When this works, the entire pipeline is proven. Everything from here is incremental.

---

### M2: Arithmetic

**Goal:** `int main() { return (2 + 3) * 4 - 1; }` → exit code 19.

#### Expand the grammar

Add the expression precedence chain:

```ebnf
expr = assignment_expr ;
assignment_expr = conditional_expr ;
conditional_expr = logical_or_expr ;
logical_or_expr = logical_and_expr ( <"||"> logical_and_expr )* ;
(* ... full chain ... *)
additive_expr = multiplicative_expr ( ( "+" | "-" ) multiplicative_expr )* ;
multiplicative_expr = unary_expr ( ( "*" | "/" | "%" ) unary_expr )* ;
unary_expr = postfix_expr | "-" unary_expr | "+" unary_expr ;
postfix_expr = primary_expr ;
primary_expr = constant | <"("> expr <")"> ;
```

The middle levels (logical, bitwise, relational, shift) exist in the grammar for correct precedence but produce no new transform code -- `transformBinarySequence()` handles them all. Add them now so you never have to restructure the expression chain later.

#### Add expression transforms

Port `transformBinarySequence()` from ChoiceScript. Add all 15 precedence levels:

```go
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
```

All untested levels are pass-throughs until you write tests that exercise them. But the grammar is correct now.

#### Add emit for binary ops

Stack-based codegen: evaluate left → push → evaluate right → pop → op.

```go
"binary_expr": func(ctx *parse.TransformContext, op string, left, right string) string {
    return left +
        "    pushq %rax\n" +
        right +
        "    popq %rcx\n" +
        emitOp(op)  // "addq %rcx, %rax" etc.
},
```

Not optimal. Correct. Optimization is a later pass.

#### Test

```bash
echo 'int main() { return (2 + 3) * 4 - 1; }' > /tmp/arith.c
go run ./cmd/cc /tmp/arith.c -o /tmp/arith
/tmp/arith; echo $?
# Must print: 19
```

Write 10-20 arithmetic tests covering: all operators, precedence, associativity, parentheses, unary minus. Run them all. They must all pass before you move on.

---

### M3: Variables and Control Flow

**Goal:** Fibonacci, factorial, and fizzbuzz compile and run correctly.

#### Grammar additions

```ebnf
(* Declarations *)
declaration = type_specifier init_declarator_list <";"> ;
init_declarator_list = init_declarator ( <","> init_declarator )* ;
init_declarator = declarator ( <"="> expr )? ;

(* Statements *)
statement = compound_stmt | if_stmt | while_stmt | for_stmt | do_stmt
          | return_stmt | expression_stmt | break_stmt | continue_stmt ;
if_stmt = <"if"> <"("> expr <")"> statement ( <"else"> statement )? ;
while_stmt = <"while"> <"("> expr <")"> statement ;
for_stmt = <"for"> <"("> for_init? <";"> expr? <";"> expr? <")"> statement ;
do_stmt = <"do"> statement <"while"> <"("> expr <")"> <";"> ;
expression_stmt = expr <";"> ;
```

Block item expands to include declarations:
```ebnf
block_item = declaration | statement ;
```

#### New AST types

`VarDecl`, `IfStmt`, `WhileStmt`, `ForStmt`, `DoWhileStmt`, `AssignExpr`. ~80 lines.

#### New transforms

```go
"declaration":     transformDeclaration,
"if_stmt":         transformIf,
"while_stmt":      transformWhile,
"for_stmt":        transformFor,
"do_stmt":         transformDoWhile,
"expression_stmt": transformExprStmt,
"assignment_expr": transformAssignment,
```

Each is small. `transformIf` extracts condition, then-body, optional else, returns `*ast.IfStmt`. Same shape as ChoiceScript's conditional transforms but simpler (braces, not indentation).

#### Emit additions

Local variables get stack slots. Track in `TransformContext.State`:

```go
"var_decl": func(ctx *parse.TransformContext, name string, init string) string {
    locals := ctx.State["locals"].(map[string]int)
    offset := len(locals) * 8 + 8
    locals[name] = offset
    if init != "" {
        return init + fmt.Sprintf("    movq %%rax, -%d(%%rbp)\n", offset)
    }
    return ""
},
```

Control flow uses labels and jumps:

```go
"if_stmt": func(ctx *parse.TransformContext, cond, then, else_ string) string {
    label := ctx.State["next_label"].(int)
    ctx.State["next_label"] = label + 2
    if else_ == "" {
        return cond +
            fmt.Sprintf("    cmpq $0, %%rax\n    je .L%d\n", label) +
            then +
            fmt.Sprintf(".L%d:\n", label)
    }
    return cond +
        fmt.Sprintf("    cmpq $0, %%rax\n    je .L%d\n", label) +
        then +
        fmt.Sprintf("    jmp .L%d\n.L%d:\n", label+1, label) +
        else_ +
        fmt.Sprintf(".L%d:\n", label+1)
},
```

#### Test

```c
// testdata/fibonacci.c
int main() {
    int a = 0;
    int b = 1;
    int i = 0;
    while (i < 10) {
        int t = a + b;
        a = b;
        b = t;
        i = i + 1;
    }
    return a;  // fib(10) = 55
}
```

```bash
go run ./cmd/cc testdata/fibonacci.c -o /tmp/fib
/tmp/fib; echo $?
# Must print: 55
```

---

### M4: Functions

**Goal:** Recursive factorial, multi-function programs, link against libc and call `printf`.

#### Grammar additions

Update `declarator` to handle parameter declarations properly. Add function calls to expressions (already in postfix from M2's grammar, now wire up the transform).

#### Key transforms

```go
"call_expr": transformCall,
"function_definition": transformFuncDef,  // update to handle params
"parameter_declaration": transformParamDecl,
```

#### Emit: calling convention

System V AMD64 ABI: first 6 integer args in `%rdi`, `%rsi`, `%rdx`, `%rcx`, `%r8`, `%r9`. Return value in `%rax`. This is the only ABI you need for Linux x86-64.

```go
"call_expr": func(ctx *parse.TransformContext, name string, args []string) string {
    abiRegs := []string{"%rdi", "%rsi", "%rdx", "%rcx", "%r8", "%r9"}
    result := ""
    for i, arg := range args {
        result += arg
        if i < 6 {
            result += fmt.Sprintf("    movq %%rax, %s\n", abiRegs[i])
        } else {
            result += "    pushq %rax\n"  // spill to stack
        }
    }
    result += fmt.Sprintf("    call %s\n", name)
    return result
},
```

#### Test

```c
// testdata/factorial.c
int factorial(int n) {
    if (n <= 1) return 1;
    return n * factorial(n - 1);
}
int main() { return factorial(5); }  // 120
```

Then link against libc:

```c
// testdata/hello.c
int printf(const char *fmt, ...);
int main() {
    printf("hello %d\n", 42);
    return 0;
}
```

```bash
go run ./cmd/cc testdata/hello.c -o /tmp/hello -lc
/tmp/hello
# Must print: hello 42
```

Linking against libc: use `cc` as the linker driver instead of raw `ld`. `cc -o out out.o` handles the C runtime startup (`crt0`, `crti`, etc.) and libc linking automatically. Your compiler's job is to produce a correct `.o` file.

---

### M5: Pointers and Structs

**Goal:** Pointer arithmetic, dereference, address-of, struct member access, dynamic allocation.

#### Grammar additions

```ebnf
(* Pointer declarators *)
pointer = ( <"*"> type_qualifier* )+ ;
declarator = pointer? direct_declarator ;

(* Pointer/member expressions -- already in postfix_suffix from M2 *)
postfix_suffix = array_index | func_call | member_access | ptr_member_access | postfix_inc | postfix_dec ;
array_index = <"["> expr <"]"> ;
member_access = <"."> identifier ;
ptr_member_access = <"->"> identifier ;

(* Unary address-of and dereference *)
unary_expr = postfix_expr
           | <"++"> unary_expr | <"--"> unary_expr
           | <"&"> cast_expr | <"*"> cast_expr
           | <"-"> cast_expr | <"+"> cast_expr
           | <"!"> cast_expr | <"~"> cast_expr
           | <"sizeof"> ( <"("> type_name <")"> | unary_expr ) ;

(* Struct/union *)
struct_or_union_specifier = ( <"struct"> | <"union"> ) identifier? <"{"> struct_declaration+ <"}">
                          | ( <"struct"> | <"union"> ) identifier ;
struct_declaration = type_specifier struct_declarator_list <";"> ;
```

#### Key insight for emit

C was designed as portable assembly. Pointer dereference is a load instruction. Address-of produces the address already in a register or stack slot. Struct member access is base + offset.

```go
"deref": func(ptr string) string {
    return ptr + "    movq (%rax), %rax\n"
},
"addr_of": func(ctx *parse.TransformContext, name string) string {
    offset := ctx.State["locals"].(map[string]int)[name]
    return fmt.Sprintf("    leaq -%d(%%rbp), %%rax\n", offset)
},
"member_access": func(ctx *parse.TransformContext, base string, field string) string {
    offset := lookupFieldOffset(ctx, field)
    return base + fmt.Sprintf("    movq %d(%%rax), %%rax\n", offset)
},
```

The mapping is direct because C was designed for it to be direct.

#### This is where you need the symbol table

Struct field offsets require knowing the struct layout. `symbols.go` builds a scope chain with type information. `typecheck.go` resolves field offsets. These passes must run before `emit.go`.

Now is when you split the transform maps into separate passes as described in the pipeline:

```go
// Pass 1: AST construction (expressions + statements + declarations)
// Pass 2: Symbol resolution (scope chain, name → type)
// Pass 3: Type checking (field offsets, pointer arithmetic types, sizeof)
// Pass 4: Emit
```

#### Test

```c
// testdata/struct.c
struct point { int x; int y; };
int main() {
    struct point p;
    p.x = 3;
    p.y = 4;
    return p.x + p.y;  // 7
}
```

```c
// testdata/pointer.c
int main() {
    int x = 42;
    int *p = &x;
    return *p;  // 42
}
```

---

### M6: The Preprocessor

**Goal:** `#define`, `#include`, `#ifdef`, function-like macros. Can compile programs that use standard headers.

#### Why this comes late

Not because the preprocessor is hard, but because you need a working compiler to test it against. Preprocessor bugs are invisible if you can't compile the result.

#### Grammar additions

The preprocessor directives go into the same `.ebnf` file:

```ebnf
(* Top level allows preprocessor directives mixed with declarations *)
translation_unit = ( preprocessing_directive | function_definition | declaration )* ;

preprocessing_directive = define_directive | include_directive
                        | ifdef_directive | ifndef_directive
                        | if_directive | elif_directive
                        | else_directive | endif_directive
                        | undef_directive | line_directive
                        | error_directive | pragma_directive ;

define_directive = <"#"> <ws>* <"define"> <ws>+ identifier
                   ( <"("> macro_params? <")"> )? replacement_list? <newline> ;
include_directive = <"#"> <ws>* <"include"> <ws>+ ( angle_path | string_path ) <newline> ;
ifdef_directive = <"#"> <ws>* <"ifdef"> <ws>+ identifier <newline> ;
ifndef_directive = <"#"> <ws>* <"ifndef"> <ws>+ identifier <newline> ;
if_directive = <"#"> <ws>* <"if"> <ws>+ pp_expr <newline> ;
elif_directive = <"#"> <ws>* <"elif"> <ws>+ pp_expr <newline> ;
else_directive = <"#"> <ws>* <"else"> <newline> ;
endif_directive = <"#"> <ws>* <"endif"> <newline>? ;
undef_directive = <"#"> <ws>* <"undef"> <ws>+ identifier <newline> ;
```

#### Preprocessor transforms are a separate pass

The preprocessor runs BEFORE AST construction. It operates on the parse tree, not the AST.

```go
func Compile(source string) (string, error) {
    tree, _ := parser.Parse(source, "translation_unit")

    // Pass 0: Preprocess (before AST construction)
    tree = preprocessConditionals(tree)  // prune #ifdef branches
    tree = preprocessIncludes(tree)      // parse and graft #include files
    tree = preprocessMacros(tree)        // expand #define macros

    // Pass 1: AST construction (on the preprocessed tree)
    ast, _ := parse.Transform(tree, ASTTransforms)
    // ... remaining passes ...
}
```

**Conditional compilation** (`#ifdef`/`#if`/`#else`/`#endif`): Top-down transform. Walk the tree, evaluate conditions using `TransformContext.State["defines"]`, prune dead branches. Structurally identical to ChoiceScript's `transformOrphanElseBlock` -- conditional tree pruning.

**Include processing** (`#include`): When you hit an include node, parse the included file with the same grammar, graft its subtree into the current tree at the include point. The parser is reentrant.

**Macro expansion** (`#define`): Bottom-up transform. Walk identifiers, check against macro table in state, substitute. Iterate until stable (handles macros expanding to other macros). Function-like macros: bind arguments, substitute in the replacement list. Stringification and token pasting are mechanical operations on tokens.

#### Test

```c
// testdata/preprocess.c
#define MAX(a, b) ((a) > (b) ? (a) : (b))
#define N 10

#ifdef N
int main() {
    return MAX(3, N);  // 10
}
#endif
```

Then test `#include` with a simple header file. Then test with an actual system header (`<stdio.h>`, `<stdlib.h>`). System headers will expose edge cases in both the preprocessor and the rest of the compiler. Fix them as they appear.

---

### M7: Declarations, Typedefs, Enums, and the Rest

**Goal:** Complete C11 declaration syntax. The full type system.

This is where C's grammar gets genuinely complex. Declarations are the hard part.

#### The typedef problem

```c
typedef int size_t;
size_t x;  // Is size_t a type or a variable? Depends on the typedef above.
```

Resolve in the symbol pass. Parse everything as identifiers. After symbol resolution, annotate which identifiers are typedef names. If a later pass needs to distinguish types from variables, the annotation is there.

#### Declaration transforms

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
}
```

`transformDeclarator` is the complex one: C's "declaration mirrors use" means `int (*f)(int, int)` declares a pointer to a function. Build the type inside-out by recursing through the declarator tree. ~100 lines of careful code.

#### Type checking pass

Once you have the full type system, implement the type checking transform:
- Integer promotion (char/short → int)
- Usual arithmetic conversions (int + float → float)
- Pointer arithmetic rules (ptr + int → ptr, ptr - ptr → ptrdiff_t)
- Assignment compatibility
- Function argument coercion
- Insert explicit cast nodes where implicit conversions occur

This is the largest single file (~1,500 lines) but it's entirely mechanical -- the C11 standard specifies every rule. No design decisions. Just implement the spec.

---

### M8: Real Programs

**Goal:** Compile small real-world C programs. Pass external test suites.

#### Progressive targets

1. **Your own test programs** -- should already be passing from M1-M7
2. **c-testsuite** -- community compiler test suite, hundreds of small programs
3. **Single-file programs** -- small utilities, algorithms
4. **Small projects** -- a tiny libc, a hash table, a simple allocator
5. **Multi-file compilation** -- compile each `.c` to `.o`, link together
6. **Real open-source projects** -- SQLite (single-file amalgamation is a good target)

Each new program you try will expose missing features or edge cases. Add the grammar rules and transform entries as needed. The architecture doesn't change -- you're just filling in more map entries.

#### Use GCC as an oracle

When something doesn't work, compile the same program with `gcc -S` and compare the assembly output. Your assembly doesn't need to match GCC's -- it just needs to be correct. But GCC's output shows you what instructions a correct implementation produces.

---

## Multi-Pass Pipeline

The complete pipeline when all milestones are done:

```go
func Compile(source string, filename string) (string, error) {
    grammar, _ := ebnf.LoadGrammar("c.ebnf")
    parser := parse.New(grammar)

    // Parse
    tree, err := parser.Parse(source, "translation_unit")
    if err != nil {
        return "", err
    }

    // Pass 0: Preprocess
    tree = preprocessConditionals(tree)
    tree = preprocessIncludes(tree)
    tree = preprocessMacros(tree)

    // Pass 1: AST construction
    astResult, err := parse.Transform(tree, ASTConstructionTransforms)
    if err != nil {
        return "", err
    }

    // Pass 2: Symbol resolution
    astResult, err = transformWithState(astResult, SymbolTransforms)
    if err != nil {
        return "", err
    }

    // Pass 3: Type checking and coercion
    astResult, err = transformWithState(astResult, TypeCheckTransforms)
    if err != nil {
        return "", err
    }

    // Pass 4: Emit
    asm, err := parse.Transform(astResult, EmitTransforms)
    if err != nil {
        return "", err
    }

    return asm.(string), nil
}
```

Five `Transform()` calls. Five maps. That's the whole compiler.

---

## Transform Map Reference

Summary of all transform maps, their purpose, and approximate size:

| File | Map | Purpose | Est. lines | When to build |
|---|---|---|---:|---|
| `expressions.go` | `ExpressionTransforms` | Operator precedence, literals, identifiers | ~400 | M1-M2 |
| `statements.go` | `StatementTransforms` | if, while, for, switch, return, etc. | ~400 | M3 |
| `declarations.go` | `DeclarationTransforms` | Variable/function/struct/enum declarations | ~800 | M1 (minimal), M5-M7 (full) |
| `preprocessor.go` | `PreprocessorTransforms` | #define, #include, #ifdef | ~1,200 | M6 |
| `symbols.go` | `SymbolTransforms` | Scope chain, name resolution, typedef | ~600 | M5 |
| `typecheck.go` | `TypeCheckTransforms` | Type promotion, coercion, lvalue checking | ~1,500 | M7 |
| `emit.go` | `EmitTransforms` | AST → x86-64 assembly | ~1,500 | M1 (minimal), grows each milestone |
| `merged.go` | `CTransforms` | Merges all maps in init() | ~60 | M1 |
| **Total** | | | **~6,500** | |

Plus ~1,000 lines of AST types, ~500 lines of grammar, ~500 lines of CLI and glue. Grand total: **~8,500 lines** of C-specific code on top of the 4,800-line toolkit.

---

## What NOT To Do

1. **Do not build a separate preprocessor.** It's grammar rules and transforms. Same `.ebnf`, same pipeline.

2. **Do not build your own assembler or linker.** Use `as` and `ld` (or `cc` as linker driver). The compiler produces assembly text. That's its job.

3. **Do not add optimization for v1.** Get correctness first. Optimization is a future `TransformMap` operating on the assembly or an IR. It can be added without modifying any existing code.

4. **Do not invent infrastructure.** The toolkit provides the tree walker, the transform engine, the multi-pass support, the context/state system. Write transform *functions*, not frameworks.

5. **Do not try to handle all of C11 in the grammar up front.** Start with the subset for M1 and expand as needed. Each new feature is more grammar rules and more transform entries. The architecture never changes.

6. **Do not add abstraction layers.** No adapters, wrappers, helpers, or "flexibility." Each grammar rule maps to one transform function. That's the architecture. It's enough.

7. **Do not skip a milestone.** Each one is designed so that the previous one's tests still pass. If you jump ahead, you'll build on an untested foundation and waste time debugging the wrong layer.

8. **Do not "fix the test."** If a test fails, the compiler has a bug. Fix the compiler. The test is the spec.

---

## When You Get Stuck

1. **Print the parse tree.** Add a `-parse-only` flag that prints the tree and exits. Most bugs are in the grammar, not the transforms.

2. **Print the AST.** The S-expression printer shows you exactly what the transforms produced. If the AST is wrong, the bug is in a transform function.

3. **Compare with GCC.** `gcc -S -O0 test.c` produces unoptimized assembly. Your output doesn't need to match, but comparing shows you what a correct implementation does.

4. **Bisect.** If a program fails, reduce it. Remove lines until you find the minimal failing case. The bug is in whatever you last removed.

5. **Read the toolkit source.** `parse/transform.go` is ~673 lines. It's well-written. When a transform behaves unexpectedly, reading the engine code is faster than guessing.

6. **Check the grammar first.** If a transform gets unexpected inputs, the grammar probably isn't producing the tree structure you think it is. Print the tree. Look at it.

---

## The Key Insight

C was designed as portable assembly. The language semantics map nearly 1:1 to hardware operations. An `int` is a register. A pointer is an address. `a + b` is an ADD instruction. `*p` is a load. `p->field` is a load at offset.

This means the entire compiler is shallow transforms. There is no deep translation. The grammar describes the syntax. The transforms describe the semantics. The emit pass describes the mapping to hardware. And that mapping is direct because C was designed for it to be direct.

Every pass is a `TransformMap`. The toolkit walks the tree. You fill in the functions.

Write the grammar. Define the types. Fill in the transforms. That's it. That's the whole compiler.
