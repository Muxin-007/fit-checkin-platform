package response

import ginResp "platform/common/gin/response"

const ModuleFitness ginResp.ModuleCode = "102"

var (
	WechatServiceError      = ginResp.Code(ModuleFitness, "001")
	SessionInvalid          = ginResp.Code(ModuleFitness, "002")
	AccountDisabled         = ginResp.Code(ModuleFitness, "003")
	PermissionDenied        = ginResp.Code(ModuleFitness, "004")
	ResourceNotFound        = ginResp.Code(ModuleFitness, "005")
	GroupFull               = ginResp.Code(ModuleFitness, "006")
	AlreadyJoined           = ginResp.Code(ModuleFitness, "007")
	InvitationInvalid       = ginResp.Code(ModuleFitness, "008")
	ContentRejected         = ginResp.Code(ModuleFitness, "009")
	ReminderRateLimited     = ginResp.Code(ModuleFitness, "010")
	SubscriptionUnavailable = ginResp.Code(ModuleFitness, "011")
	ContentReviewPending    = ginResp.Code(ModuleFitness, "012")
)
