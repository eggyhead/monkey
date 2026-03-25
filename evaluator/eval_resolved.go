package evaluator

import (
	"monkey/ast"
	"monkey/object"
	"monkey/resolver"
)

// EvalResolved evaluates the program using resolved variable indices
// and array-backed environments. This replaces hash-map lookups with
// direct array indexing for all resolved variables.
func EvalResolved(program *ast.Program, table resolver.Table) object.Object {
	// Count top-level variables for the global scope
	globalSize := countScopeVars(program.Statements)
	env := object.NewFastEnvironment(globalSize)
	return evalResolved(program, env, table)
}

func evalResolved(node ast.Node, env *object.FastEnvironment, table resolver.Table) object.Object {
	switch node := node.(type) {

	case *ast.Program:
		return evalResolvedProgram(node, env, table)

	case *ast.BlockStatement:
		return evalResolvedBlockStatement(node, env, table)

	case *ast.ExpressionStatement:
		return evalResolved(node.Expression, env, table)

	case *ast.ReturnStatement:
		val := evalResolved(node.ReturnValue, env, table)
		if isError(val) {
			return val
		}
		return &object.ReturnValue{Value: val}

	case *ast.LetStatement:
		val := evalResolved(node.Value, env, table)
		if isError(val) {
			return val
		}
		// Use resolved index for Set if available
		if rv, ok := table[node.Name]; ok {
			env.SetByIndex(rv.Depth, rv.Index, val)
		}

	case *ast.IntegerLiteral:
		return object.CachedInteger(node.Value)

	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)

	case *ast.PrefixExpression:
		right := evalResolved(node.Right, env, table)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)

	case *ast.InfixExpression:
		left := evalResolved(node.Left, env, table)
		if isError(left) {
			return left
		}
		right := evalResolved(node.Right, env, table)
		if isError(right) {
			return right
		}
		return evalInfixExpression(node.Operator, left, right)

	case *ast.IfExpression:
		return evalResolvedIfExpression(node, env, table)

	case *ast.Identifier:
		// Fast path: use resolved index
		if rv, ok := table[node]; ok {
			val := env.GetByIndex(rv.Depth, rv.Index)
			if val != nil {
				return val
			}
		}
		return newError("identifier not found: %s", node.Value)

	case *ast.FunctionLiteral:
		params := node.Parameters
		body := node.Body
		return &object.ResolvedFunction{
			Parameters: params,
			Body:       body,
			Env:        env,
			Table:      table,
		}

	case *ast.CallExpression:
		function := evalResolved(node.Function, env, table)
		if isError(function) {
			return function
		}
		args := evalResolvedExpressions(node.Arguments, env, table)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		return applyResolvedFunction(function, args, table)
	}

	return nil
}

func evalResolvedProgram(program *ast.Program, env *object.FastEnvironment, table resolver.Table) object.Object {
	var result object.Object
	for _, statement := range program.Statements {
		result = evalResolved(statement, env, table)
		switch result := result.(type) {
		case *object.ReturnValue:
			return result.Value
		case *object.Error:
			return result
		}
	}
	return result
}

func evalResolvedBlockStatement(block *ast.BlockStatement, env *object.FastEnvironment, table resolver.Table) object.Object {
	var result object.Object
	for _, statement := range block.Statements {
		result = evalResolved(statement, env, table)
		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ {
				return result
			}
		}
	}
	return result
}

func evalResolvedIfExpression(ie *ast.IfExpression, env *object.FastEnvironment, table resolver.Table) object.Object {
	condition := evalResolved(ie.Condition, env, table)
	if isError(condition) {
		return condition
	}
	if isTruthy(condition) {
		return evalResolved(ie.Consequence, env, table)
	} else if ie.Alternative != nil {
		return evalResolved(ie.Alternative, env, table)
	} else {
		return NULL
	}
}

func evalResolvedExpressions(exps []ast.Expression, env *object.FastEnvironment, table resolver.Table) []object.Object {
	var result []object.Object
	for _, e := range exps {
		evaluated := evalResolved(e, env, table)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}
	return result
}

func applyResolvedFunction(fn object.Object, args []object.Object, table resolver.Table) object.Object {
	switch function := fn.(type) {
	case *object.ResolvedFunction:
		fnTable := function.Table.(resolver.Table)
		scopeSize := len(function.Parameters) + countScopeVars(function.Body.Statements)
		extendedEnv := object.NewEnclosedFastEnvironment(scopeSize, function.Env)
		for i, param := range function.Parameters {
			_ = param
			extendedEnv.SetByIndex(0, i, args[i])
		}
		evaluated := evalResolved(function.Body, extendedEnv, fnTable)
		return unwrapReturnValue(evaluated)
	default:
		return newError("not a function: %s", fn.Type())
	}
}

// countScopeVars counts the number of let statements in a list of statements.
// This tells us how many slots a scope needs for local variables.
func countScopeVars(stmts []ast.Statement) int {
	count := 0
	for _, stmt := range stmts {
		if _, ok := stmt.(*ast.LetStatement); ok {
			count++
		}
	}
	return count
}
