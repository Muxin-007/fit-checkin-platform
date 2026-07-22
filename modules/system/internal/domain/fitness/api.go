package fitness

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"

	ginResp "platform/common/gin/response"
	"platform/common/gin/validatorx"
	"platform/ent/fitnesscheckinimage"
	"platform/ent/sysstoragefile"
	"platform/modules/shared"
	"platform/modules/system/internal/domain/fitness/support"
	"platform/modules/system/internal/global"
	fitnessReq "platform/modules/system/internal/models/request"
	systemUpload "platform/modules/system/pkg/upload"
)

type API struct{ *Service }

func (a *API) AdminWeb(c *gin.Context) {
	if c.Param("asset") == "" {
		c.Redirect(http.StatusFound, c.Request.URL.Path+"/")
		return
	}
	support.ServeAdminWeb(c)
}

func bind[T any](c *gin.Context) (*T, bool) {
	req, err := validatorx.BindAndValid[T](c)
	if err != nil {
		ginResp.RespWithMsg(ginResp.ReqParameterException, err.Error(), c)
		return nil, false
	}
	return req, true
}

func (a *API) Login(c *gin.Context) {
	req, ok := bind[fitnessReq.WechatLogin](c)
	if !ok {
		return
	}
	a.Service.Login(c, req).Ok(c)
}

func (a *API) Logout(c *gin.Context) {
	support.DeleteSession(c, c.GetHeader("Authorization"))
	(&ginResp.Response{Code: ginResp.OperationSuccess}).Ok(c)
}

func (a *API) Profile(c *gin.Context) {
	a.Service.GetProfile(c, support.UserID(c)).Ok(c)
}

func (a *API) UpdateProfile(c *gin.Context) {
	req, ok := bind[fitnessReq.UpdateProfile](c)
	if ok {
		a.Service.UpdateProfile(c, support.UserID(c), req).Ok(c)
	}
}

func (a *API) UpdateSettings(c *gin.Context) {
	req, ok := bind[fitnessReq.UpdateSettings](c)
	if ok {
		a.Service.UpdateSettings(c, support.UserID(c), req).Ok(c)
	}
}

func (a *API) CancelAccount(c *gin.Context) {
	a.Service.CancelAccount(c, support.UserID(c)).Ok(c)
}

func (a *API) Home(c *gin.Context) {
	a.Service.Home(c, support.UserID(c)).Ok(c)
}

func (a *API) UploadImage(c *gin.Context) {
	_, file, err := c.Request.FormFile("file")
	if err != nil {
		ginResp.RespWithMsg(ginResp.ReqParameterException, "请选择图片", c)
		return
	}
	a.Service.UploadImage(c, support.UserID(c), c.PostForm("purpose"), file).Ok(c)
}

func (a *API) FileStatus(c *gin.Context) {
	a.Service.FileStatus(c, support.UserID(c), c.Param("id")).Ok(c)
}

func (a *API) PublicConfig(c *gin.Context) {
	a.Service.PublicConfig(c).Ok(c)
}

func (a *API) ListGroups(c *gin.Context) {
	a.Service.ListGroups(c, support.UserID(c)).Ok(c)
}

func (a *API) CreateGroup(c *gin.Context) {
	req, ok := bind[fitnessReq.CreateGroup](c)
	if ok {
		a.Service.CreateGroup(c, support.UserID(c), req).Ok(c)
	}
}

func (a *API) GetGroup(c *gin.Context) {
	a.Service.GetGroup(c, support.UserID(c), c.Param("id")).Ok(c)
}

func (a *API) UpdateGroup(c *gin.Context) {
	req, ok := bind[fitnessReq.UpdateGroup](c)
	if ok {
		req.ID = c.Param("id")
		a.Service.UpdateGroup(c, support.UserID(c), req).Ok(c)
	}
}

func (a *API) CreateInvitation(c *gin.Context) {
	req, ok := bind[fitnessReq.CreateInvitation](c)
	if ok {
		req.GroupID = c.Param("id")
		a.Service.CreateInvitation(c, support.UserID(c), req).Ok(c)
	}
}

