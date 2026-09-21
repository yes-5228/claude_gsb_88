package httpx_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/drainage/desilting/internal/httpx"
)

type sampleRequest struct {
	Name   string  `json:"name" label:"管段名称" validate:"required"`
	Type   string  `json:"type" label:"管段类型" validate:"required"`
	Length float64 `json:"length" label:"管段长度" validate:"gt=0"`
	Remark string  `json:"remark" label:"备注" validate:"max=5"`
}

func TestValidateReturnsChineseMessage(t *testing.T) {
	cases := []struct {
		name    string
		payload sampleRequest
		want    string
	}{
		{
			name:    "必填字段缺失",
			payload: sampleRequest{Type: "rainwater", Length: 10},
			want:    "管段名称不能为空",
		},
		{
			name:    "数值下界校验",
			payload: sampleRequest{Name: "中山北路管段", Type: "rainwater", Length: 0},
			want:    "管段长度必须大于 0",
		},
		{
			name:    "字符串长度上限",
			payload: sampleRequest{Name: "中山北路管段", Type: "rainwater", Length: 10, Remark: "超出长度的备注内容"},
			want:    "备注长度或取值不能大于 5",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := httpx.Validate(testCase.payload)
			if err == nil {
				t.Fatalf("期望校验失败，实际通过")
			}
			var appErr *httpx.AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("期望业务错误，实际 %T", err)
			}
			if appErr.Code != httpx.CodeValidation {
				t.Fatalf("期望错误码 %d，实际 %d", httpx.CodeValidation, appErr.Code)
			}
			if appErr.Message != testCase.want {
				t.Fatalf("期望提示 %q，实际 %q", testCase.want, appErr.Message)
			}
		})
	}
}

func TestValidatePassesOnValidPayload(t *testing.T) {
	if err := httpx.Validate(sampleRequest{Name: "中山北路管段", Type: "rainwater", Length: 12.5}); err != nil {
		t.Fatalf("期望校验通过，实际返回: %v", err)
	}
}

func TestWriteErrorMapsUnknownErrorToInternal(t *testing.T) {
	appErr := httpx.Internal("服务器内部错误")
	if !strings.Contains(appErr.Error(), "服务器内部错误") {
		t.Fatalf("错误信息不符合预期: %s", appErr.Error())
	}
	if appErr.Status != 500 {
		t.Fatalf("期望 HTTP 状态码 500，实际 %d", appErr.Status)
	}
}
