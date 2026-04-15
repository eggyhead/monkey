# Monkey Interpreter — Performance Optimization Results

This document records the implementation, measurement, and analysis of three
tree-walker optimizations applied to the Monkey interpreter. Each optimization
targets a specific cost center identified in the research ("Beyond the tree walk"):
heap allocation, redundant computation, and hash-map variable lookup.

**Platform:** Apple M1, Go 1.24.1, macOS (Darwin arm64)

---

## Baseline: What Makes the Tree-Walker Slow

Before optimization, the evaluator has three dominant costs:

1. **Per-operation heap allocation.** Every `&object.Integer{Value: n}` allocates
   24 bytes on the heap (16-byte interface + 8-byte int64). `fib(35)` triggers
   **278 million allocations** totaling 12.4 GB of churn through the GC.

2. **Hash-map variable lookup.** `Environment.Get("x")` hashes a string and
   probes a `map[string]Object` for every variable access. In nested scopes,
   this chains through multiple maps via `outer` pointers.

3. **Redundant constant evaluation.** Expressions like `(2 + 3) * (10 - 4)`
   are re-evaluated at runtime every time they're encountered, even though
   the result is always the same.

### Baseline Measurements

| Benchmark | Time | Allocs | Bytes |
|-----------|------|--------|-------|
| Fib(20) | 7.5 ms | 203,895 | 9.1 MB |
| Fib(35) | 10.5 s | 277,974,086 | 12.4 GB |
| Sum(5000) | 2.3 ms | 45,106 | 2.2 MB |
| NestedArith(500) | 396 μs | 12,153 | 438 KB |
| ConstantHeavy(1000) | 867 μs | 30,154 | 918 KB |

These baselines use the original `map[string]Object` environment and allocate a
new `object.Integer` for every arithmetic result.

---

## Optimization 1: Integer Caching

### Concept

Pre-allocate `object.Integer` instances for values 0–255 in a global array.
When the evaluator would create `&object.Integer{Value: n}`, it checks if `n`
is in range and returns the cached pointer instead. This eliminates the heap
allocation entirely for small integers.

### Design Decisions

- **Range 0–255 (not negative).** Monkey's parser produces non-negative integer
  literals; negative numbers are prefix expressions (`-5` = negate the literal 5).
  The majority of intermediate values in fibonacci (0, 1, 2, 3, ...) and loop
  counters fall in this range. Cost: 256 × 24 bytes ≈ 6 KB of permanent memory.

- **Returned as `*object.Integer`, not `object.Object`.** This preserves type
  information and avoids an extra interface boxing step.

- **Boolean singletons already existed.** The original code already cached `TRUE`,
  `FALSE`, and `NULL` as package-level singletons. Integer caching extends the
  same principle.

### What Changed

- `object/object.go`: Added `integerCache [256]*Integer` array, `init()` to
  populate it, and `CachedInteger(val int64) *Integer` function.
- `evaluator/evaluator.go`: Replaced 6 instances of `&object.Integer{Value: ...}`
  with `object.CachedInteger(...)` in `Eval(*ast.IntegerLiteral)`,
  `evalMinusPrefixOperatorExpression`, and all 4 arithmetic cases in
  `evalIntegerInfixExpression`.

### Results (Integer Caching Only, vs Original Baseline)

| Benchmark | Baseline | +Cache | Alloc Δ | Speed Δ |
|-----------|----------|--------|---------|---------|
| Fib(20) | 7.5 ms / 203,895 | 6.5 ms / 98,655 | **−52%** | **1.15×** |
| Fib(35) | 10.5 s / 278M | 9.0 s / 134M | **−52%** | **1.17×** |
| Sum(5000) | 2.3 ms / 45,106 | 2.3 ms / 34,848 | **−23%** | ~1.0× |
| NestedArith(500) | 396 μs / 12,153 | 361 μs / 7,821 | **−36%** | **1.10×** |
| ConstantHeavy(1000) | 867 μs / 30,154 | 697 μs / 12,309 | **−59%** | **1.24×** |

### Analysis

Allocation counts dropped 23–59% across all benchmarks. The time improvement
(10–24%) is less dramatic than the allocation reduction because the remaining
cost is dominated by environment creation (map allocation per function call)
and Eval dispatch overhead, not just integer allocation.

