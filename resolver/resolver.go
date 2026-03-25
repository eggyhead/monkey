// Package resolver performs a pre-evaluation pass over the AST to resolve
// variable references to numeric (depth, index) pairs. This enables the
// evaluator to use array-based environments instead of hash-map lookups.
//
// The resolver walks the AST, tracking scopes as a stack. Each scope maps
// variable names to their slot index. When an Identifier is encountered,
// the resolver walks the scope stack to find which depth and index the
// variable lives at, and records this in a resolution table.
package resolver

import (
	"monkey/ast"
)

// ResolvedVar holds the resolved location of a variable:
// Depth = number of scope hops from the current scope (0 = local)
// Index = slot index within that scope's flat array
type ResolvedVar struct {
	Depth int
	Index int
}

// Table maps AST expression nodes (Identifiers) to their resolved locations.
// Using the ast.Expression pointer as a key means each unique variable
// *reference* in the AST gets its own resolution, even if two references
// share the same name.
type Table map[ast.Expression]ResolvedVar

// scope tracks variables declared in one lexical scope.
type scope struct {
	vars  map[string]int // name → slot index
	count int            // next available slot
}

func newScope() *scope {
	return &scope{vars: make(map[string]int)}
}

func (s *scope) define(name string) int {
	idx := s.count
	s.vars[name] = idx
	s.count++
	return idx
}

func (s *scope) lookup(name string) (int, bool) {
	idx, ok := s.vars[name]
	return idx, ok
}

// Resolver walks the AST and builds a resolution table.
type Resolver struct {
	scopes []*scope
	table  Table
}

// Resolve performs variable resolution on the program and returns the
// resolution table that the evaluator will use.
func Resolve(program *ast.Program) Table {
	r := &Resolver{
		table: make(Table),
	}
	// Top-level scope for global let bindings
	r.pushScope()
	for _, stmt := range program.Statements {
		r.resolveStatement(stmt)
	}
	r.popScope()
	return r.table
}

func (r *Resolver) pushScope() {
	r.scopes = append(r.scopes, newScope())
}

func (r *Resolver) popScope() {
	r.scopes = r.scopes[:len(r.scopes)-1]
}

func (r *Resolver) currentScope() *scope {
	return r.scopes[len(r.scopes)-1]
}

// define registers a variable in the current scope and returns its slot index.
func (r *Resolver) define(name string) int {
	return r.currentScope().define(name)
}

// resolveVar looks up a variable name through the scope chain and records
// the resolution. Returns true if found.
func (r *Resolver) resolveVar(node ast.Expression, name string) bool {
	// Walk scopes from innermost to outermost
	for i := len(r.scopes) - 1; i >= 0; i-- {
		if idx, ok := r.scopes[i].lookup(name); ok {
			depth := len(r.scopes) - 1 - i
			r.table[node] = ResolvedVar{Depth: depth, Index: idx}
			return true
		}
	}
	return false
}

func (r *Resolver) resolveStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.LetStatement:
		// Define the name first so recursive references (e.g., let fib = fn(x) { fib(x-1) })
		// can resolve the binding within the function body.
		r.define(s.Name.Value)
		r.resolveExpression(s.Value)
		r.resolveVar(s.Name, s.Name.Value)

	case *ast.ReturnStatement:
		if s.ReturnValue != nil {
			r.resolveExpression(s.ReturnValue)
		}

	case *ast.ExpressionStatement:
		if s.Expression != nil {
			r.resolveExpression(s.Expression)
		}

	case *ast.BlockStatement:
		for _, stmt := range s.Statements {
			r.resolveStatement(stmt)
		}
	}
}

func (r *Resolver) resolveExpression(expr ast.Expression) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *ast.Identifier:
		r.resolveVar(e, e.Value)

	case *ast.IntegerLiteral:
		// nothing to resolve

	case *ast.Boolean:
		// nothing to resolve

	case *ast.PrefixExpression:
		r.resolveExpression(e.Right)

	case *ast.InfixExpression:
		r.resolveExpression(e.Left)
		r.resolveExpression(e.Right)

	case *ast.IfExpression:
		r.resolveExpression(e.Condition)
		if e.Consequence != nil {
			r.resolveStatement(e.Consequence)
		}
		if e.Alternative != nil {
			r.resolveStatement(e.Alternative)
		}

	case *ast.FunctionLiteral:
		// Functions create a new scope for their parameters and body.
		r.pushScope()
		for _, param := range e.Parameters {
			r.define(param.Value)
		}
		if e.Body != nil {
			r.resolveStatement(e.Body)
		}
		r.popScope()

	case *ast.CallExpression:
		r.resolveExpression(e.Function)
		for _, arg := range e.Arguments {
			r.resolveExpression(arg)
		}
	}
}
