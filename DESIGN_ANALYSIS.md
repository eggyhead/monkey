# Monkey Interpreter — Design Decisions & Alternative Approaches

This document analyzes each major component of the Monkey interpreter (chapters 1–3)
and explores the tradeoffs between the book's choices and alternatives.

---

## 1. Token Representation

### Book's Choice: `TokenType` as `string`

```go
type TokenType string
const PLUS = "+"
```

**Why the book does this:** Easy to debug — you can just print a token and it's readable.
No helper functions needed. Fast to implement.

**Tradeoff:** Comparing strings is slower than comparing integers. No compile-time
exhaustiveness checks. Typos in string constants won't be caught by the compiler.

### Alternative A: `iota`-based integer enum

```go
type TokenType int
const (
    ILLEGAL TokenType = iota
    EOF
    IDENT
    INT
    PLUS
    // ...
)
```

**Pros:** Faster comparisons. Uses less memory. Enables bitset operations. Go `switch`
can be checked for exhaustiveness with linters. Standard Go idiom.

**Cons:** Need a `String()` method for debugging. Slightly more boilerplate.

**When to prefer this:** Production interpreters, performance-sensitive scenarios.

### Alternative B: Token carries source position (line/col)

```go
type Token struct {
    Type    TokenType
    Literal string
    Line    int
    Column  int
}
```

**Pros:** Enables meaningful error messages like `"line 5, col 12: unexpected token"`.

**Cons:** More memory per token. Lexer needs to track position.

**When to prefer this:** Any language intended for real users.

---

## 2. Lexer Architecture

### Book's Choice: Hand-written, byte-at-a-time, single lookahead

```go
type Lexer struct {
    input        string
    position     int     // current char
    readPosition int     // next char
    ch           byte    // current byte (not rune!)
}
```

**Why:** Simple state machine. Easy to understand. No dependencies.

**Tradeoff:** Only handles ASCII (`byte`, not `rune`). Can't lex Unicode identifiers
or string literals with multi-byte characters.

### Alternative A: `rune`-based lexer (Unicode support)

Replace `ch byte` with `ch rune`, use `utf8.DecodeRuneInString()` instead of
indexing. Allows `let π = 3;`.

**Cost:** Slightly more complex character reading. Variable-width characters
mean you can't just index `input[position]`.

### Alternative B: `io.Reader`-based streaming lexer

Instead of loading the entire input into a string, read from an `io.Reader`.
Enables lexing files too large to fit in memory.

**Cost:** Need buffering logic. Can't easily "peek" backward. More complex.

### Alternative C: Scanner/lexer generator (e.g., `ragel`, `re2c`)

Generate the lexer from a state machine description. Produces very fast code.

**Cost:** External tool dependency. Generated code is hard to read/debug.
Loses the educational value.

### Key Design Question: Keyword identification

**Book's approach:** Lex all letter sequences as identifiers, then look up in a
`map[string]TokenType`. Simple and correct.

**Alternative:** Use a trie for keyword lookup — O(k) where k is keyword length
instead of O(k) for map hashing. Matters when keyword count is large. For Monkey's
7 keywords, the map is fine.

---

## 3. AST Design

### Book's Choice: Go interfaces with marker methods

```go
type Node interface { TokenLiteral() string; String() string }
type Statement interface { Node; statementNode() }
type Expression interface { Node; expressionNode() }
```

**Why:** Go-idiomatic. The marker methods (`statementNode()`, `expressionNode()`)
prevent accidental interface satisfaction. Compile-time type safety via interface
checks.

**Tradeoff:** Evaluator uses type switches, which bypass compile-time exhaustiveness.
Adding a new node type means hunting through all `switch` statements.

### Alternative A: Visitor pattern

```go
type Visitor interface {
    VisitLetStatement(*LetStatement) interface{}
    VisitReturnStatement(*ReturnStatement) interface{}
    VisitInfixExpression(*InfixExpression) interface{}
    // ...one method per node type
}

type Node interface {
    Accept(Visitor) interface{}
}
```

**Pros:** Adding a new *operation* (printer, optimizer, compiler) is easy — just
implement a new Visitor. Compile-time checks ensure you handle every node.

**Cons:** Adding a new *node type* requires touching every Visitor. More boilerplate.
Less idiomatic in Go than in Java/C++.

**When to prefer this:** When you have many operations over a stable set of node types
(e.g., formatter, type-checker, optimizer, code-generator all operating on the same AST).

