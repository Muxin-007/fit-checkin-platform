package response

import (
	"encoding/json"
	"fmt"
	"strconv"
)

const (
	ModuleCommon ModuleCode = "100"
	ModuleAdmin  ModuleCode = "101"
)

type ModuleCode string

type ResponseCode struct {
	moduleCode ModuleCode
	code       string
}

func Code(m ModuleCode, code string) ResponseCode {
	return ResponseCode{
		moduleCode: m,
		code:       code,
	}
}

func (r ResponseCode) Error() string {
	return r.code
}

func (rc ResponseCode) MarshalJSON() ([]byte, error) {
	code := fmt.Sprintf("%s%s", rc.moduleCode, rc.code)
	intCode, err := strconv.Atoi(code)
	if err != nil {
		return nil, err
	}
	return json.Marshal(intCode)
}

var (
	OperationSuccess      ResponseCode = Code("", "0")   // 0 操作成功
	ReqParameterException ResponseCode = Code("", "400") // 400 请求参数异常
	Unauthorized          ResponseCode = Code("", "401") // 401 未授权
	UserDisableAccess     ResponseCode = Code("", "403") // 403 禁止访问
	OperationFailed       ResponseCode = Code("", "500") // 500 操作失败

	Failed               ResponseCode = Code(ModuleCommon, "001") // 100021 接口异常读取msg属性返回
	ServiceUnavailable   ResponseCode = Code(ModuleCommon, "002") // 100002 服务不可用
	ServiceException     ResponseCode = Code(ModuleCommon, "003") // 100003 服务异常
	AuthFailed           ResponseCode = Code(ModuleCommon, "005") // 100005 认证失败
	TokenInvalid         ResponseCode = Code(ModuleCommon, "006") // 100006 用户信息解析失败
	TokenUnavailable     ResponseCode = Code(ModuleCommon, "008") // 100008 帐户令牌已失效,请重新登陆
	TokenIllegitimate    ResponseCode = Code(ModuleCommon, "009") // 100009 非法的帐户令牌,请重新登陆
	TokenExpired         ResponseCode = Code(ModuleCommon, "010") // 100010 授权已过期,请重新登陆
	TokenUnauthorized    ResponseCode = Code(ModuleCommon, "011") // 100011 非法访问，请登陆
	TokenAllowedNotEmpty ResponseCode = Code(ModuleCommon, "012") // 100012 token不能为空
	TokenException       ResponseCode = Code(ModuleCommon, "013") // 100013 token有误
	ExportFileError      ResponseCode = Code(ModuleCommon, "014") // 100014 导出文件失败
	FileUploadFailed     ResponseCode = Code(ModuleCommon, "015") // 100015 文件上传失败
	FileDownloadFailed   ResponseCode = Code(ModuleCommon, "016") // 100016 文件下载失败

	TransactionStartFailed    ResponseCode = Code(ModuleCommon, "017") // 100017 sql事务开启失败
	TransactionCommitFailed   ResponseCode = Code(ModuleCommon, "018") // 100018 sql事务提交失败
	TransactionRollbackFailed ResponseCode = Code(ModuleCommon, "019") // 100019 sql事务回滚失败

	JsonMarshalFailed ResponseCode = Code(ModuleCommon, "020") // 100020 json序列化失败

)
