package steps

type EnvMode = RuntimeSelect

func NewEnvMode() EnvMode {
	return NewRuntimeSelect("How would you like to run this locally?", []string{"Local (.env)", "Docker Compose"})
}
