// Package optimizer performs AST-level optimizations before evaluation.
//
// Currently implements constant folding: expressions made entirely of literals
// are evaluated at compile time and replaced with a single literal node.
// This eliminates runtime computation for constant subexpressions like (2+3)*4.
package optimizer

import (
	"fmt"
	"monkey/ast"
	"monkey/token"
)

// FoldConstants walks the AST bottom-up and replaces constant subexpressions
// with their computed literal values. It modifies the AST in place.
func FoldConstants(program *ast.Program) *ast.Program {
	for i, stmt := range program.Statements {
		program.Statements[i] = foldStatement(stmt)
	}
	return program
}

func foldStatement(stmt ast.Statement) ast.Statement {
	switch s := stmt.(type) {
	case *ast.LetStatement:
		s.Value = foldExpression(s.Value)
		return s
	case *ast.ReturnStatement:
		s.ReturnValue = foldExpression(s.ReturnValue)
		return s
	case *ast.ExpressionStatement:
		s.Expression = foldExpression(s.Expression)
		return s
	case *ast.BlockStatement:
		return foldBlock(s)
	default:
		return stmt
	}
}

func foldBlock(block *ast.BlockStatement) *ast.BlockStatement {
	for i, stmt := range block.Statements {
		block.Statements[i] = foldStatement(stmt)
	}
	return block
}

// foldExpression is the core: recurse into children first (bottom-up),
// then try to fold the current node if all children are now literals.
func foldExpression(expr ast.Expression) ast.Expression {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *ast.InfixExpression:
		e.Left = foldExpression(e.Left)
		e.Right = foldExpression(e.Right)
		return tryFoldInfix(e)

	case *ast.PrefixExpression:
		e.Right = foldExpression(e.Right)
		return tryFoldPrefix(e)

	case *ast.IfExpression:
		// Fold subexpressions inside if/else, but don't eliminate branches —
		// the condition might depend on runtime values even if its
		// subexpressions are partially constant.
		e.Condition = foldExpression(e.Condition)
		if e.Consequence != nil {
			foldBlock(e.Consequence)
		}
		if e.Alternative != nil {
			foldBlock(e.Alternative)
		}
		return e

	case *ast.FunctionLiteral:
		// Fold constants inside function bodies. This runs once at parse time,
		// so all calls to this function benefit.
		if e.Body != nil {
			foldBlock(e.Body)
		}
		return e

	case *ast.CallExpression:
		e.Function = foldExpression(e.Function)
		for i, arg := range e.Arguments {
			e.Arguments[i] = foldExpression(arg)
		}
		return e

	default:
		// IntegerLiteral, Boolean, Identifier — already leaves, nothing to fold.
		return expr
	}
}

// tryFoldInfix attempts to compute an infix expression where both operands
// are literals. Returns the folded literal on success, or the original node
// if either operand isn't a literal.
func tryFoldInfix(expr *ast.InfixExpression) ast.Expression {
	leftInt, leftIsInt := expr.Left.(*ast.IntegerLiteral)
	rightInt, rightIsInt := expr.Right.(*ast.IntegerLiteral)

	// Integer OP Integer → Integer or Boolean
	if leftIsInt && rightIsInt {
		if folded := foldIntegerInfix(expr.Operator, leftInt.Value, rightInt.Value); folded != nil {
			return folded
		}
		return expr
	}

	leftBool, leftIsBool := expr.Left.(*ast.Boolean)
	rightBool, rightIsBool := expr.Right.(*ast.Boolean)

	// Boolean == Boolean, Boolean != Boolean → Boolean
	if leftIsBool && rightIsBool {
		switch expr.Operator {
		case "==":
			return makeBoolNode(leftBool.Value == rightBool.Value)
		case "!=":
			return makeBoolNode(leftBool.Value != rightBool.Value)
		}
	}

	return expr
}

func foldIntegerInfix(op string, left, right int64) ast.Expression {
	switch op {
	case "+":
		return makeIntNode(left + right)
	case "-":
		return makeIntNode(left - right)
	case "*":
		return makeIntNode(left * right)
	case "/":
		if right == 0 {
			// Leave division by zero for the evaluator to report as a runtime error.
			return nil
		}
		return makeIntNode(left / right)
	case "<":
		return makeBoolNode(left < right)
	case ">":
		return makeBoolNode(left > right)
	case "==":
		return makeBoolNode(left == right)
	case "!=":
		return makeBoolNode(left != right)
	default:
		return nil
	}
}

// tryFoldPrefix attempts to compute a prefix expression on a literal operand.
func tryFoldPrefix(expr *ast.PrefixExpression) ast.Expression {
	switch expr.Operator {
	case "-":
		if intLit, ok := expr.Right.(*ast.IntegerLiteral); ok {
			return makeIntNode(-intLit.Value)
		}
	case "!":
		if boolLit, ok := expr.Right.(*ast.Boolean); ok {
			return makeBoolNode(!boolLit.Value)
		}
	}
	return expr
}

func makeIntNode(val int64) *ast.IntegerLiteral {
	return &ast.IntegerLiteral{
		Token: token.Token{Type: token.INT, Literal: fmt.Sprintf("%d", val)},
		Value: val,
	}
}

func makeBoolNode(val bool) *ast.Boolean {
	lit := "false"
	tokType := token.TokenType(token.FALSE)
	if val {
		lit = "true"
		tokType = token.TokenType(token.TRUE)
	}
	return &ast.Boolean{
		Token: token.Token{Type: tokType, Literal: lit},
		Value: val,
	}
}
