package testsupport

import (
	"errors"
	"testing"

	"github.com/drainage/desilting/internal/httpx"
)

// RequireAppError 断言返回指定错误码的业务错误，并返回该错误以便进一步校验文案。
func RequireAppError(t *testing.T, err error, code int) *httpx.AppError {
	t.Helper()
	if err == nil {
		t.Fatalf("期望返回错误（错误码 %d），实际调用成功", code)
	}
	var appErr *httpx.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("期望业务错误，实际得到 %T: %v", err, err)
	}
	if appErr.Code != code {
		t.Fatalf("期望错误码 %d，实际 %d（%s）", code, appErr.Code, appErr.Message)
	}
	return appErr
}

// RequireNoError 断言调用没有出错。
func RequireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("期望调用成功，实际返回错误: %v", err)
	}
}
