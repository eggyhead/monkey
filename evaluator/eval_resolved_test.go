package evaluator

import (
	"monkey/lexer"
	"monkey/object"
	"monkey/parser"
	"monkey/resolver"
	"testing"
)

func testResolvedEval(input string) object.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	table := resolver.Resolve(program)
	return EvalResolved(program, table)
}

// TestResolvedEvalIntegerExpression verifies basic integer arithmetic
// works with array-backed environments.
func TestResolvedEvalIntegerExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"5", 5},
		{"10", 10},
		{"-5", -5},
		{"5 + 5 + 5 + 5 - 10", 10},
		{"2 * 2 * 2 * 2 * 2", 32},
		{"50 / 2 * 2 + 10", 60},
		{"(5 + 10 * 2 + 15 / 3) * 2 + -10", 50},
	}

	for _, tt := range tests {
		evaluated := testResolvedEval(tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

// TestResolvedEvalLetStatements verifies variable binding with fast env.
func TestResolvedEvalLetStatements(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"let a = 5; a;", 5},
		{"let a = 5 * 5; a;", 25},
		{"let a = 5; let b = a; b;", 5},
		{"let a = 5; let b = a; let c = a + b + 5; c;", 15},
	}

	for _, tt := range tests {
		evaluated := testResolvedEval(tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

// TestResolvedEvalFunctionApplication verifies functions with fast env.
func TestResolvedEvalFunctionApplication(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"let identity = fn(x) { x; }; identity(5);", 5},
		{"let identity = fn(x) { return x; }; identity(5);", 5},
		{"let double = fn(x) { x * 2; }; double(5);", 10},
		{"let add = fn(x, y) { x + y; }; add(5, 5);", 10},
		{"let add = fn(x, y) { x + y; }; add(5 + 5, add(5, 5));", 20},
		{"fn(x) { x; }(5)", 5},
	}

	for _, tt := range tests {
		evaluated := testResolvedEval(tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

// TestResolvedEvalFibonacci verifies recursive fibonacci works correctly.
func TestResolvedEvalFibonacci(t *testing.T) {
	input := `
let fibonacci = fn(x) {
	if (x == 0) { return 0; }
	if (x == 1) { return 1; }
	fibonacci(x - 1) + fibonacci(x - 2);
};
fibonacci(10);
`
	evaluated := testResolvedEval(input)
	testIntegerObject(t, evaluated, 55)
}

// TestResolvedEvalClosures verifies closure variable capture.
func TestResolvedEvalClosures(t *testing.T) {
	input := `
let newAdder = fn(x) {
	fn(y) { x + y; };
};
let addTwo = newAdder(2);
addTwo(3);
`
	evaluated := testResolvedEval(input)
	testIntegerObject(t, evaluated, 5)
}

// TestResolvedEvalBooleanExpression verifies boolean operations.
func TestResolvedEvalBooleanExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"1 < 2", true},
		{"1 > 2", false},
		{"1 == 1", true},
		{"1 != 1", false},
		{"true == true", true},
		{"false == false", true},
		{"true != false", true},
	}

	for _, tt := range tests {
		evaluated := testResolvedEval(tt.input)
		testBooleanObject(t, evaluated, tt.expected)
	}
}

// TestResolvedEvalIfElse verifies conditional expressions.
func TestResolvedEvalIfElseExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"if (true) { 10 }", 10},
		{"if (false) { 10 }", nil},
		{"if (1 < 2) { 10 }", 10},
		{"if (1 > 2) { 10 }", nil},
		{"if (1 > 2) { 10 } else { 20 }", 20},
		{"if (1 < 2) { 10 } else { 20 }", 10},
	}

	for _, tt := range tests {
		evaluated := testResolvedEval(tt.input)
		integer, ok := tt.expected.(int)
		if ok {
			testIntegerObject(t, evaluated, int64(integer))
		} else {
			testNullObject(t, evaluated)
		}
	}
}


