# Plan: Implement Chapter 4.4–4.6 (Arrays, Hashes, puts)

## Problem
Complete the remaining sections of Chapter 4: Arrays (4.4), Hashes (4.5), and the `puts` builtin (4.6).

## What's Already Done
- String literals (lexer, parser, AST, evaluator) ✅
- String concatenation (`+` operator) ✅
- `BuiltinFunction` type, `Builtin` struct in object.go ✅
- `len` builtin (strings only) ✅
- `TestBuiltinFunctions` test ✅
- Evaluator wired to check builtins in `evalIdentifier` ✅
- `applyFunction` handles `*object.Builtin` ✅

## Approach
Follow the book's incremental, test-driven approach. Each todo is a self-contained unit: add tests first, then implementation, verify tests pass before moving on.

---

## Todos

### 1. arrays-lexer — Add `[` and `]` tokens to lexer
- Add `LBRACKET = "["` and `RBRACKET = "]"` to `token/token.go`
- Add `case '[':` and `case ']':` to `lexer/lexer.go` NextToken
- Extend `TestNextToken` input with `[1, 2];` and add expected tokens
- Run `go test ./lexer`

### 2. arrays-parser-literal — Parse array literals
- Add `ArrayLiteral` AST node to `ast/ast.go` (Token, Elements []Expression)
- Refactor `parseCallArguments` → `parseExpressionList(end token.TokenType)` in parser
- Update `parseCallExpression` to use `parseExpressionList(token.RPAREN)`
- Register `token.LBRACKET` → `parseArrayLiteral` as prefix
- Add `TestParsingArrayLiterals` test
- Run `go test ./parser`

### 3. arrays-parser-index — Parse index operator expressions
- Add `IndexExpression` AST node to `ast/ast.go` (Token, Left, Index)
- Add `INDEX` as highest precedence constant in parser
- Add `token.LBRACKET: INDEX` to precedences map
- Register `token.LBRACKET` → `parseIndexExpression` as infix
- Add `TestParsingIndexExpressions` test
- Add index operator precedence cases to `TestOperatorPrecedenceParsing`
- Run `go test ./parser`

### 4. arrays-evaluator — Evaluate array literals and index expressions
- Add `ARRAY_OBJ` constant and `Array` struct to `object/object.go`
- Add `*ast.ArrayLiteral` case in `Eval` (reuse `evalExpressions`)
- Add `*ast.IndexExpression` case in `Eval`
- Add `evalIndexExpression` and `evalArrayIndexExpression` functions
- Add `TestArrayLiterals` and `TestArrayIndexExpressions` tests
- Run `go test ./evaluator`

### 5. arrays-builtins — Add array built-in functions
- Extend `len` to support `*object.Array`
- Add `first`, `last`, `rest`, `push` builtins to `evaluator/builtins.go`
- Add test cases to `TestBuiltinFunctions` for all array builtins
- Run `go test ./evaluator`

### 6. hashes-lexer — Add `:` token to lexer
- Add `COLON = ":"` to `token/token.go`
- Add `case ':':` to lexer NextToken
- Extend `TestNextToken` input with `{"foo": "bar"}` and expected tokens
- Run `go test ./lexer`

### 7. hashes-parser — Parse hash literals
- Add `HashLiteral` AST node to `ast/ast.go` (Token, Pairs map[Expression]Expression)
- Register `token.LBRACE` → `parseHashLiteral` as prefix
- Add `TestParsingHashLiteralsStringKeys`, `TestParsingEmptyHashLiteral`, `TestParsingHashLiteralsWithExpressions` tests
- Run `go test ./parser`

### 8. hashes-object — Implement HashKey and Hashable
- Add `hash/fnv` import to object.go
- Add `HashKey` struct (Type ObjectType, Value uint64)
- Implement `HashKey()` on `*Boolean`, `*Integer`, `*String`
- Add `Hashable` interface
- Add `TestStringHashKey` (and optionally integer/boolean) tests in `object/object_test.go`
- Run `go test ./object`

### 9. hashes-evaluator — Evaluate hash literals and index expressions
- Add `HASH_OBJ` constant, `HashPair` and `Hash` structs to object.go
- Add `*ast.HashLiteral` case in `Eval` → `evalHashLiteral`
- Add `evalHashIndexExpression` and wire into `evalIndexExpression`
- Add `TestHashLiterals` and `TestHashIndexExpressions` tests
- Add `{"name": "Monkey"}[fn(x) { x }]` → "unusable as hash key: FUNCTION" to `TestErrorHandling`
- Run `go test ./evaluator`

### 10. puts-builtin — Add `puts` built-in function
- Add `puts` to builtins map: prints each arg via `fmt.Println(arg.Inspect())`, returns `NULL`
- Add `fmt` import to builtins.go
- Verify in REPL
- Run `go test ./...` (final full suite)

## Notes
- The book refactors `parseCallArguments` into `parseExpressionList` — this is reused for both arrays and function calls.
- Arrays are immutable in Monkey — `push` returns a new array, doesn't mutate.
- Hash keys use FNV-1a hashing for strings; hash collisions are not handled (noted as exercise).
- `puts` returns NULL, so the REPL will print `null` after the output.