### Alternative B: Sum types (tagged union)

Go doesn't natively support sum types, but you can simulate with:

```go
type Node struct {
    Type     NodeType
    // union-like fields
    IntVal   int64
    Left     *Node
    Right    *Node
    Op       string
    Name     string
    // ...
}
```

**Pros:** Cache-friendly (single allocation per node). No interface overhead.
Used in high-performance parsers (e.g., esbuild).

**Cons:** Waste memory on unused fields. Easy to access wrong field. Loses type safety.

### Key Design Question: "Everything is an expression"

The book makes `if/else` an expression (returns a value), and wraps bare expressions
in `ExpressionStatement`. This is a deliberate language design choice that simplifies
the evaluator — everything evaluates to something.

**Alternative:** Strict statement/expression separation (like C/Java). Simpler to
understand but limits composability (`let x = if (...) ...` wouldn't work).

---

## 4. Parser Strategy

### Book's Choice: Pratt Parsing (Top-Down Operator Precedence)

```go
prefixParseFns map[token.TokenType]prefixParseFn
infixParseFns  map[token.TokenType]infixParseFn
```

**Why:** Elegant handling of operator precedence. No grammar file needed. Easy to
extend — just register a new function. The core loop is ~15 lines.

**Tradeoff:** Unfamiliar to many programmers (less widely taught than recursive descent
or parser generators). Harder to derive from a formal grammar.

### Alternative A: Classic Recursive Descent

Each grammar rule becomes a function: `parseAddition()` calls `parseMultiplication()`
calls `parseUnary()` calls `parsePrimary()`.

```go
func (p *Parser) parseAddition() ast.Expression {
    left := p.parseMultiplication()
    for p.peekTokenIs(token.PLUS) || p.peekTokenIs(token.MINUS) {
        op := p.curToken
        p.nextToken()
        right := p.parseMultiplication()
        left = &ast.InfixExpression{Left: left, Operator: op.Literal, Right: right}
    }
    return left
}
```

**Pros:** Directly mirrors the grammar. Easy to understand if you know BNF.
Most taught approach in compiler courses.

**Cons:** Adding a new precedence level means inserting a new function in the chain.
Deep call stacks for many precedence levels. More code.

### Alternative B: Parser Generator (ANTLR, yacc, goyacc)

Write a formal grammar:

```
expression : expression '+' expression
           | expression '*' expression
           | INTEGER
           ;
```

**Pros:** Grammar is the single source of truth. Handles ambiguity resolution
declaratively. Can generate parsers in multiple languages.

**Cons:** External dependency. Generated code is opaque. Harder to customize error
messages. Loses educational value — you learn the tool, not parsing.

### Alternative C: Precedence Climbing

Similar to Pratt but driven by a loop with explicit precedence comparison rather than
function dispatch maps. Slightly less abstraction.

**Pros:** Very compact. No maps needed.

**Cons:** Less extensible for non-operator constructs (if, fn, etc.).

### Key Design Question: Error recovery

**Book's approach:** Accumulate errors in a `[]string` slice. No recovery — after an
error, parsing of that statement is abandoned.

**Alternative:** Synchronization-based recovery — on error, skip tokens until a
known synchronization point (`;`, `}`, keyword). Allows reporting multiple errors
per parse. Most production parsers do this.

---

## 5. Object System (Runtime Values)

### Book's Choice: Interface-based with concrete struct types

```go
type Object interface {
    Type() ObjectType
    Inspect() string
}

type Integer struct { Value int64 }
type Boolean struct { Value bool }
type Null struct {}
type ReturnValue struct { Value Object }
type Error struct { Message string }
type Function struct { Parameters, Body, Env }
```

**Why:** Clean separation. Each type is self-contained. Go-idiomatic.

**Tradeoff:** Every integer is heap-allocated (interface boxing). GC pressure.

### Alternative A: NaN-boxing

Pack all values into a single `float64`. Use NaN bit patterns to encode pointers,
booleans, nil, and small integers. Used by LuaJIT, JavaScriptCore, some Lisps.

**Pros:** No allocations for primitives. Values fit in a register. No GC pressure
for numbers/booleans.

**Cons:** Extremely tricky to implement. Platform-dependent. Hard to debug.

### Alternative B: Tagged union (discriminated union)

```go
type Object struct {
    Type  ObjectType
    IntVal   int64
    BoolVal  bool
    FnVal    *Function
    // ...
}
```

**Pros:** Single struct, no interface dispatch. Cache-friendly.

**Cons:** Wastes memory. No type safety on field access.

### Key Design Decision: Singleton TRUE/FALSE/NULL

The book reuses single instances:
```go
var TRUE  = &object.Boolean{Value: true}
var FALSE = &object.Boolean{Value: false}
var NULL  = &object.Null{}
```

This enables **pointer comparison** for boolean/null equality (`left == right`
compares pointers, not values). Clever optimization that avoids deep comparison.

---

## 6. Evaluation Strategy

### Book's Choice: Tree-Walking Interpreter

Recursively traverse the AST, evaluating each node directly.

**Why:** Simplest possible approach. ~200 lines of code. Easy to extend and debug.
Natural match for a recursive AST. Used by early Ruby (< 1.9), early JavaScript
engines, many Lisps.

**Tradeoff:** Slowest evaluation strategy. Each node requires function call overhead,
interface dispatch, and potential heap allocation. No optimization opportunities.

### Alternative A: Bytecode + Virtual Machine

Compile AST to bytecode instructions, then execute on a stack- or register-based VM.

```
// "1 + 2" compiles to:
PUSH 1
PUSH 2
ADD
```

**Pros:** 5–50× faster than tree-walking. Bytecode is compact (cache-friendly).
VM dispatch loop is tight. Enables optimizations (constant folding, etc.).

**Cons:** Much more complex. Need to design an instruction set, implement a compiler
pass, and build a VM. (This is covered in "Writing a Compiler in Go", the sequel book.)

### Alternative B: AST-Walking with Optimizations

Keep tree-walking but add optimizations:
- **Constant folding:** `1 + 2` → `3` at parse time
- **Inline caching:** Cache environment lookups for repeated identifier access
- **Tail-call optimization:** Reuse stack frame for tail-recursive calls

**Pros:** Incremental improvement. Keep simplicity while gaining speed.

**Cons:** Limited ceiling. Eventually you need bytecode for real performance.

### Key Design Decision: Errors as values

The book represents errors as `object.Error` — a regular object that propagates
through the evaluation. No Go `panic`/`recover`, no exceptions.

**Pros:** Simple control flow. Errors compose naturally. Easy to test.

**Cons:** Every evaluation step must check `isError()`. Forgetting a check means
errors silently disappear. An exception-based approach would propagate automatically.

---

## 7. Environment & Scoping

### Book's Choice: Linked-list of hash maps

```go
type Environment struct {
    store map[string]Object
    outer *Environment
}
```

**Why:** Natural model for lexical scoping. Function calls create a new environment
linked to the function's definition environment (closures). Simple to implement.

**Tradeoff:** O(n) lookup in worst case (n = scope depth). Hash map per scope
has overhead for small scopes.

### Alternative A: Flat array with compile-time indices

At parse/compile time, assign each variable a `(depth, index)` pair. At runtime,
look up via `envs[depth].slots[index]`.

**Pros:** O(1) lookup. No hashing. Cache-friendly.

**Cons:** Requires a resolution pass before evaluation. More complex implementation.
This is how many production interpreters work (including CPython's LOAD_FAST).

### Alternative B: Single global map (no scoping)

**Pros:** Trivial to implement.

**Cons:** No local variables. No closures. Not useful for real languages.

---

## Summary: Where Could You Go From Here?

| Component       | Book's Choice            | Most Common "Next Step"              |
|-----------------|--------------------------|--------------------------------------|
| Token types     | String constants         | iota-based int enum                  |
| Lexer           | byte-based, hand-written | rune-based for Unicode               |
| AST             | Interface + type switch  | Visitor pattern (if adding passes)   |
| Parser          | Pratt parsing            | Keep Pratt (it's excellent)          |
| Object system   | Interface per type       | Tagged union or NaN-boxing           |
| Evaluator       | Tree-walking             | Bytecode + VM (the sequel book)      |
| Scoping         | Linked environments      | Compile-time variable resolution     |
| Error reporting | Accumulated strings      | Source positions + structured errors  |

Each of these is a spectrum from "simplest correct thing" to "production-grade".
The book consistently picks the simplest correct thing, which is exactly right for
learning. The alternatives above show where you'd move each dial for performance,
usability, or extensibility.
