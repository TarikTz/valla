package steps

type OutputStructure = RuntimeSelect

func NewOutputStructure() OutputStructure {
	return NewRuntimeSelect("Project output structure:", []string{"Monorepo", "Separate folders", "WordPress"})
}
