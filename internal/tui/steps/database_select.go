package steps

type DatabaseSelect = RuntimeSelect

func NewDatabaseSelect(options []string) DatabaseSelect {
	return NewRuntimeSelect("Select your database:", runtimeOptionsAll(options))
}
