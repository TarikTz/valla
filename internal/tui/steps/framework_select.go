package steps

// FrameworkSelect wraps RuntimeSelect for framework selection.
type FrameworkSelect = RuntimeSelect

func NewFrameworkSelect(prompt string, options []string) FrameworkSelect {
	return NewRuntimeSelect(prompt, runtimeOptionsAll(options))
}
