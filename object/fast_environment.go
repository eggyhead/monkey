package object

// FastEnvironment is an array-backed environment for resolved variables.
// Instead of map[string]Object lookups, variables are accessed by numeric
// slot index, turning every variable read/write into a bounds-checked
// array access (~1ns vs ~20-50ns for map lookup).
type FastEnvironment struct {
	slots []Object
	outer *FastEnvironment
}

// NewFastEnvironment creates a new environment with the given number of slots.
func NewFastEnvironment(size int) *FastEnvironment {
	return &FastEnvironment{
		slots: make([]Object, size),
	}
}

// NewEnclosedFastEnvironment creates a child environment linked to an outer.
func NewEnclosedFastEnvironment(size int, outer *FastEnvironment) *FastEnvironment {
	return &FastEnvironment{
		slots: make([]Object, size),
		outer: outer,
	}
}

// GetByIndex retrieves a variable by walking `depth` outer links then
// indexing into that scope's slot array.
func (e *FastEnvironment) GetByIndex(depth, index int) Object {
	env := e
	for i := 0; i < depth; i++ {
		env = env.outer
	}
	return env.slots[index]
}

// SetByIndex stores a value at the given depth and index.
func (e *FastEnvironment) SetByIndex(depth, index int, val Object) {
	env := e
	for i := 0; i < depth; i++ {
		env = env.outer
	}
	env.slots[index] = val
}
