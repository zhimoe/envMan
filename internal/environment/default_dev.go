//go:build !windows

package environment

func NewDefault() Backend {
	return NewMemory(DemoVariables())
}
