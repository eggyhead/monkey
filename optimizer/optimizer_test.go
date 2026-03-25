package optimizer

import (
	"monkey/lexer"
	"monkey/parser"
	"testing"
)

func parseAndFold(input string) string {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	FoldConstants(program)
	return program.String()
}

func TestFoldIntegerArithmetic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Basic arithmetic
		{"2 + 3;", "5"},
		{"10 - 4;", "6"},
		{"3 * 7;", "21"},
		{"20 / 4;", "5"},

		// Nested constant expressions
		{"(2 + 3) * 4;", "20"},
		{"(10 - 4) + (3 * 2);", "12"},
		{"((1 + 2) * (3 + 4));", "21"},

		// Mixed: constant part folds, variable part stays
		{"x + (2 + 3);", "(x + 5)"},
		{"(1 + 2) + x;", "(3 + x)"},

		// Division by zero: should NOT fold (leave for runtime error)
		{"10 / 0;", "(10 / 0)"},
	}

	for _, tt := range tests {
		result := parseAndFold(tt.input)
		if result != tt.expected {
			t.Errorf("input=%q: expected=%q, got=%q", tt.input, tt.expected, result)
		}
	}
}

func TestFoldIntegerComparisons(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"3 < 5;", "true"},
		{"5 > 3;", "true"},
		{"3 > 5;", "false"},
		{"3 == 3;", "true"},
		{"3 != 3;", "false"},
		{"3 == 4;", "false"},
		{"3 != 4;", "true"},
	}

	for _, tt := range tests {
		result := parseAndFold(tt.input)
		if result != tt.expected {
			t.Errorf("input=%q: expected=%q, got=%q", tt.input, tt.expected, result)
		}
	}
}

func TestFoldPrefixExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"-5;", "-5"},       // Prefix minus on literal — folds to negative IntegerLiteral
		{"!true;", "false"},   // Bang on boolean
		{"!false;", "true"},   // Bang on boolean
		{"!!true;", "true"},   // Double bang
		{"-x;", "(-x)"},      // Variable — cannot fold
		{"!x;", "(!x)"},      // Variable — cannot fold
	}

	for _, tt := range tests {
		result := parseAndFold(tt.input)
		if result != tt.expected {
			t.Errorf("input=%q: expected=%q, got=%q", tt.input, tt.expected, result)
		}
	}
}

func TestFoldBooleanInfix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"true == true;", "true"},
		{"true == false;", "false"},
		{"true != false;", "true"},
		{"false != false;", "false"},
	}

	for _, tt := range tests {
		result := parseAndFold(tt.input)
		if result != tt.expected {
			t.Errorf("input=%q: expected=%q, got=%q", tt.input, tt.expected, result)
		}
	}
}

func TestFoldInsideFunctionBodies(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Constants inside function body should fold
		{"let f = fn(x) { x + (2 + 3); };", "let f = fn(x) (x + 5);"},
		// Constants in let value
		{"let x = 2 + 3;", "let x = 5;"},
		// Constants in return value
		{"return 2 + 3;", "return 5;"},
	}

	for _, tt := range tests {
		result := parseAndFold(tt.input)
		if result != tt.expected {
			t.Errorf("input=%q: expected=%q, got=%q", tt.input, tt.expected, result)
		}
	}
}

func TestFoldInsideIfExpression(t *testing.T) {
	input := "if (2 + 3) { 10 - 4; }"
	expected := "if5 6"

	result := parseAndFold(input)
	if result != expected {
		t.Errorf("input=%q: expected=%q, got=%q", input, expected, result)
	}
}

func TestFoldCallArguments(t *testing.T) {
	input := "add(2 + 3, 4 * 5);"
	expected := "add(5, 20)"

	result := parseAndFold(input)
	if result != expected {
		t.Errorf("input=%q: expected=%q, got=%q", input, expected, result)
	}
}

func TestFoldPreservesNonConstants(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Variables can't be folded
		{"x + y;", "(x + y)"},
		// Function calls can't be folded
		{"add(x, y);", "add(x, y)"},
		// If conditions with variables stay
		{"if (x) { 1; }", "ifx 1"},
	}

	for _, tt := range tests {
		result := parseAndFold(tt.input)
		if result != tt.expected {
			t.Errorf("input=%q: expected=%q, got=%q", tt.input, tt.expected, result)
		}
	}
}