The constant-heavy benchmark benefited most (59% fewer allocs) because its
inner function `x + 30 + 20 + 50` produces many small intermediate results
that all hit the cache. Fibonacci benefited from caching 0 and 1 (the base
cases checked millions of times).

**Tradeoff:** The cache costs 6 KB of permanent memory and adds one branch
per integer creation (`if val >= 0 && val <= 255`). The branch is essentially
free on modern CPUs due to branch prediction (the hot path is almost always
"in range" for small-number programs).

---

## Optimization 2: Constant Folding

### Concept

After parsing, walk the AST and replace constant subexpressions with their
computed results. `(2 + 3) * (10 - 4)` becomes `IntegerLiteral{Value: 30}`.
This eliminates all runtime dispatch, evaluation, and allocation for those nodes.

### Design Decisions

- **Separate optimizer pass** (`optimizer/` package) rather than inlining into
  the parser. This keeps concerns separated and makes the optimization toggleable
  for benchmarking.

- **Bottom-up folding.** Recurse into children first, then try folding the
  parent. This handles nested constants like `(2 + 3) * (10 - 4)` in a single
  pass — the children fold to `5` and `6`, then the multiplication folds to `30`.

- **Division by zero left unfolded.** `10 / 0` is left as-is so the evaluator
  produces its normal runtime error. Folding it would require the optimizer to
  have its own error reporting path.

- **Only pure operations folded.** Integer arithmetic (+, -, *, /), integer
  comparisons (<, >, ==, !=), boolean prefix (!), boolean infix (==, !=).
  Nothing involving identifiers, function calls, or side effects.

- **Functions are folded internally.** Constants inside `fn(x) { x + (2 + 3) }`
  are folded to `fn(x) { x + 5 }`, so every call to the function benefits
  from the optimization.

### What Changed

- New package `optimizer/optimizer.go` (190 LOC): `FoldConstants(program)`,
  recursive `foldExpression`, `tryFoldInfix`, `tryFoldPrefix`.
- `optimizer/optimizer_test.go` (155 LOC): Tests for integer arithmetic,
  comparisons, prefix folding, boolean infix, function bodies, if-expressions,
  call arguments, and preservation of non-constants.

### Results (Constant Folding + Integer Cache, vs Cache Only)

| Benchmark | Cache Only | +Folding | Speed Δ |
|-----------|-----------|----------|---------|
| Fib(20) | 6.6 ms | 6.6 ms | **1.00×** |
| Sum(5000) | 2.3 ms | 2.3 ms | **1.00×** |
| NestedArith(500) | 361 μs | 361 μs | **1.00×** |
| ConstantHeavy(1000) | 689 μs | 613 μs | **1.12×** |

### Analysis

Constant folding only helps the constant-heavy benchmark (12% speedup). For
fibonacci and sum, there are essentially no foldable constants — every
arithmetic operation involves the variable `x`. The `ConstantHeavy` function
`x + (2 + 3) * (10 - 4) + (100 / 5) + (7 * 8 - 6)` folds three constant
subtrees, eliminating 8 Eval dispatches per call.

**This matches the research prediction:** constant folding is moderate for
constant-heavy code and zero for variable-heavy recursive code. It's a
"free" optimization (runs once at parse time, no runtime cost) but its
impact depends entirely on program structure.

**Tradeoff:** The optimizer adds ~190 LOC and a one-time O(n) pass over
the AST. For a REPL (where you parse once and eval once), the parse-time
cost is negligible. For a hypothetical "run this file 1000 times" scenario,
the amortized cost approaches zero.

---

## Optimization 3: Variable Resolution (Flat-Array Environment)

### Concept

Before evaluation, walk the AST and assign each variable a numeric
`(depth, index)` pair. `depth` = how many enclosing scopes to traverse,
`index` = which slot in that scope's array. Replace `map[string]Object`
environments with `[]Object` arrays, turning every variable access from
a hash-map probe into a direct array index.

### Design Decisions

