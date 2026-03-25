package resolver

import (
	"monkey/ast"
	"monkey/lexer"
	"monkey/parser"
	"testing"
)

func parseProgram(input string) *ast.Program {
	l := lexer.New(input)
	p := parser.New(l)
	return p.ParseProgram()
}

func TestResolveLetBinding(t *testing.T) {
	program := parseProgram("let x = 5; x;")
	table := Resolve(program)

	// x is defined at global scope, slot 0
	// The reference to x in the ExpressionStatement should resolve to (0, 0)
	exprStmt := program.Statements[1].(*ast.ExpressionStatement)
	ident := exprStmt.Expression.(*ast.Identifier)

	rv, ok := table[ident]
	if !ok {
		t.Fatal("expected identifier 'x' to be resolved")
	}
	if rv.Depth != 0 || rv.Index != 0 {
		t.Errorf("expected (0, 0), got (%d, %d)", rv.Depth, rv.Index)
	}
}

func TestResolveMultipleBindings(t *testing.T) {
	program := parseProgram("let x = 5; let y = 10; y;")
	table := Resolve(program)

	// y is slot 1 at global scope
	exprStmt := program.Statements[2].(*ast.ExpressionStatement)
	ident := exprStmt.Expression.(*ast.Identifier)

	rv, ok := table[ident]
	if !ok {
		t.Fatal("expected identifier 'y' to be resolved")
	}
	if rv.Depth != 0 || rv.Index != 1 {
		t.Errorf("expected (0, 1), got (%d, %d)", rv.Depth, rv.Index)
	}
}

func TestResolveFunctionParams(t *testing.T) {
	program := parseProgram(`
		let add = fn(a, b) { a + b; };
	`)
	table := Resolve(program)

	// Inside the function, a=slot0, b=slot1
	letStmt := program.Statements[0].(*ast.LetStatement)
	fnLit := letStmt.Value.(*ast.FunctionLiteral)
	body := fnLit.Body.Statements[0].(*ast.ExpressionStatement)
	infix := body.Expression.(*ast.InfixExpression)

	aIdent := infix.Left.(*ast.Identifier)
	bIdent := infix.Right.(*ast.Identifier)

	aRv, ok := table[aIdent]
	if !ok {
		t.Fatal("expected 'a' to be resolved")
	}
	if aRv.Depth != 0 || aRv.Index != 0 {
		t.Errorf("'a' expected (0, 0), got (%d, %d)", aRv.Depth, aRv.Index)
	}

	bRv, ok := table[bIdent]
	if !ok {
		t.Fatal("expected 'b' to be resolved")
	}
	if bRv.Depth != 0 || bRv.Index != 1 {
		t.Errorf("'b' expected (0, 1), got (%d, %d)", bRv.Depth, bRv.Index)
	}
}

func TestResolveClosureDepth(t *testing.T) {
	program := parseProgram(`
		let x = 10;
		let f = fn() { x; };
	`)
	table := Resolve(program)

	// x is at global scope (depth 1 from inside function body)
	letStmt := program.Statements[1].(*ast.LetStatement)
	fnLit := letStmt.Value.(*ast.FunctionLiteral)
	body := fnLit.Body.Statements[0].(*ast.ExpressionStatement)
	xIdent := body.Expression.(*ast.Identifier)

	rv, ok := table[xIdent]
	if !ok {
		t.Fatal("expected 'x' to be resolved")
	}
	if rv.Depth != 1 || rv.Index != 0 {
		t.Errorf("'x' in closure expected (1, 0), got (%d, %d)", rv.Depth, rv.Index)
	}
}

func TestResolveRecursiveFunction(t *testing.T) {
	// This tests the fibonacci pattern — the function references itself
	// from the enclosing scope
	program := parseProgram(`
		let fib = fn(x) {
			if (x == 0) { return 0; }
			if (x == 1) { return 1; }
			fib(x - 1) + fib(x - 2);
		};
		fib(10);
	`)
	table := Resolve(program)

	// Verify it resolves without error and has entries
	if len(table) == 0 {
		t.Fatal("expected non-empty resolution table")
	}

	// The 'fib' reference inside the function body should resolve at depth 1
	letStmt := program.Statements[0].(*ast.LetStatement)
	fnLit := letStmt.Value.(*ast.FunctionLiteral)
	// Third statement in body: fib(x-1) + fib(x-2)
	exprStmt := fnLit.Body.Statements[2].(*ast.ExpressionStatement)
	infix := exprStmt.Expression.(*ast.InfixExpression)
	call1 := infix.Left.(*ast.CallExpression)
	fibIdent := call1.Function.(*ast.Identifier)

	rv, ok := table[fibIdent]
	if !ok {
		t.Fatal("expected 'fib' to be resolved inside its body")
	}
	if rv.Depth != 1 {
		t.Errorf("'fib' inside body expected depth 1, got %d", rv.Depth)
	}
}
