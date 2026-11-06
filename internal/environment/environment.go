package environment

import (
	"errors"
	"fmt"

	"envman/internal/domain"
)

var ErrNeedElevation = errors.New("需要管理员权限")

// Backend is the platform boundary used by the application. Implementations
// may access the Windows registry or keep an in-memory environment for local
// development and tests.
type Backend interface {
	Load(domain.Scope) ([]domain.Variable, error)
	ApplyChanges(scope domain.Scope, before, desired []domain.Variable) error
	Elevated() bool
	RelaunchElevated(params string) error
}

func DescribeError(scope domain.Scope, err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, ErrNeedElevation) {
		return "系统变量需要管理员权限"
	}
	return fmt.Sprintf("无法读取%s：%v", scope, err)
}
