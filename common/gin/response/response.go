package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Response struct {
	Code ResponseCode `json:"code,inline"`
	Data any          `json:"data"`
	Msg  string       `json:"msg"`
}

func (r *Response) Ok(c *gin.Context) {
	c.JSON(http.StatusOK, r)
}

func (r *Response) HttpStatus(status int, c *gin.Context) {
	c.JSON(status, r)
}

func RespWithCode(code ResponseCode, c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code: code,
	})
}

func RespWithData(code ResponseCode, data any, c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code: code,
		Data: data,
	})
}

func RespWithMsg(code ResponseCode, msg string, c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code: code,
		Msg:  msg,
	})
}

func RespWithMsgAndData(code ResponseCode, msg string, data any, c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code: code,
		Msg:  msg,
		Data: data,
	})
}

func RespWithStatusUnauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, Response{
		Code: Unauthorized,
	})
}

func RespWithStatusForbidden(c *gin.Context) {
	c.JSON(http.StatusForbidden, Response{
		Code: UserDisableAccess,
	})
}

func CheckError(err error, logger *zap.SugaredLogger, errCode, successCode ResponseCode, templete string) *Response {
	if err != nil {
		logger.Errorf(templete, err)
		return &Response{Code: errCode}
	}
	return &Response{Code: successCode}
}

func NewRespWithData(code ResponseCode, data any) *Response {
	return &Response{Code: code, Data: data}
}

type PageResult[T any] struct {
	List     []T `json:"list"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}
