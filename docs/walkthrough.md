# Evaluator Walkthrough — Sequence Diagrams & Code References

Sequence diagrams for the key discussion questions from [Discussion #163](https://github.com/github/tech-book-club/discussions/163), traced through the Monkey evaluator.

---

## Q1: Tracing `let x = 5; let y = x + 8; y`

> At each step: which AST node type is being evaluated, what does Eval return, and what changes in the environment?

```mermaid
sequenceDiagram
    participant P as evalProgram
    participant E as Eval
    participant Env as Environment

    Note over P: Program has 3 statements

    P->>E: Eval(*ast.LetStatement "let x = 5")
    activate E
    E->>E: Eval(*ast.IntegerLiteral 5)
    Note right of E: CachedInteger(5) → Integer{5}
    E->>Env: Set("x", Integer{5})
    Note over Env: store: {"x": 5}
    deactivate E

    P->>E: Eval(*ast.LetStatement "let y = x + 8")
    activate E
    E->>E: Eval(*ast.InfixExpression "x + 8")
    activate E
    E->>E: Eval(*ast.Identifier "x")
    E->>Env: Get("x")
    Env-->>E: Integer{5}
    E->>E: Eval(*ast.IntegerLiteral 8)
    Note right of E: CachedInteger(8) → Integer{8}
    E->>E: evalIntegerInfixExpression("+", 5, 8)
    Note right of E: 5 + 8 = 13
    deactivate E
    E-->>E: Integer{13}
    E->>Env: Set("y", Integer{13})
    Note over Env: store: {"x": 5, "y": 13}
    deactivate E

    P->>E: Eval(*ast.ExpressionStatement "y")
    activate E
    E->>E: Eval(*ast.Identifier "y")
    E->>Env: Get("y")
    Env-->>E: Integer{13}
    deactivate E

    Note over P: evalProgram returns Integer{13}
```

**Code references:**
- `Eval` type switch dispatches each node: [`evaluator/evaluator.go:15-94`](evaluator/evaluator.go#L15-L94)
- `LetStatement` evaluates RHS then calls `env.Set()`: [`evaluator/evaluator.go:35-40`](evaluator/evaluator.go#L35-L40)
- `InfixExpression` evaluates left, right, then dispatches: [`evaluator/evaluator.go:56-67`](evaluator/evaluator.go#L56-L67)
- `evalIntegerInfixExpression` does the arithmetic: [`evaluator/evaluator.go:194-222`](evaluator/evaluator.go#L194-L222)
- `Identifier` lookup via `env.Get()`: [`evaluator/evaluator.go:72-73`](evaluator/evaluator.go#L72-L73) → [`object/environment.go:19-25`](object/environment.go#L19-L25)
- `evalProgram` iterates statements, unwraps return/error: [`evaluator/evaluator.go:97-112`](evaluator/evaluator.go#L97-L112)

---

## Q5: Prefix Expressions — `!5` and `!true`

> Why does the order matter? What does `evalBangOperatorExpression` do for `!5`?

```mermaid
sequenceDiagram
    participant E as Eval
    participant PFX as evalPrefixExpression
    participant Bang as evalBangOperatorExpression

    Note over E: Program: !5

    E->>E: Eval(*ast.PrefixExpression "!5")
    activate E
    E->>E: Eval(*ast.IntegerLiteral 5)
    Note right of E: Must evaluate operand FIRST<br/>before applying operator
    E-->>E: Integer{5}
    E->>PFX: evalPrefixExpression("!", Integer{5})
    activate PFX
    PFX->>Bang: evalBangOperatorExpression(Integer{5})
    activate Bang
    Note over Bang: switch right {<br/>case TRUE → FALSE<br/>case FALSE → TRUE<br/>case NULL → TRUE<br/>default → FALSE ← hits this!<br/>}
    Bang-->>PFX: FALSE
    deactivate Bang
    PFX-->>E: FALSE
    deactivate PFX
    deactivate E

    Note over E: !5 = false because 5 is "not<br/>TRUE/FALSE/NULL" → default → FALSE
```

```mermaid
sequenceDiagram
    participant E as Eval
    participant Bang as evalBangOperatorExpression

    Note over E: Program: !true

    E->>E: Eval(*ast.PrefixExpression "!true")
    activate E
    E->>E: Eval(*ast.Boolean true)
    Note right of E: Returns singleton TRUE (ptr: 0x…380)
    E-->>E: TRUE (singleton)
    E->>Bang: evalBangOperatorExpression(TRUE)
    activate Bang
    Note over Bang: case TRUE → return FALSE
    Bang-->>E: FALSE
    deactivate Bang
    deactivate E
```

**Code references:**
- Operand evaluated first, then operator applied: [`evaluator/evaluator.go:49-54`](evaluator/evaluator.go#L49-L54)
- `evalBangOperatorExpression` — `default` case makes all non-boolean/null values falsy under `!`: [`evaluator/evaluator.go:172-183`](evaluator/evaluator.go#L172-L183)
- Boolean singletons (`TRUE`/`FALSE`): [`evaluator/evaluator.go:9-12`](evaluator/evaluator.go#L9-L12)

---

## Q6: Pointer Comparison — `==` for Booleans vs Integers

> Why does pointer comparison work for booleans but not integers?

```mermaid
sequenceDiagram
    participant E as Eval
    participant Infix as evalInfixExpression
    participant IntInfix as evalIntegerInfixExpression

    Note over E: Program: true == true

    E->>E: Eval(left: *ast.Boolean true)
    Note right of E: Returns singleton TRUE (ptr: 0xAAA)
    E->>E: Eval(right: *ast.Boolean true)
    Note right of E: Returns same singleton TRUE (ptr: 0xAAA)
    E->>Infix: evalInfixExpression("==", TRUE, TRUE)
    activate Infix
    Note over Infix: Both are BOOLEAN_OBJ, but<br/>== case comes before type check:<br/> case operator == "==":<br/>   return left == right<br/>Pointer comparison: 0xAAA == 0xAAA ✓
    Infix-->>E: TRUE
    deactivate Infix

    Note over E: ✅ Works! Both "true" literals<br/>are the same singleton pointer
```

```mermaid
sequenceDiagram
    participant E as Eval
    participant Infix as evalInfixExpression
    participant IntInfix as evalIntegerInfixExpression

    Note over E: Program: 5 == 5

    E->>E: Eval(left: *ast.IntegerLiteral 5)
    Note right of E: CachedInteger(5) → ptr: 0xBBB
    E->>E: Eval(right: *ast.IntegerLiteral 5)
    Note right of E: CachedInteger(5) → ptr: 0xBBB (same — cached!)
    E->>Infix: evalInfixExpression("==", Integer{5}, Integer{5})
    activate Infix
    Note over Infix: case left.Type() == INTEGER_OBJ<br/>  && right.Type() == INTEGER_OBJ:<br/>→ routes to evalIntegerInfixExpression<br/>(does NOT use pointer comparison)
    Infix->>IntInfix: evalIntegerInfixExpression("==", 5, 5)
    activate IntInfix
    Note over IntInfix: Unwraps int64 values:<br/>leftVal == rightVal → 5 == 5 ✓
    IntInfix-->>Infix: TRUE
    deactivate IntInfix
    Infix-->>E: TRUE
    deactivate Infix

    Note over E: ⚠️ Pointer comparison HAPPENS to<br/>work here (cache), but 256 == 256<br/>would FAIL — different heap allocs!
```

**Code references:**
- `evalInfixExpression` — integer path vs pointer-comparison path: [`evaluator/evaluator.go:152-170`](evaluator/evaluator.go#L152-L170)
  - Line 157: integers routed to value comparison
  - Line 159-160: `==` uses `left == right` (pointer) for non-integers
- `evalIntegerInfixExpression` — unwraps `int64`, compares values: [`evaluator/evaluator.go:214-215`](evaluator/evaluator.go#L214-L215)
- Integer cache (0–255 share pointers): [`object/object.go:39-57`](object/object.go#L39-L57)
- Boolean singletons guarantee pointer equality: [`evaluator/evaluator.go:9-12`](evaluator/evaluator.go#L9-L12)

---

## Q8: Truthiness — `if (5) { 10 }` and `if (0) { 99 }`

> `isTruthy` returns true for everything that isn't NULL or FALSE — including 0!

```mermaid
sequenceDiagram
    participant E as Eval
    participant If as evalIfExpression
    participant T as isTruthy

    Note over E: Program: if (5) { 10 }

    E->>If: evalIfExpression(...)
    activate If
    If->>E: Eval(condition: IntegerLiteral 5)
    E-->>If: Integer{5}
    If->>T: isTruthy(Integer{5})
    activate T
    Note over T: switch obj {<br/>case NULL → false<br/>case TRUE → true<br/>case FALSE → false<br/>default → true ← Integer{5} hits this!<br/>}
    T-->>If: true
    deactivate T
    If->>E: Eval(consequence: BlockStatement { 10 })
    E-->>If: Integer{10}
    deactivate If
```

```mermaid
sequenceDiagram
    participant E as Eval
    participant If as evalIfExpression
    participant T as isTruthy

    Note over E: Program: if (0) { 99 }
    Note over E: ⚠️ 0 is TRUTHY in Monkey!

    E->>If: evalIfExpression(...)
    activate If
    If->>E: Eval(condition: IntegerLiteral 0)
    E-->>If: Integer{0}
    If->>T: isTruthy(Integer{0})
    activate T
    Note over T: 0 is NOT null, NOT true, NOT false<br/>→ default → true
    T-->>If: true
    deactivate T
    If->>E: Eval(consequence: { 99 })
    E-->>If: Integer{99}
    deactivate If
```

**Code references:**
- `evalIfExpression` calls `isTruthy` on condition result: [`evaluator/evaluator.go:224-240`](evaluator/evaluator.go#L224-L240)
- `isTruthy` — only `NULL` and `FALSE` are falsy, everything else (including `0`) is truthy: [`evaluator/evaluator.go:254-265`](evaluator/evaluator.go#L254-L265)

---

## Q9: Return Statements — Wrapping Propagates Without Unwrapping

> Why does `evalBlockStatement` NOT unwrap, while `evalProgram` DOES?

```mermaid
sequenceDiagram
    participant P as evalProgram
    participant B1 as evalBlockStatement<br/>(outer if)
    participant B2 as evalBlockStatement<br/>(inner if)
    participant E as Eval

    Note over P: if (true) { if (true) { return 10; } return 1; }

    P->>E: Eval(outer IfExpression)
    E->>B1: evalBlockStatement [2 stmts]
    activate B1
    B1->>E: Eval(inner IfExpression)
    E->>B2: evalBlockStatement [1 stmt]
    activate B2

    B2->>E: Eval(*ast.ReturnStatement "return 10")
    E->>E: Eval(IntegerLiteral 10)
    E-->>E: Integer{10}
    Note right of E: Wraps: ReturnValue{Integer{10}}
    E-->>B2: ReturnValue{10}

    Note over B2: Sees RETURN_VALUE_OBJ →<br/>propagates WITHOUT unwrapping
    B2-->>B1: ReturnValue{10}
    deactivate B2

    Note over B1: Sees RETURN_VALUE_OBJ →<br/>propagates WITHOUT unwrapping<br/>⚡ "return 1" is NEVER reached!
    B1-->>P: ReturnValue{10}
    deactivate B1

    Note over P: evalProgram UNWRAPS:<br/>result.Value → Integer{10}
    P-->>P: Integer{10}
```

```mermaid
sequenceDiagram
    participant P as evalProgram
    participant B1 as evalBlockStatement<br/>(outer if)
    participant B2 as evalBlockStatement<br/>(inner if)
    participant E as Eval

    Note over P: ❌ BROKEN: if evalBlockStatement also unwrapped

    P->>E: Eval(outer IfExpression)
    E->>B1: evalBlockStatement [2 stmts]
    activate B1
    B1->>E: Eval(inner IfExpression)
    E->>B2: evalBlockStatement [1 stmt]
    activate B2

    B2->>E: Eval(ReturnStatement "return 10")
    E-->>B2: ReturnValue{10}

    Note over B2: ❌ Unwraps here!
    B2-->>B1: Integer{10} (no wrapper!)
    deactivate B2

    Note over B1: Sees plain Integer{10}<br/>NOT a ReturnValue — loop continues!
    B1->>E: Eval(ReturnStatement "return 1")
    E-->>B1: ReturnValue{1}
    Note over B1: ❌ Unwraps AGAIN
    B1-->>P: Integer{1}
    deactivate B1

    Note over P: ❌ Returns 1 instead of 10!
```

**Code references:**
- `ReturnStatement` wraps value in `ReturnValue`: [`evaluator/evaluator.go:28-33`](evaluator/evaluator.go#L28-L33)
- `evalBlockStatement` — checks for `RETURN_VALUE_OBJ` but does **not** unwrap, just propagates: [`evaluator/evaluator.go:114-132`](evaluator/evaluator.go#L114-L132) (lines 124-127)
- `evalProgram` — **does** unwrap `ReturnValue` to get the inner value: [`evaluator/evaluator.go:97-112`](evaluator/evaluator.go#L97-L112) (lines 104-105)
- `unwrapReturnValue` helper (used by `applyFunction`): [`evaluator/evaluator.go:319-325`](evaluator/evaluator.go#L319-L325)

---

## Q10: Error Propagation — `isError` Guards

> What does the `isError` guard prevent? What if you removed one?

```mermaid
sequenceDiagram
    participant P as evalProgram
    participant E as Eval
    participant Infix as evalInfixExpression

    Note over P: Program: 5 + true; 10

    P->>E: Eval(ExpressionStatement "5 + true")
    activate E
    E->>E: Eval(InfixExpression "5 + true")
    activate E
    E->>E: Eval(IntegerLiteral 5) → Integer{5}
    E->>E: Eval(Boolean true) → TRUE
    E->>Infix: evalInfixExpression("+", Integer, Boolean)
    activate Infix
    Note over Infix: left.Type() ≠ right.Type()<br/>→ Error{"type mismatch: INTEGER + BOOLEAN"}
    Infix-->>E: Error
    deactivate Infix
    deactivate E
    deactivate E

    Note over P: evalProgram sees Error →<br/>short-circuits, returns Error<br/>⚡ Statement "10" is NEVER evaluated
```

```mermaid
sequenceDiagram
    participant E as Eval
    participant Infix as evalInfixExpression

    Note over E: ❌ Without isError guard in InfixExpression
    Note over E: Program: (5 + true) + 3

    E->>E: Eval(InfixExpression "(5 + true) + 3")
    activate E
    E->>E: Eval(left: InfixExpression "5 + true")
    E->>Infix: evalInfixExpression("+", Integer, Boolean)
    Infix-->>E: Error{"type mismatch"}

    Note over E: ❌ No isError check!<br/>Proceeds to evaluate right side anyway
    E->>E: Eval(right: IntegerLiteral 3) → Integer{3}
    E->>Infix: evalInfixExpression("+", Error, Integer{3})
    activate Infix
    Note over Infix: Error.Type() = "ERROR"<br/>Integer.Type() = "INTEGER"<br/>Types don't match →<br/>Error{"type mismatch: ERROR + INTEGER"}<br/>❌ Original error message is LOST!
    Infix-->>E: new Error (misleading message)
    deactivate Infix
    deactivate E
```

**Code references:**
- `isError` helper: [`evaluator/evaluator.go:271-276`](evaluator/evaluator.go#L271-L276)
- Guard in `InfixExpression` — removing this would lose original errors: [`evaluator/evaluator.go:58-59`](evaluator/evaluator.go#L58-L59)
- Guard in `LetStatement`: [`evaluator/evaluator.go:37-38`](evaluator/evaluator.go#L37-L38)
- `evalProgram` short-circuits on Error: [`evaluator/evaluator.go:106-107`](evaluator/evaluator.go#L106-L107)

---

## Q11: Closures — `newAdder(2)(3)` — Where Does the `2` Live?

> Walk through what happens when `newAdder(2)` is called and then `addTwo(3)` is called.

```mermaid
sequenceDiagram
    participant P as evalProgram
    participant E as Eval
    participant GEnv as Global Env<br/>{}
    participant FnEnv1 as Enclosed Env 1<br/>(outer → Global)
    participant FnEnv2 as Enclosed Env 2<br/>(outer → Env1)

    Note over P: let newAdder = fn(x) { fn(y) { x + y } }

    P->>E: Eval(LetStatement "let newAdder = fn(x)...")
    activate E
    Note right of E: FunctionLiteral captures<br/>current env (Global) as closure
    E->>GEnv: Set("newAdder", Function{params:[x], env: Global})
    deactivate E
    Note over GEnv: {"newAdder": fn(x){...}}

    Note over P: let addTwo = newAdder(2)

    P->>E: Eval(CallExpression "newAdder(2)")
    activate E
    E->>GEnv: Get("newAdder") → Function{env: Global}
    Note right of E: extendFunctionEnv creates Env1<br/>enclosed by fn's CLOSURE (Global),<br/>NOT the call-site env
    E->>FnEnv1: NewEnclosedEnvironment(Global)
    E->>FnEnv1: Set("x", Integer{2})
    Note over FnEnv1: {"x": 2} → outer → Global

    Note right of E: Evaluating body: fn(y) { x + y }<br/>This inner fn captures Env1 as closure!
    E-->>E: Function{params:[y], env: Env1}
    deactivate E
    E->>GEnv: Set("addTwo", Function{env: Env1})
    Note over GEnv: {"newAdder": fn, "addTwo": fn(y){x+y}}

    Note over P: addTwo(3)

    P->>E: Eval(CallExpression "addTwo(3)")
    activate E
    E->>GEnv: Get("addTwo") → Function{env: Env1}
    Note right of E: extendFunctionEnv creates Env2<br/>enclosed by addTwo's CLOSURE (Env1)
    E->>FnEnv2: NewEnclosedEnvironment(Env1)
    E->>FnEnv2: Set("y", Integer{3})
    Note over FnEnv2: {"y": 3} → outer → Env1{"x": 2}

    Note right of E: Evaluating body: x + y
    E->>FnEnv2: Get("x")
    Note over FnEnv2: "x" not in Env2 → walk outer
    FnEnv2->>FnEnv1: Get("x")
    FnEnv1-->>E: Integer{2} ✅ Found in closure!
    E->>FnEnv2: Get("y")
    FnEnv2-->>E: Integer{3} ✅ Found locally

    Note right of E: evalIntegerInfixExpression("+", 2, 3)
    E-->>P: Integer{5}
    deactivate E
```

**Key insight:** `extendFunctionEnv` on line 310 uses `fn.Env` (the environment captured at **definition** time), not the caller's environment. This is why `addTwo` can still find `x=2` — it's in the closure chain.

**Code references:**
- `FunctionLiteral` captures current `env` as closure: [`evaluator/evaluator.go:75-78`](evaluator/evaluator.go#L75-L78)
- `applyFunction` calls `extendFunctionEnv`: [`evaluator/evaluator.go:295-304`](evaluator/evaluator.go#L295-L304)
- `extendFunctionEnv` — creates enclosed env from **`fn.Env`** (closure), not call-site: [`evaluator/evaluator.go:306-317`](evaluator/evaluator.go#L306-L317)
- `Environment.Get` walks the `outer` chain recursively: [`object/environment.go:19-25`](object/environment.go#L19-L25)
- `NewEnclosedEnvironment` links outer pointer: [`object/environment.go:3-7`](object/environment.go#L3-L7)
- `Function` object stores `Env` field (the closure environment): [`object/object.go:85-89`](object/object.go#L85-L89)
