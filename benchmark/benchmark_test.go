package benchmark

import (
	"fmt"
	"monkey/ast"
	"monkey/evaluator"
	"monkey/lexer"
	"monkey/object"
	"monkey/optimizer"
	"monkey/parser"
	"monkey/resolver"
	"testing"
)

// evalSource runs the full interpreter pipeline (with integer caching, no folding).
func evalSource(source string) object.Object {
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()
	env := object.NewEnvironment()
	return evaluator.Eval(program, env)
}

// parseSource parses source into an AST (for benchmarks that separate parse from eval).
func parseSource(source string) *ast.Program {
	l := lexer.New(source)
	p := parser.New(l)
	return p.ParseProgram()
}

// parseAndOptimize parses and constant-folds (for benchmarks measuring eval-only).
func parseAndOptimize(source string) *ast.Program {
	program := parseSource(source)
	optimizer.FoldConstants(program)
	return program
}

// --- Benchmark programs ---
// Each targets a specific cost center from the research.

// fibSource: recursive fibonacci — the canonical interpreter benchmark.
// Why: each call creates a new Environment (map alloc), every + and - allocates
// a new object.Integer, and the two base-case comparisons (== 0, == 1) happen
// at every leaf. fib(35) makes ~29 million calls to Eval.
const fibSource = `
let fibonacci = fn(x) {
	if (x == 0) { return 0; }
	if (x == 1) { return 1; }
	fibonacci(x - 1) + fibonacci(x - 2);
};
fibonacci(%d);
`

// sumLoopSource: iterative summation via recursion (Monkey has no loops).
// Why: stresses variable lookup in shallow scopes and integer allocation
// from the accumulator addition. Less branching than fib, more arithmetic.
const sumLoopSource = `
let sum = fn(n, acc) {
	if (n == 0) { return acc; }
	sum(n - 1, acc + n);
};
sum(%d, 0);
`

// nestedArithSource: deep arithmetic without function calls.
// Why: isolates the cost of integer allocation and infix dispatch
// without environment creation overhead. Every subexpression allocates.
const nestedArithSource = `
let compute = fn(x) {
	((x + 1) * 2 - 3) + ((x * x + x) - (x / 2));
};
let result = fn(n) {
	if (n == 0) { return 0; }
	compute(n) + result(n - 1);
};
result(%d);
`

// constantHeavySource: lots of constant subexpressions.
// Why: after constant folding is implemented, this should show dramatic
// improvement since all the math can be resolved at parse time.
const constantHeavySource = `
let f = fn(x) {
	x + (2 + 3) * (10 - 4) + (100 / 5) + (7 * 8 - 6);
};
let loop = fn(n) {
	if (n == 0) { return 0; }
	f(n) + loop(n - 1);
};
loop(%d);
`

// --- Benchmarks ---

func BenchmarkFib20(b *testing.B) {
	program := parseSource(fmt.Sprintf(fibSource, 20))
	benchEvalOnly(b, program)
}

func BenchmarkFib35(b *testing.B) {
	program := parseSource(fmt.Sprintf(fibSource, 35))
	benchEvalOnly(b, program)
}

func BenchmarkSum5000(b *testing.B) {
	program := parseSource(fmt.Sprintf(sumLoopSource, 5000))
	benchEvalOnly(b, program)
}

func BenchmarkNestedArith500(b *testing.B) {
	program := parseSource(fmt.Sprintf(nestedArithSource, 500))
	benchEvalOnly(b, program)
}

func BenchmarkConstantHeavy1000(b *testing.B) {
	program := parseSource(fmt.Sprintf(constantHeavySource, 1000))
	benchEvalOnly(b, program)
}

// --- Optimized benchmarks (with constant folding) ---
// Parse and fold ONCE before the timer; benchmark only measures eval time.
// This reflects real usage: you parse once and potentially eval many times.

func benchEvalOnly(b *testing.B, program *ast.Program) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := object.NewEnvironment()
		evaluator.Eval(program, env)
	}
}

func BenchmarkFib20_Optimized(b *testing.B) {
	program := parseAndOptimize(fmt.Sprintf(fibSource, 20))
	benchEvalOnly(b, program)
}

func BenchmarkFib35_Optimized(b *testing.B) {
	program := parseAndOptimize(fmt.Sprintf(fibSource, 35))
	benchEvalOnly(b, program)
}

func BenchmarkSum5000_Optimized(b *testing.B) {
	program := parseAndOptimize(fmt.Sprintf(sumLoopSource, 5000))
	benchEvalOnly(b, program)
}

func BenchmarkNestedArith500_Optimized(b *testing.B) {
	program := parseAndOptimize(fmt.Sprintf(nestedArithSource, 500))
	benchEvalOnly(b, program)
}

func BenchmarkConstantHeavy1000_Optimized(b *testing.B) {
	program := parseAndOptimize(fmt.Sprintf(constantHeavySource, 1000))
	benchEvalOnly(b, program)
}

// --- Resolved benchmarks (variable resolution + integer cache) ---
// Parse and resolve ONCE; benchmark only measures eval with fast environments.

func benchEvalResolved(b *testing.B, source string) {
	program := parseSource(source)
	table := resolver.Resolve(program)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		evaluator.EvalResolved(program, table)
	}
}

func BenchmarkFib20_Resolved(b *testing.B) {
	benchEvalResolved(b, fmt.Sprintf(fibSource, 20))
}

func BenchmarkFib35_Resolved(b *testing.B) {
	benchEvalResolved(b, fmt.Sprintf(fibSource, 35))
}

func BenchmarkSum5000_Resolved(b *testing.B) {
	benchEvalResolved(b, fmt.Sprintf(sumLoopSource, 5000))
}

func BenchmarkNestedArith500_Resolved(b *testing.B) {
	benchEvalResolved(b, fmt.Sprintf(nestedArithSource, 500))
}

func BenchmarkConstantHeavy1000_Resolved(b *testing.B) {
	benchEvalResolved(b, fmt.Sprintf(constantHeavySource, 1000))
}

// --- All optimizations combined (fold + resolve + cache) ---

func benchEvalAllOptimized(b *testing.B, source string) {
	program := parseSource(source)
	optimizer.FoldConstants(program)
	table := resolver.Resolve(program)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		evaluator.EvalResolved(program, table)
	}
}

func BenchmarkFib20_AllOpt(b *testing.B) {
	benchEvalAllOptimized(b, fmt.Sprintf(fibSource, 20))
}

func BenchmarkFib35_AllOpt(b *testing.B) {
	benchEvalAllOptimized(b, fmt.Sprintf(fibSource, 35))
}

func BenchmarkSum5000_AllOpt(b *testing.B) {
	benchEvalAllOptimized(b, fmt.Sprintf(sumLoopSource, 5000))
}

func BenchmarkNestedArith500_AllOpt(b *testing.B) {
	benchEvalAllOptimized(b, fmt.Sprintf(nestedArithSource, 500))
}

func BenchmarkConstantHeavy1000_AllOpt(b *testing.B) {
	benchEvalAllOptimized(b, fmt.Sprintf(constantHeavySource, 1000))
}