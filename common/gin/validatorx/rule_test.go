package validatorx

import (
	"testing"
)

func TestIsOverLength(t *testing.T) {
	// 正常情况，utf8 编码，不超过长度
	err := IsOverLength("hello", "utf8", 1)
	if err != nil {
		t.Log(err)
	}

	// 正常情况，utf16 编码，不超过长度
	err = IsOverLength("hello", "utf16", 2)
	if err != nil {
		t.Log(err)
	}

	// 超过长度，utf8 编码
	err = IsOverLength("hello world", "utf8", 3)
	if err != nil {
		t.Log(err)
	}

	// 超过长度，utf16 编码
	err = IsOverLength("hello world", "utf16", 4)
	if err != nil {
		t.Log(err)
	}

	// 非法编码
	err = IsOverLength("hello", "invalid", 5)
	if err != nil {
		t.Log(err)
	}
}
