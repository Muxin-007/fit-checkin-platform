package validatorx

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

const CustomPatternTagName = "pattern"

var (
	regexpMap     map[string]*regexp.Regexp // key：正则表达式名称   value：正则表达式
	patternErrMsg map[string]string         // key：正则表达式名称   value：校验不通过时的错误消息提示
)

// 注册自定义正则表达式校验规则
func RegisterCustomPatterns() {
	// 账号用户名校验，使用该种方式可以复用正则表达式以及错误提示
	// 使用方式如：Username string `json:"username" binding:"pattern=account_username"`
	RegisterPattern("account_username", "^[a-zA-Z0-9_]{5,20}$", "只允许输入5-20位大小写字母、数字、下划线")
}

// 注册自定义正则表达式
func RegisterPattern(patternName string, regexpStr string, errMsg string) {
	if regexpMap == nil {
		regexpMap = make(map[string]*regexp.Regexp, 0)
		patternErrMsg = make(map[string]string)
	}
	regexpMap[patternName] = regexp.MustCompile(regexpStr)
	patternErrMsg[patternName] = errMsg
}

// 自定义正则表达式校验器函数
func patternValidFunc(f validator.FieldLevel) bool {
	reg := regexpMap[f.Param()]
	if reg == nil {
		return false
	}

	return reg.MatchString(f.Field().String())
}
