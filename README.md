# Monkey Interpreter

A tree-walking interpreter for the [Monkey programming language](https://monkeylang.org/), built in Go by following Thorsten Ball's *Writing an Interpreter in Go* (chapters 1–3).

See https://docs.google.com/document/d/110UmsQylotN2sh1FKxDoXX7jwXg-WeW_/edit for notes about project setup. 

## The Monkey Language

```
let fibonacci = fn(x) {
  if (x == 0) { return 0; }
  if (x == 1) { return 1; }
  fibonacci(x - 1) + fibonacci(x - 2);
};

fibonacci(10);
```

Monkey supports:

- **Integer arithmetic** — `+`, `-`, `*`, `/`
- **Boolean logic** — `==`, `!=`, `<`, `>`, `!`, `-` (prefix)
- **Variables** — `let x = 5;`
- **Functions** — first-class, with closures
- **Conditionals** — `if`/`else` expressions (they return values)
- **Return statements** — early exit from functions

## Project Structure

```
monkey/
├── token/       # Token types and keyword lookup
├── lexer/       # Lexical analysis (source → tokens)
├── ast/         # Abstract syntax tree node definitions
├── parser/      # Pratt parser (tokens → AST)
├── object/      # Runtime value types and environment
├── evaluator/   # Tree-walking evaluator (AST → result)
├── repl/        # Interactive read-eval-print loop
└── main.go      # Entry point
```

## Getting Started

**Requirements:** Go 1.24+

```sh
# Run the REPL
go run main.go

# Run all tests
go test ./...
```

The REPL evaluates expressions interactively:

```
>> let add = fn(a, b) { a + b; };
>> add(3, 4)
7
>> let x = 10;
>> if (x > 5) { true } else { false }
true
```

## How It Works

Source code flows through three stages:

1. **Lexer** — scans the input string character-by-character, producing tokens
2. **Parser** — consumes tokens using [Pratt parsing](https://en.wikipedia.org/wiki/Operator-precedence_parser#Pratt_parsing) to build an AST with correct operator precedence
3. **Evaluator** — recursively walks the AST, evaluating each node and returning Monkey objects

## Design Decisions

See [`DESIGN_ANALYSIS.md`](DESIGN_ANALYSIS.md) for a detailed breakdown of each component's implementation choices, alternative approaches, and tradeoffs — intended as a reference for future iterations on this codebase.

## Based On

[*Writing an Interpreter in Go*](https://interpreterbook.com/) by Thorsten Ball — this implementation covers chapters 1–3 (lexing, parsing, evaluation). Chapter 4 (strings, arrays, hashes, built-in functions) is not yet implemented.
