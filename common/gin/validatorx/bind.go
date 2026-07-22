package validatorx

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/jinzhu/copier"
)

// 绑定并校验请求结构体参数
func BindAndValid[T any](c *gin.Context) (*T, error) {
	var data T
	if err := c.ShouldBind(&data); err != nil {
		// 统一recover处理
		errs := ConvBindValidationError(data, err)
		return nil, errs
	} else {
		return &data, nil
	}
}

// 绑定请求体中的s结构体拷贝至另一T结构体
func BindJsonAndCopyTo[S any, T any](c *gin.Context) (*T, error) {
	data, err := BindAndValid[S](c)
	if err != nil {
		return nil, err
	}
	var toStruct T
	err = copier.Copy(&toStruct, &data)
	if err != nil {
		return nil, err
	}
	return &toStruct, nil
}

// 转译参数校验错误，并将参数校验错误为业务异常错误（统一recover处理）
func ConvBindValidationError(data any, err error) error {
	if e, ok := err.(validator.ValidationErrors); ok {
		// 调用validatorx.Translate2Str方法进行校验错误转译
		return errors.New(Translate2Str(data, e))
	}
	return err
}