func (a *API) GetInvitation(c *gin.Context) {
	a.Service.GetInvitation(c, support.UserID(c), c.Param("id")).Ok(c)
}

func (a *API) InvitationPreview(c *gin.Context) {
	a.Service.InvitationPreview(c, c.Param("code")).Ok(c)
}

func (a *API) JoinGroup(c *gin.Context) {
	req, ok := bind[fitnessReq.JoinGroup](c)
	if ok {
		a.Service.JoinGroup(c, support.UserID(c), req).Ok(c)
	}
}

func (a *API) ReviewMember(c *gin.Context) {
	req, ok := bind[fitnessReq.ReviewMember](c)
	if ok {
		a.Service.ReviewMember(c, support.UserID(c), c.Param("id"), req).Ok(c)
	}
}

func (a *API) LeaveGroup(c *gin.Context) {
	a.Service.LeaveGroup(c, support.UserID(c), c.Param("id")).Ok(c)
}

func (a *API) RemoveMember(c *gin.Context) {
	a.Service.RemoveMember(c, support.UserID(c), c.Param("id"), c.Param("memberId")).Ok(c)
}

func (a *API) DissolveGroup(c *gin.Context) {
	a.Service.DissolveGroup(c, support.UserID(c), c.Param("id")).Ok(c)
}

func (a *API) UpsertCheckin(c *gin.Context) {
	req, ok := bind[fitnessReq.UpsertCheckin](c)
	if ok {
		a.Service.UpsertTodayCheckin(c, support.UserID(c), req).Ok(c)
	}
}

func (a *API) TodayCheckin(c *gin.Context) {
	a.Service.GetTodayCheckin(c, support.UserID(c)).Ok(c)
}

func (a *API) GetCheckin(c *gin.Context) {
	a.Service.GetCheckin(c, support.UserID(c), c.Param("id")).Ok(c)
}

func (a *API) Calendar(c *gin.Context) {
	a.Service.Calendar(c, support.UserID(c), c.Query("month")).Ok(c)
}

func (a *API) DeleteCheckin(c *gin.Context) {
	a.Service.DeleteCheckin(c, support.UserID(c), c.Param("id")).Ok(c)
}

func (a *API) AuthorizeSubscription(c *gin.Context) {
	req, ok := bind[fitnessReq.AuthorizeSubscription](c)
	if ok {
		a.Service.AuthorizeSubscription(c, support.UserID(c), req).Ok(c)
	}
}

func (a *API) ManualReminder(c *gin.Context) {
	req, ok := bind[fitnessReq.ManualReminder](c)
	if ok {
		a.Service.ManualReminder(c, support.UserID(c), req).Ok(c)
	}
}

func (a *API) DownloadFile(c *gin.Context) {
	id := c.Param("id")
	scope := c.Query("scope")
	expiresAt, err := strconv.ParseInt(c.Query("expires"), 10, 64)
	if (scope != "public" && scope != "private" && scope != "audit") ||
		err != nil || !support.VerifyFileSignature(id, scope, expiresAt, c.Query("signature")) {
		c.Status(http.StatusForbidden)
		return
	}
	query := shared.EntClient.SysStorageFile.Query().Where(sysstoragefile.IDEQ(id))
	if scope == "public" {
		query = query.Where(sysstoragefile.AuditStatusEQ(sysstoragefile.AuditStatusApproved))
	} else {
		query = query.Where(sysstoragefile.AuditStatusNEQ(sysstoragefile.AuditStatusRejected))
	}
	file, err := query.Only(c)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	reader, err := systemUpload.DownloadFile(c, id)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer reader.Close()
	contentType := mime.TypeByExtension(filepath.Ext(file.Name))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)
	if scope == "public" {
		c.Header("Cache-Control", "private, max-age=3600")
	} else {
		c.Header("Cache-Control", "no-store")
	}
	if _, err = io.Copy(c.Writer, reader); err != nil {
		shared.Logger.Errorf("write image response failed: %s", err)
	}
}

type mediaCallback struct {
	TraceID string `json:"trace_id"`
	Result  struct {
		Suggest string `json:"suggest"`
	} `json:"result"`
}