- **Side-table resolution** (map of AST node → ResolvedVar) rather than
  modifying AST structs. This preserves the original AST design and avoids
  breaking the parser. The evaluator checks the side-table; if a node isn't
  resolved (shouldn't happen), it falls back to an error.

- **Separate `EvalResolved` function** rather than modifying `Eval`. This
  keeps the original evaluator intact for comparison and avoids adding
  branching overhead to every eval call.

- **`ResolvedFunction` object type.** Functions created during resolved
  evaluation capture a `*FastEnvironment` and the resolution table, forming
  a closure over the fast environment chain.

- **Pre-define `let` names before resolving values.** This supports recursive
  functions (`let fib = fn(x) { fib(x-1) }`). The alternative (define after
  value) would fail to resolve the self-reference. This matches JavaScript's
  `let` hoisting behavior in practice.

- **Scope size computed from `let` count.** Each scope's array size is the
  number of `let` statements + function parameters. This is computed by a
  simple count at resolution time.

### What Changed

- New package `resolver/resolver.go` (165 LOC): Scope stack, `Resolve(program)`,
  walks AST assigning `(depth, index)` pairs.
- `resolver/resolver_test.go` (140 LOC): Tests for let bindings, multiple
  bindings, function params, closure depth, recursive functions.
- `object/fast_environment.go` (45 LOC): `FastEnvironment` with `[]Object`
  slots, `GetByIndex(depth, index)`, `SetByIndex(depth, index, val)`.
- `object/object.go`: Added `ResolvedFunction` type (captures `FastEnvironment`).
- `evaluator/eval_resolved.go` (175 LOC): Complete `EvalResolved` implementation
  using fast environments.
- `evaluator/eval_resolved_test.go` (145 LOC): Integration tests for integers,
  let statements, functions, fibonacci, closures, booleans, if/else.

### Results (Resolution + Integer Cache, vs Cache Only)

| Benchmark | Cache Only | +Resolved | Alloc Δ | Bytes Δ | Speed Δ |
|-----------|-----------|-----------|---------|---------|---------|
| Fib(20) | 6.6 ms / 98,547 | 4.8 ms / 76,655 | **−22%** | **−81%** | **1.40×** |
| Fib(35) | 9.0 s / 134M | 6.4 s / 105M | **−22%** | **−81%** | **1.41×** |
| Sum(5000) | 2.3 ms / 34,755 | 1.5 ms / 29,753 | **−14%** | **−69%** | **1.52×** |
| NestedArith(500) | 350 μs / 7,681 | 270 μs / 6,679 | **−13%** | **−76%** | **1.30×** |
| ConstantHeavy(1000) | 697 μs / 12,168 | 505 μs / 10,166 | **−16%** | **−79%** | **1.38×** |

### Analysis

This is the highest-impact optimization by far. Every benchmark improved
30–52% in speed and 69–81% in memory usage. The memory reduction is even
more dramatic than the time reduction because `[]Object` slices are vastly
more compact than `map[string]Object` hash tables. A Go map with 2 entries
costs ~200+ bytes of overhead (hash buckets, metadata). A 2-element `[]Object`
slice costs 16 bytes (slice header) + 16 bytes (data) = 32 bytes.

The research predicted ~25% improvement from this technique. We measured
30–52%, likely because:
1. The Go map overhead is proportionally larger for Monkey's tiny scopes
   (1–3 variables per scope).
2. The `FastEnvironment` eliminates string hashing entirely, which is
   expensive for short strings.
3. The array-based approach has better cache locality — slots are contiguous.

**Tradeoff:** This optimization requires a resolution pass (O(n) in AST
size, ~165 LOC) and a parallel evaluator implementation (~175 LOC). The
resolution table uses memory proportional to the number of variable
references in the AST. For a typical Monkey program, this is negligible.

---

## Combined: All Optimizations Together

### Results (All Optimizations vs Original Baseline)

| Benchmark | Original Baseline | All Optimized | Total Speed Δ | Total Bytes Δ |
|-----------|-------------------|---------------|---------------|---------------|
| Fib(20) | 7.5 ms / 203,895 allocs / 9.1 MB | 4.7 ms / 76,655 allocs / 1.6 MB | **1.60×** | **−82%** |
| Fib(35) | 10.5 s / 278M allocs / 12.4 GB | 6.5 s / 105M allocs / 2.2 GB | **1.62×** | **−83%** |
| Sum(5000) | 2.3 ms / 45,106 allocs / 2.2 MB | 1.6 ms / 29,753 allocs / 638 KB | **1.44×** | **−71%** |
| NestedArith(500) | 396 μs / 12,153 allocs / 438 KB | 280 μs / 6,679 allocs / 94 KB | **1.41×** | **−79%** |
| ConstantHeavy(1000) | 867 μs / 30,154 allocs / 918 KB | 430 μs / 10,166 allocs / 162 KB | **2.02×** | **−82%** |

### Individual Contribution Breakdown (Fib35)

| Optimization | Time | Δ from Previous | Cumulative Speedup |
|-------------|------|------------------|--------------------|
| **Baseline** (original) | 10.5 s | — | 1.00× |
| **+ Integer Cache** | 9.0 s | −14% | 1.17× |
| **+ Constant Folding** | ~9.0 s | ~0% | 1.17× |
| **+ Variable Resolution** | 6.4 s | −29% | 1.64× |
| **All Combined** | 6.5 s | — | 1.62× |

### Key Insights

1. **Variable resolution dominates.** It accounts for roughly 2/3 of the
   total speedup because it attacks the most frequent operation (variable
   lookup) AND reduces memory allocation simultaneously.

2. **Integer caching is the best effort-to-reward ratio.** It's ~20 lines
   of code for a 15–24% speedup and 50% allocation reduction. Excellent
   for a first optimization.

3. **Constant folding is situational.** Its benefit depends entirely on
   program structure. For computation-heavy code with few constants (like
   fibonacci), it does nothing. For configuration or formula-heavy code,
   it can be meaningful.

4. **Memory reduction exceeds time reduction.** We achieved 80%+ memory
   reduction but only 40–60% time reduction. This is because Go's allocator
   and GC are highly optimized — they handle the allocation volume gracefully,
   so eliminating allocations saves memory but only moderately reduces time.
   In a language with a less efficient allocator, the time savings would be
   larger.

5. **We're still in tree-walker territory.** Even with all optimizations,
   fib(35) takes 6.5 seconds — compared to ~5s for Ball's bytecode VM and
   ~0.05s for native Go. The remaining cost is in AST pointer-chasing
   (cache-unfriendly tree traversal) and Eval dispatch overhead (type switch
   on every node). These are fundamental to the tree-walking architecture
   and can only be eliminated by compiling to bytecode.

---

## Architecture of Changes

```
Source → Lexer → Parser → [Optimizer] → [Resolver] → Evaluator → Result
                              ↓              ↓              ↓
                        Fold constants   Build Table    Use CachedInteger
                        (AST → AST)    (AST → Table)   Use FastEnvironment
```

### New Packages

| Package | Files | LOC | Purpose |
|---------|-------|-----|---------|
| `optimizer/` | 2 | ~345 | Constant folding pass |
| `resolver/` | 2 | ~305 | Variable resolution pass |
| `benchmark/` | 1 | ~200 | Go benchmarks for all variants |

### Modified Files

| File | Changes |
|------|---------|
| `object/object.go` | +Integer cache, +`CachedInteger()`, +`ResolvedFunction` |
| `object/fast_environment.go` | New: array-backed environment |
| `evaluator/evaluator.go` | `CachedInteger()` calls (6 replacements) |
| `evaluator/eval_resolved.go` | New: resolved evaluator using fast environments |

---

## What's Left on the Table

These optimizations recover ~1.6× of the ~3–5× gap between tree-walking
and bytecode. The remaining gap comes from:

1. **AST pointer-chasing** — each Eval call follows a pointer to a heap-
   allocated node potentially far away in memory. Bytecode VMs use a
   contiguous byte array that streams through CPU cache.

2. **Type-switch dispatch** — every Eval call performs a type switch on the
   node. Bytecode VMs use a single `switch` on an opcode byte, which CPUs
   can predict and pipeline more efficiently.

3. **Function call overhead** — every Monkey function call allocates a new
   environment (even with fast arrays). Bytecode VMs reuse a fixed-size
   call stack.

4. **No tail-call optimization** — recursive Monkey functions like `sum`
   build deep call stacks. TCO could convert tail-recursive functions to
   loops.

These would require moving to bytecode compilation (Ball's second book)
or implementing a Truffle-style self-optimizing AST (impractical without
a JIT).
