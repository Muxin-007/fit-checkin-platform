package validatorx

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"unicode/utf16"

	"golang.org/x/net/idna"
)

const (
	MaxNameLength = 255
)

// IsValidDomain 检查给定的字符串是否为有效的域名
func IsValidDomain(domain string) bool {
	// 域名不能为空
	if domain == "" {
		return false
	}

	// 域名不能以"."开始或结束
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}

	// isValidLabel 检查单个域名标签是否有效
	isValidLabel := func(label string) bool {
		for _, r := range label {
			if ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') || ('0' <= r && r <= '9') || r == '-' {
				continue
			}
			return false
		}
		return !strings.HasPrefix(label, "-") && !strings.HasSuffix(label, "-")
	}

	// 使用"."分割域名，并检查每个部分
	domainParts := strings.Split(domain, ".")
	for _, part := range domainParts {
		// 每个部分必须是非空的
		if part == "" {
			return false
		}

		// 每个标签（子域名）长度限制在1-63个字符之间
		if len(part) < 1 || len(part) > 63 {
			return false
		}

		// 根据RFC 1034和RFC 1123的规定，域名部分只能包含字母(a-z/A-Z)，数字(0-9)以及连字符(-)
		// 连字符不能出现在开头或结尾
		if !isValidLabel(part) {
			return false
		}
	}

	// 尝试用net包中的方法解析域名，进一步验证其正确性
	if _, err := net.LookupHost(domain); err != nil {
		return false
	}
	return true
}

// 校验域名的正则表达式
var domainRegex = regexp.MustCompile(`^([a-zA-Z0-9-]+\.)+[a-zA-Z0-9-]{2,}$`)

// IsDomain 校验域名是否有效
func IsDomain(domain string) bool {
	// 转换为Punycode格式
	punycodeDomain, err := idna.ToASCII(domain)
	if err != nil {
		return false
	}

	// 检查是否符合域名的正则表达式
	if !domainRegex.MatchString(punycodeDomain) {
		return false
	}

	// 通过解析域名来进一步验证
	_, err = net.LookupHost(punycodeDomain)
	return err == nil
}

// IsOverLength 检查字符串是否超过指定的最大长度
func IsOverLength(str, encode string, num int64) error {
	switch encode {
	case "utf16":
		if num < int64(len(utf16.Encode([]rune(str)))) {
			return fmt.Errorf("最多支持%d个字符", num)
		}
	case "utf8":
		if num < int64(len(str)) {
			return fmt.Errorf("最多支持%d个字符", num)
		}
	default:
		if num < int64(len(str)) {
			return fmt.Errorf("最多支持%d个字符", num)
		}
	}
	return nil
}
