package steps

// StepDone is sent by a step model when the user has made a selection.
type StepDone struct {
	Value interface{}
}