func (a *API) VerifyMediaCallback(c *gin.Context) {
	if !support.VerifyMessageSignature(
		global.Cfg.System.Wechat.MessageToken,
		c.Query("timestamp"), c.Query("nonce"), c.Query("signature"),
	) {
		c.Status(http.StatusForbidden)
		return
	}
	c.String(http.StatusOK, c.Query("echostr"))
}

func (a *API) MediaCallback(c *gin.Context) {
	if !support.VerifyMessageSignature(
		global.Cfg.System.Wechat.MessageToken,
		c.Query("timestamp"), c.Query("nonce"), c.Query("signature"),
	) {
		c.Status(http.StatusForbidden)
		return
	}
	var callback mediaCallback
	if err := c.ShouldBindJSON(&callback); err != nil || callback.TraceID == "" {
		c.Status(http.StatusBadRequest)
		return
	}
	status := sysstoragefile.AuditStatusRejected
	if callback.Result.Suggest == "pass" {
		status = sysstoragefile.AuditStatusApproved
	} else if callback.Result.Suggest == "review" {
		status = sysstoragefile.AuditStatusPending
	}
	files, err := shared.EntClient.SysStorageFile.Query().
		Where(sysstoragefile.AuditTraceIDEQ(callback.TraceID)).All(c)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	fileIDs := make([]string, 0, len(files))
	for _, file := range files {
		fileIDs = append(fileIDs, file.ID)
	}
	if len(fileIDs) == 0 {
		c.String(http.StatusOK, "success")
		return
	}
	tx, err := shared.EntClient.Tx(c)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.SysStorageFile.Update().
		Where(sysstoragefile.IDIn(fileIDs...)).SetAuditStatus(status).Save(c); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if _, err = tx.FitnessCheckinImage.Update().
		Where(fitnesscheckinimage.StorageFileIDIn(fileIDs...)).
		SetAuditStatus(fitnesscheckinimage.AuditStatus(status)).Save(c); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if err = tx.Commit(); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.String(http.StatusOK, "success")
}

func (a *API) AdminDashboard(c *gin.Context) { a.Service.AdminDashboard(c).Ok(c) }

func (a *API) AdminUsers(c *gin.Context) {
	req, ok := bind[fitnessReq.AdminPage](c)
	if ok {
		a.Service.AdminUsers(c, req).Ok(c)
	}
}

func (a *API) AdminUpdateUserStatus(c *gin.Context) {
	req, ok := bind[fitnessReq.AdminUserStatus](c)
	if ok {
		a.Service.AdminUpdateUserStatus(c, c.Param("id"), req).Ok(c)
	}
}

func (a *API) AdminGroups(c *gin.Context) {
	req, ok := bind[fitnessReq.AdminPage](c)
	if ok {
		a.Service.AdminGroups(c, req).Ok(c)
	}
}

func (a *API) AdminDissolveGroup(c *gin.Context) {
	a.Service.AdminDissolveGroup(c, c.Param("id")).Ok(c)
}

func (a *API) AdminCheckins(c *gin.Context) {
	req, ok := bind[fitnessReq.AdminPage](c)
	if ok {
		a.Service.AdminCheckins(c, req).Ok(c)
	}
}

func (a *API) AdminDeleteCheckin(c *gin.Context) {
	a.Service.AdminDeleteCheckin(c, c.Param("id")).Ok(c)
}

func (a *API) AdminAuditCheckin(c *gin.Context) {
	req, ok := bind[fitnessReq.AdminCheckinAudit](c)
	if ok {
		a.Service.AdminAuditCheckin(c, c.Param("id"), req).Ok(c)
	}
}

func (a *API) AdminReminders(c *gin.Context) {
	req, ok := bind[fitnessReq.AdminPage](c)
	if ok {
		a.Service.AdminReminders(c, req).Ok(c)
	}
}

func (a *API) AdminUpdateConfig(c *gin.Context) {
	req, ok := bind[fitnessReq.UpdateConfig](c)
	if ok {
		a.Service.AdminUpdateConfig(c, c.Param("key"), req).Ok(c)
	}
}
