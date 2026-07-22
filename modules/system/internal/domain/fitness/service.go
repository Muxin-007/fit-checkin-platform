package fitness

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	ginResp "platform/common/gin/response"
	"platform/ent"
	"platform/ent/fitnesscheckin"
	"platform/ent/fitnesscheckingroup"
	"platform/ent/fitnesscheckinimage"
	"platform/ent/fitnessconfig"
	"platform/ent/fitnesscontentreport"
	"platform/ent/fitnessgroup"
	"platform/ent/fitnessgroupinvitation"
	"platform/ent/fitnessgroupmember"
	"platform/ent/fitnessreminderlog"
	"platform/ent/fitnesssubscriptionauthorization"
	"platform/ent/sysstoragefile"
	"platform/ent/sysuser"
	"platform/modules/shared"
	"platform/modules/system/internal/domain/fitness/support"
	"platform/modules/system/internal/global"
	fitnessReq "platform/modules/system/internal/models/request"
	fitnessResp "platform/modules/system/internal/models/response"
	internalUpload "platform/modules/system/internal/tools/upload"
	systemUpload "platform/modules/system/pkg/upload"
)

type Service struct{}

func success(data any) *ginResp.Response {
	return &ginResp.Response{Code: ginResp.OperationSuccess, Data: data}
}

func failed(code ginResp.ResponseCode, message string) *ginResp.Response {
	return &ginResp.Response{Code: code, Msg: message}
}

func (s *Service) Login(ctx context.Context, req *fitnessReq.WechatLogin) *ginResp.Response {
	session, err := support.Code2Session(ctx, req.Code)
	if err != nil {
		shared.Logger.Errorf("wechat login failed: %s", err)
		return failed(fitnessResp.WechatServiceError, "微信登录失败，请稍后重试")
	}
	user, err := shared.EntClient.SysUser.Query().Where(sysuser.OpenidEQ(session.OpenID)).Only(ctx)
	if ent.IsNotFound(err) {
		user, err = shared.EntClient.SysUser.Create().
			SetOpenid(session.OpenID).
			SetNillableUnionid(nonEmpty(session.UnionID)).
			Save(ctx)
	}
	if err != nil {
		shared.Logger.Errorf("create or query wechat user failed: %s", err)
		return failed(ginResp.OperationFailed, "登录用户保存失败")
	}
	if user.Unionid == "" && session.UnionID != "" {
		if updated, updateErr := shared.EntClient.SysUser.UpdateOneID(user.ID).SetUnionid(session.UnionID).Save(ctx); updateErr == nil {
			user = updated
		}
	}
	if user.Status != sysuser.StatusActive {
		return failed(fitnessResp.AccountDisabled, "账号当前不可用")
	}
	token, expiresAt, err := support.CreateSession(ctx, user.ID)
	if err != nil {
		shared.Logger.Errorf("create mini program session failed: %s", err)
		return failed(ginResp.OperationFailed, "登录会话创建失败")
	}
	files, _ := loadFiles(ctx, []string{user.AvatarFileID})
	return success(&fitnessResp.Login{
		Token:     token,
		ExpiresAt: expiresAt.Unix(),
		User:      userResponse(user, files),
	})
}

func (s *Service) GetProfile(ctx context.Context, userID string) *ginResp.Response {
	user, err := shared.EntClient.SysUser.Get(ctx, userID)
	if err != nil {
		return failed(fitnessResp.ResourceNotFound, "用户不存在")
	}
	files, err := loadFiles(ctx, []string{user.AvatarFileID})
	if err != nil {
		return failed(ginResp.OperationFailed, "头像信息读取失败")
	}
	checkins, err := shared.EntClient.FitnessCheckin.Query().
		Where(fitnesscheckin.UserIDEQ(userID)).
		Order(ent.Asc(fitnesscheckin.FieldCheckinDate)).
		All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "打卡统计读取失败")
	}
	dates := make([]string, 0, len(checkins))
	durations := make([]int, 0, len(checkins))
	for _, item := range checkins {
		dates = append(dates, item.CheckinDate)
		durations = append(durations, item.DurationMinutes)
	}
	location := support.Location(global.Cfg.System.Timezone)
	return success(map[string]any{
		"user":  userResponse(user, files),
		"stats": support.CalculateStats(dates, durations, time.Now().In(location)),
	})
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, req *fitnessReq.UpdateProfile) *ginResp.Response {
	var oldAvatarFileID string
	if req.AvatarFileID != nil {
		user, err := shared.EntClient.SysUser.Get(ctx, userID)
		if err != nil {
			return failed(fitnessResp.ResourceNotFound, "用户不存在")
		}
		oldAvatarFileID = user.AvatarFileID
	}
	update := shared.EntClient.SysUser.UpdateOneID(userID)
	if req.Nickname != nil {
		status, err := s.auditUserText(ctx, userID, *req.Nickname)
		if err != nil {
			return failed(fitnessResp.WechatServiceError, "昵称安全审核暂时不可用")
		}
		if status != "approved" {
			return failed(fitnessResp.ContentRejected, "昵称未通过内容安全审核")
		}
		update.SetNickname(strings.TrimSpace(*req.Nickname))
	}
	if req.AvatarFileID != nil {
		if *req.AvatarFileID == "" {
			update.ClearAvatarFileID()
		} else {
			exists, err := shared.EntClient.SysStorageFile.Query().
				Where(
					sysstoragefile.IDEQ(*req.AvatarFileID),
					sysstoragefile.OwnerUserIDEQ(userID),
					sysstoragefile.PurposeEQ(sysstoragefile.PurposeAvatar),
					sysstoragefile.AuditStatusEQ(sysstoragefile.AuditStatusApproved),
				).
				Exist(ctx)
			if err != nil || !exists {
				return failed(fitnessResp.ResourceNotFound, "头像文件不存在")
			}
			update.SetAvatarFileID(*req.AvatarFileID)
		}
	}
	if _, err := update.Save(ctx); err != nil {
		shared.Logger.Errorf("update fitness profile failed: %s", err)
		return failed(ginResp.OperationFailed, "资料保存失败")
	}
	if req.AvatarFileID != nil && oldAvatarFileID != "" && oldAvatarFileID != *req.AvatarFileID {
		if err := systemUpload.DeleteFiles(ctx, []string{oldAvatarFileID}); err != nil {
			shared.Logger.Errorf("delete replaced user avatar failed: %s", err)
		}
	}
	return s.GetProfile(ctx, userID)
}

func (s *Service) UpdateSettings(ctx context.Context, userID string, req *fitnessReq.UpdateSettings) *ginResp.Response {
	update := shared.EntClient.SysUser.UpdateOneID(userID).
		SetNillableReminderEnabled(req.ReminderEnabled)
	if _, err := update.Save(ctx); err != nil {
		return failed(ginResp.OperationFailed, "设置保存失败")
	}
	return s.GetProfile(ctx, userID)
}

func (s *Service) CancelAccount(ctx context.Context, userID string) *ginResp.Response {
	tx, err := shared.EntClient.Tx(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "注销事务启动失败")
	}
	defer func() { _ = tx.Rollback() }()
	owned, err := tx.FitnessGroup.Query().Where(fitnessgroup.OwnerIDEQ(userID), fitnessgroup.StatusEQ(fitnessgroup.StatusActive)).Exist(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "账号状态检查失败")
	}
	if owned {
		return failed(fitnessResp.PermissionDenied, "请先解散自己管理的小组")
	}

	checkins, err := tx.FitnessCheckin.Query().Where(fitnesscheckin.UserIDEQ(userID)).All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "打卡数据读取失败")
	}
	checkinIDs := make([]string, 0, len(checkins))
	for _, checkin := range checkins {
		checkinIDs = append(checkinIDs, checkin.ID)
	}

	ownedGroups, err := tx.FitnessGroup.Query().Where(fitnessgroup.OwnerIDEQ(userID)).All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "历史小组读取失败")
	}
	groupIDs := make([]string, 0, len(ownedGroups))
	for _, group := range ownedGroups {
		groupIDs = append(groupIDs, group.ID)
	}

	files, err := tx.SysStorageFile.Query().Where(sysstoragefile.OwnerUserIDEQ(userID)).All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "账号文件读取失败")
	}
	fileKeys := make([]string, 0, len(files))
	for _, file := range files {
		fileKeys = append(fileKeys, file.Key)
	}

	if len(checkinIDs) > 0 {
		if _, err = tx.FitnessContentReport.Delete().Where(fitnesscontentreport.CheckinIDIn(checkinIDs...)).Exec(ctx); err != nil {
			return failed(ginResp.OperationFailed, "打卡举报清理失败")
		}
		if _, err = tx.FitnessCheckinGroup.Delete().Where(fitnesscheckingroup.CheckinIDIn(checkinIDs...)).Exec(ctx); err != nil {
			return failed(ginResp.OperationFailed, "打卡小组关系清理失败")
		}
		if _, err = tx.FitnessCheckinImage.Delete().Where(fitnesscheckinimage.CheckinIDIn(checkinIDs...)).Exec(ctx); err != nil {
			return failed(ginResp.OperationFailed, "打卡图片关系清理失败")
		}
		if _, err = tx.FitnessCheckin.Delete().Where(fitnesscheckin.IDIn(checkinIDs...)).Exec(ctx); err != nil {
			return failed(ginResp.OperationFailed, "打卡数据清理失败")
		}
	}
	if len(groupIDs) > 0 {
		if _, err = tx.FitnessCheckinGroup.Delete().Where(fitnesscheckingroup.GroupIDIn(groupIDs...)).Exec(ctx); err != nil {
			return failed(ginResp.OperationFailed, "历史小组打卡关系清理失败")
		}
		if _, err = tx.FitnessGroupInvitation.Delete().Where(fitnessgroupinvitation.GroupIDIn(groupIDs...)).Exec(ctx); err != nil {
			return failed(ginResp.OperationFailed, "历史小组邀请清理失败")
		}
		if _, err = tx.FitnessGroupMember.Delete().Where(fitnessgroupmember.GroupIDIn(groupIDs...)).Exec(ctx); err != nil {
			return failed(ginResp.OperationFailed, "历史小组成员清理失败")
		}
		if _, err = tx.FitnessGroup.Delete().Where(fitnessgroup.IDIn(groupIDs...)).Exec(ctx); err != nil {
			return failed(ginResp.OperationFailed, "历史小组清理失败")
		}
	}
	if _, err = tx.FitnessGroupMember.Delete().Where(fitnessgroupmember.UserIDEQ(userID)).Exec(ctx); err != nil {
		return failed(ginResp.OperationFailed, "成员关系清理失败")
	}
	if _, err = tx.FitnessGroupInvitation.Delete().Where(fitnessgroupinvitation.CreatorIDEQ(userID)).Exec(ctx); err != nil {
		return failed(ginResp.OperationFailed, "邀请记录清理失败")
	}
	if _, err = tx.FitnessSubscriptionAuthorization.Delete().Where(fitnesssubscriptionauthorization.UserIDEQ(userID)).Exec(ctx); err != nil {
		return failed(ginResp.OperationFailed, "订阅授权清理失败")
	}
	if _, err = tx.FitnessContentReport.Delete().Where(fitnesscontentreport.ReporterUserIDEQ(userID)).Exec(ctx); err != nil {
		return failed(ginResp.OperationFailed, "举报记录清理失败")
	}
	if _, err = tx.FitnessReminderLog.Delete().Where(fitnessreminderlog.TargetUserIDEQ(userID)).Exec(ctx); err != nil {
		return failed(ginResp.OperationFailed, "提醒记录清理失败")
	}
	if _, err = tx.FitnessReminderLog.Delete().Where(fitnessreminderlog.SenderUserIDEQ(userID)).Exec(ctx); err != nil {
		return failed(ginResp.OperationFailed, "提醒发起记录清理失败")
	}
	if _, err = tx.SysStorageFile.Delete().Where(sysstoragefile.OwnerUserIDEQ(userID)).Exec(ctx); err != nil {
		return failed(ginResp.OperationFailed, "文件记录清理失败")
	}
	if err = tx.SysUser.DeleteOneID(userID).Exec(ctx); err != nil {
		return failed(ginResp.OperationFailed, "账号注销失败")
	}
	if err = tx.Commit(); err != nil {
		return failed(ginResp.OperationFailed, "账号注销提交失败")
	}
	support.RevokeUserSessions(ctx, userID)
	if len(fileKeys) > 0 {
		if err = internalUpload.NewOss().DeleteFiles(fileKeys); err != nil {
			shared.Logger.Errorf("delete cancelled user objects failed: %s", err)
		}
	}
	return success(nil)
}

func (s *Service) UploadImage(ctx context.Context, userID, purpose string, fileHeader *multipart.FileHeader) *ginResp.Response {
	allowed := map[string]sysstoragefile.Purpose{
		"avatar":  sysstoragefile.PurposeAvatar,
		"checkin": sysstoragefile.PurposeCheckin,
	}
	purposeValue, ok := allowed[purpose]
	if !ok {
		return failed(ginResp.ReqParameterException, "文件用途不正确")
	}
	if fileHeader.Size <= 0 || fileHeader.Size > 10*1024*1024 {
		return failed(ginResp.ReqParameterException, "图片大小必须在 10MB 以内")
	}
	file, err := fileHeader.Open()
	if err != nil {
		return failed(ginResp.OperationFailed, "图片读取失败")
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return failed(ginResp.ReqParameterException, "仅支持 JPG、PNG 或 WebP 图片")
	}
	stored, err := systemUpload.UploadFile(ctx, file, "fitness/"+userID+"/"+fileHeader.Filename, systemUpload.UploadOptions{
		OwnerUserID: userID,
		Purpose:     string(purposeValue),
		Size:        uint64(fileHeader.Size),
		Type:        systemUpload.TypeImg,
	})
	if err != nil {
		shared.Logger.Errorf("upload fitness image failed: %s", err)
		return failed(ginResp.FileUploadFailed, "图片上传失败")
	}
	user, err := shared.EntClient.SysUser.Get(ctx, userID)
	if err != nil {
		_ = systemUpload.DeleteFiles(ctx, []string{stored.ID})
		return failed(ginResp.OperationFailed, "用户读取失败")
	}
	if global.Cfg.System.DevelopmentMode {
		stored, err = shared.EntClient.SysStorageFile.UpdateOneID(stored.ID).
			SetAuditStatus(sysstoragefile.AuditStatusApproved).
			Save(ctx)
		if err != nil {
			_ = systemUpload.DeleteFiles(ctx, []string{stored.ID})
			return failed(ginResp.OperationFailed, "图片审核状态保存失败")
		}
		return success(privateFileResponse(stored))
	}
	publicURL := auditFileURL(stored)
	traceID, err := support.CheckImage(ctx, user.Openid, publicURL)
	if err != nil {
		_ = systemUpload.DeleteFiles(ctx, []string{stored.ID})
		shared.Logger.Errorf("start image security review failed: %s", err)
		return failed(fitnessResp.WechatServiceError, "图片安全审核暂时不可用")
	}
	stored, err = shared.EntClient.SysStorageFile.UpdateOneID(stored.ID).SetAuditTraceID(traceID).Save(ctx)
	if err != nil {
		_ = systemUpload.DeleteFiles(ctx, []string{stored.ID})
		return failed(ginResp.OperationFailed, "图片审核状态保存失败")
	}
	return success(privateFileResponse(stored))
}

func (s *Service) FileStatus(ctx context.Context, userID, fileID string) *ginResp.Response {
	file, err := shared.EntClient.SysStorageFile.Query().
		Where(sysstoragefile.IDEQ(fileID), sysstoragefile.OwnerUserIDEQ(userID)).
		Only(ctx)
	if err != nil {
		return failed(fitnessResp.ResourceNotFound, "图片不存在")
	}
	result := map[string]any{"id": file.ID, "auditStatus": string(file.AuditStatus)}
	if response := privateFileResponse(file); response != nil {
		result["file"] = response
	}
	return success(result)
}

func (s *Service) auditUserText(ctx context.Context, userID, content string) (string, error) {
	user, err := shared.EntClient.SysUser.Get(ctx, userID)
	if err != nil {
		return "", err
	}
	return support.CheckText(ctx, user.Openid, content)
}

func nonEmpty(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func invitationCode() (string, error) {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return strings.TrimRight(base64.RawURLEncoding.EncodeToString(raw), "="), nil
}

func createInvitationQRCode(ctx context.Context, userID, code string) (*ent.SysStorageFile, error) {
	image, err := support.GenerateInviteQRCode(ctx, code)
	if err != nil {
		return nil, err
	}
	file, err := systemUpload.UploadFile(ctx, bytes.NewReader(image), "fitness/invitations/"+code+".png", systemUpload.UploadOptions{
		OwnerUserID: userID,
		Purpose:     string(sysstoragefile.PurposeInvitationQr),
		Size:        uint64(len(image)),
		Type:        systemUpload.TypeImg,
	})
	if err != nil {
		return nil, err
	}
	file, err = shared.EntClient.SysStorageFile.UpdateOneID(file.ID).
		SetAuditStatus(sysstoragefile.AuditStatusApproved).
		Save(ctx)
	if err != nil {
		_ = systemUpload.DeleteFiles(ctx, []string{file.ID})
		return nil, err
	}
	return file, nil
}

var errPermission = errors.New("permission denied")

func ensureGroupAdmin(ctx context.Context, groupID, userID string) (*ent.FitnessGroup, error) {
	group, err := shared.EntClient.FitnessGroup.Get(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group.OwnerID != userID || group.Status != fitnessgroup.StatusActive {
		return nil, errPermission
	}
	return group, nil
}

func (s *Service) CreateGroup(ctx context.Context, userID string, req *fitnessReq.CreateGroup) *ginResp.Response {
	if !support.ValidReminderTime(req.ReminderTime) {
		return failed(ginResp.ReqParameterException, "提醒时间格式应为 HH:mm")
	}
	code, err := invitationCode()
	if err != nil {
		return failed(ginResp.OperationFailed, "邀请码生成失败")
	}
	qrFile, err := createInvitationQRCode(ctx, userID, code)
	if err != nil {
		shared.Logger.Errorf("create invitation qr code failed: %s", err)
		return failed(fitnessResp.WechatServiceError, "邀请二维码生成失败")
	}
	tx, err := shared.EntClient.Tx(ctx)
	if err != nil {
		_ = systemUpload.DeleteFiles(ctx, []string{qrFile.ID})
		return failed(ginResp.OperationFailed, "创建小组事务启动失败")
	}
	committed := false
	defer func() {
		_ = tx.Rollback()
		if !committed {
			_ = systemUpload.DeleteFiles(ctx, []string{qrFile.ID})
		}
	}()
	group, err := tx.FitnessGroup.Create().
		SetOwnerID(userID).
		SetName(support.PublicGroupName(req.Name)).
		SetWeeklyTarget(req.WeeklyTarget).
		SetReminderTime(req.ReminderTime).
		SetRequireApproval(req.RequireApproval).
		SetMemberLimit(req.MemberLimit).
		Save(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "小组创建失败")
	}
	if _, err = tx.FitnessGroupMember.Create().
		SetGroupID(group.ID).
		SetUserID(userID).
		SetRole(fitnessgroupmember.RoleAdmin).
		SetStatus(fitnessgroupmember.StatusActive).
		Save(ctx); err != nil {
		return failed(ginResp.OperationFailed, "管理员成员关系创建失败")
	}
	invitation, err := tx.FitnessGroupInvitation.Create().
		SetGroupID(group.ID).
		SetCreatorID(userID).
		SetCode(code).
		SetQrFileID(qrFile.ID).
		SetExpiresAt(time.Now().Add(7 * 24 * time.Hour)).
		Save(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "邀请创建失败")
	}
	if err = tx.Commit(); err != nil {
		return failed(ginResp.OperationFailed, "小组创建提交失败")
	}
	committed = true
	return success(map[string]any{"id": group.ID, "invitationCode": invitation.Code, "qrCode": fileResponse(qrFile)})
}

func (s *Service) UpdateGroup(ctx context.Context, userID string, req *fitnessReq.UpdateGroup) *ginResp.Response {
	group, err := ensureGroupAdmin(ctx, req.ID, userID)
	if err != nil {
		return failed(fitnessResp.PermissionDenied, "只有小组管理员可以修改")
	}
	if req.ReminderTime != nil && !support.ValidReminderTime(*req.ReminderTime) {
		return failed(ginResp.ReqParameterException, "提醒时间格式应为 HH:mm")
	}
	if req.Name != nil {
		name := support.PublicGroupName(*req.Name)
		req.Name = &name
	}
	update := shared.EntClient.FitnessGroup.UpdateOneID(req.ID).
		SetNillableName(req.Name).
		SetDescription("").
		SetAnnouncement("").
		ClearAvatarFileID().
		SetNillableWeeklyTarget(req.WeeklyTarget).
		SetNillableReminderTime(req.ReminderTime).
		SetNillableRequireApproval(req.RequireApproval).
		SetNillableMemberLimit(req.MemberLimit)
	if _, err := update.Save(ctx); err != nil {
		return failed(ginResp.OperationFailed, "小组保存失败")
	}
	if group.AvatarFileID != "" {
		if err := systemUpload.DeleteFiles(ctx, []string{group.AvatarFileID}); err != nil {
			shared.Logger.Errorf("delete replaced group avatar failed: %s", err)
		}
	}
	return s.GetGroup(ctx, userID, req.ID)
}

func (s *Service) ListGroups(ctx context.Context, userID string) *ginResp.Response {
	memberships, err := shared.EntClient.FitnessGroupMember.Query().
		Where(fitnessgroupmember.UserIDEQ(userID), fitnessgroupmember.StatusIn(fitnessgroupmember.StatusActive, fitnessgroupmember.StatusPending)).
		Order(ent.Desc(fitnessgroupmember.FieldJoinedAt)).
		All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "小组列表读取失败")
	}
	groupIDs := make([]string, 0, len(memberships))
	roleByGroup := make(map[string]*ent.FitnessGroupMember, len(memberships))
	for _, member := range memberships {
		groupIDs = append(groupIDs, member.GroupID)
		roleByGroup[member.GroupID] = member
	}
	if len(groupIDs) == 0 {
		return success([]fitnessResp.GroupSummary{})
	}
	groups, err := shared.EntClient.FitnessGroup.Query().
		Where(fitnessgroup.IDIn(groupIDs...), fitnessgroup.StatusEQ(fitnessgroup.StatusActive)).
		Order(ent.Desc(fitnessgroup.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "小组信息读取失败")
	}
	allMembers, err := shared.EntClient.FitnessGroupMember.Query().
		Where(fitnessgroupmember.GroupIDIn(groupIDs...), fitnessgroupmember.StatusEQ(fitnessgroupmember.StatusActive)).
		All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "成员统计读取失败")
	}
	memberCount := make(map[string]int)
	for _, member := range allMembers {
		memberCount[member.GroupID]++
	}
	today := support.Today(support.Location(global.Cfg.System.Timezone))
	todayCheckins, err := shared.EntClient.FitnessCheckin.Query().
		Where(fitnesscheckin.CheckinDateEQ(today), fitnesscheckin.AuditStatusNEQ(fitnesscheckin.AuditStatusRejected)).
		All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "今日打卡读取失败")
	}
	checkinIDs := make([]string, 0, len(todayCheckins))
	ownerByCheckin := make(map[string]string, len(todayCheckins))
	for _, checkin := range todayCheckins {
		checkinIDs = append(checkinIDs, checkin.ID)
		ownerByCheckin[checkin.ID] = checkin.UserID
	}
	checkedCount := make(map[string]int)
	currentChecked := make(map[string]bool)
	if len(checkinIDs) > 0 {
		relations, queryErr := shared.EntClient.FitnessCheckinGroup.Query().
			Where(fitnesscheckingroup.CheckinIDIn(checkinIDs...), fitnesscheckingroup.GroupIDIn(groupIDs...)).
			All(ctx)
		if queryErr != nil {
			return failed(ginResp.OperationFailed, "小组打卡统计失败")
		}
		for _, relation := range relations {
			checkedCount[relation.GroupID]++
			if ownerByCheckin[relation.CheckinID] == userID {
				currentChecked[relation.GroupID] = true
			}
		}
	}
	result := make([]fitnessResp.GroupSummary, 0, len(groups))
	for _, group := range groups {
		membership := roleByGroup[group.ID]
		result = append(result, fitnessResp.GroupSummary{
			ID: group.ID, Name: support.PublicGroupName(group.Name),
			MemberCount:  memberCount[group.ID],
			CheckedCount: checkedCount[group.ID], CurrentChecked: currentChecked[group.ID],
			Role: string(membership.Role), MembershipStatus: string(membership.Status),
		})
	}
	return success(result)
}

func (s *Service) CreateInvitation(ctx context.Context, userID string, req *fitnessReq.CreateInvitation) *ginResp.Response {
	group, err := ensureGroupAdmin(ctx, req.GroupID, userID)
	if err != nil {
		return failed(fitnessResp.PermissionDenied, "只有小组管理员可以创建邀请")
	}
	hours := req.ValidHours
	if hours == 0 {
		hours = 168
	}
	oldInvitations, err := shared.EntClient.FitnessGroupInvitation.Query().
		Where(fitnessgroupinvitation.GroupIDEQ(group.ID), fitnessgroupinvitation.ActiveEQ(true)).
		All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "旧邀请读取失败")
	}
	oldQRFileIDs := make([]string, 0, len(oldInvitations))
	for _, invitation := range oldInvitations {
		oldQRFileIDs = append(oldQRFileIDs, invitation.QrFileID)
	}
	code, err := invitationCode()
	if err != nil {
		return failed(ginResp.OperationFailed, "邀请码生成失败")
	}
	qrFile, err := createInvitationQRCode(ctx, userID, code)
	if err != nil {
		shared.Logger.Errorf("create invitation qr code failed: %s", err)
		return failed(fitnessResp.WechatServiceError, "邀请二维码生成失败")
	}
	tx, err := shared.EntClient.Tx(ctx)
	if err != nil {
		_ = systemUpload.DeleteFiles(ctx, []string{qrFile.ID})
		return failed(ginResp.OperationFailed, "邀请事务启动失败")
	}
	committed := false
	defer func() {
		_ = tx.Rollback()
		if !committed {
			_ = systemUpload.DeleteFiles(ctx, []string{qrFile.ID})
		}
	}()
	if _, err = tx.FitnessGroupInvitation.Update().
		Where(fitnessgroupinvitation.GroupIDEQ(group.ID), fitnessgroupinvitation.ActiveEQ(true)).
		SetActive(false).Save(ctx); err != nil {
		return failed(ginResp.OperationFailed, "旧邀请失效失败")
	}
	invitation, err := tx.FitnessGroupInvitation.Create().
		SetGroupID(group.ID).SetCreatorID(userID).SetCode(code).
		SetQrFileID(qrFile.ID).
		SetExpiresAt(time.Now().Add(time.Duration(hours) * time.Hour)).Save(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "邀请创建失败")
	}
	if err = tx.Commit(); err != nil {
		return failed(ginResp.OperationFailed, "邀请提交失败")
	}
	committed = true
	if err = systemUpload.DeleteFiles(ctx, oldQRFileIDs); err != nil {
		shared.Logger.Errorf("delete expired invitation QR files failed: %s", err)
	}
	return success(map[string]any{"code": invitation.Code, "expiresAt": invitation.ExpiresAt.Unix(), "qrCode": fileResponse(qrFile)})
}

func (s *Service) GetInvitation(ctx context.Context, userID, groupID string) *ginResp.Response {
	member, err := shared.EntClient.FitnessGroupMember.Query().
		Where(
			fitnessgroupmember.GroupIDEQ(groupID),
			fitnessgroupmember.UserIDEQ(userID),
			fitnessgroupmember.StatusEQ(fitnessgroupmember.StatusActive),
		).
		Only(ctx)
	if err != nil || member == nil {
		return failed(fitnessResp.PermissionDenied, "你还不是该小组成员")
	}
	invitation, err := shared.EntClient.FitnessGroupInvitation.Query().
		Where(
			fitnessgroupinvitation.GroupIDEQ(groupID),
			fitnessgroupinvitation.ActiveEQ(true),
			fitnessgroupinvitation.ExpiresAtGT(time.Now()),
		).
		Order(ent.Desc(fitnessgroupinvitation.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		return failed(fitnessResp.InvitationInvalid, "当前没有可用邀请")
	}
	files, err := loadFiles(ctx, []string{invitation.QrFileID})
	if err != nil {
		return failed(ginResp.OperationFailed, "邀请二维码读取失败")
	}
	return success(map[string]any{
		"code": invitation.Code, "expiresAt": invitation.ExpiresAt.Unix(),
		"qrCode": fileResponse(files[invitation.QrFileID]),
	})
}

func (s *Service) InvitationPreview(ctx context.Context, code string) *ginResp.Response {
	invitation, err := shared.EntClient.FitnessGroupInvitation.Query().
		Where(fitnessgroupinvitation.CodeEQ(code), fitnessgroupinvitation.ActiveEQ(true), fitnessgroupinvitation.ExpiresAtGT(time.Now())).
		Only(ctx)
	if err != nil {
		return failed(fitnessResp.InvitationInvalid, "邀请已失效")
	}
	group, err := shared.EntClient.FitnessGroup.Get(ctx, invitation.GroupID)
	if err != nil || group.Status != fitnessgroup.StatusActive {
		return failed(fitnessResp.InvitationInvalid, "小组已不可加入")
	}
	count, err := shared.EntClient.FitnessGroupMember.Query().
		Where(fitnessgroupmember.GroupIDEQ(group.ID), fitnessgroupmember.StatusEQ(fitnessgroupmember.StatusActive)).
		Count(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "小组人数读取失败")
	}
	return success(fitnessResp.Invitation{
		Code: code, ExpiresAt: invitation.ExpiresAt.Unix(), GroupID: group.ID,
		GroupName:   support.PublicGroupName(group.Name),
		MemberCount: count, WeeklyTarget: group.WeeklyTarget, RequireApproval: group.RequireApproval,
	})
}

func (s *Service) JoinGroup(ctx context.Context, userID string, req *fitnessReq.JoinGroup) *ginResp.Response {
	invitation, err := shared.EntClient.FitnessGroupInvitation.Query().
		Where(fitnessgroupinvitation.CodeEQ(req.Code), fitnessgroupinvitation.ActiveEQ(true), fitnessgroupinvitation.ExpiresAtGT(time.Now())).
		Only(ctx)
	if err != nil {
		return failed(fitnessResp.InvitationInvalid, "邀请已失效")
	}
	group, err := shared.EntClient.FitnessGroup.Get(ctx, invitation.GroupID)
	if err != nil || group.Status != fitnessgroup.StatusActive {
		return failed(fitnessResp.InvitationInvalid, "小组已不可加入")
	}
	existing, err := shared.EntClient.FitnessGroupMember.Query().
		Where(fitnessgroupmember.GroupIDEQ(group.ID), fitnessgroupmember.UserIDEQ(userID)).
		Only(ctx)
	if err == nil && existing.Status != fitnessgroupmember.StatusRejected {
		return failed(fitnessResp.AlreadyJoined, "你已经加入或正在等待审核")
	}
	if err != nil && !ent.IsNotFound(err) {
		return failed(ginResp.OperationFailed, "成员关系检查失败")
	}
	count, err := shared.EntClient.FitnessGroupMember.Query().
		Where(fitnessgroupmember.GroupIDEQ(group.ID), fitnessgroupmember.StatusIn(fitnessgroupmember.StatusActive, fitnessgroupmember.StatusPending)).
		Count(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "小组人数读取失败")
	}
	if count >= group.MemberLimit {
		return failed(fitnessResp.GroupFull, "小组人数已满")
	}
	status := fitnessgroupmember.StatusActive
	if group.RequireApproval {
		status = fitnessgroupmember.StatusPending
	}
	if existing != nil {
		_, err = shared.EntClient.FitnessGroupMember.UpdateOneID(existing.ID).SetStatus(status).SetJoinedAt(time.Now()).Save(ctx)
	} else {
		_, err = shared.EntClient.FitnessGroupMember.Create().
			SetGroupID(group.ID).SetUserID(userID).SetRole(fitnessgroupmember.RoleMember).SetStatus(status).Save(ctx)
	}
	if err != nil {
		return failed(ginResp.OperationFailed, "加入小组失败")
	}
	return success(map[string]any{"groupId": group.ID, "status": string(status)})
}

func (s *Service) ReviewMember(ctx context.Context, userID, groupID string, req *fitnessReq.ReviewMember) *ginResp.Response {
	if _, err := ensureGroupAdmin(ctx, groupID, userID); err != nil {
		return failed(fitnessResp.PermissionDenied, "只有小组管理员可以审核成员")
	}
	member, err := shared.EntClient.FitnessGroupMember.Query().
		Where(fitnessgroupmember.IDEQ(req.MemberID), fitnessgroupmember.GroupIDEQ(groupID), fitnessgroupmember.StatusEQ(fitnessgroupmember.StatusPending)).
		Only(ctx)
	if err != nil {
		return failed(fitnessResp.ResourceNotFound, "待审核成员不存在")
	}
	status := fitnessgroupmember.StatusRejected
	if req.Approve {
		status = fitnessgroupmember.StatusActive
	}
	if _, err = shared.EntClient.FitnessGroupMember.UpdateOneID(member.ID).SetStatus(status).Save(ctx); err != nil {
		return failed(ginResp.OperationFailed, "审核保存失败")
	}
	return success(nil)
}

func (s *Service) LeaveGroup(ctx context.Context, userID, groupID string) *ginResp.Response {
	group, err := shared.EntClient.FitnessGroup.Get(ctx, groupID)
	if err != nil {
		return failed(fitnessResp.ResourceNotFound, "小组不存在")
	}
	if group.OwnerID == userID {
		return failed(fitnessResp.PermissionDenied, "管理员不能直接退出，请先解散小组")
	}
	deleted, err := shared.EntClient.FitnessGroupMember.Delete().
		Where(fitnessgroupmember.GroupIDEQ(groupID), fitnessgroupmember.UserIDEQ(userID)).
		Exec(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "退出小组失败")
	}
	if deleted == 0 {
		return failed(fitnessResp.ResourceNotFound, "成员关系不存在")
	}
	return success(nil)
}

func (s *Service) RemoveMember(ctx context.Context, userID, groupID, memberID string) *ginResp.Response {
	group, err := ensureGroupAdmin(ctx, groupID, userID)
	if err != nil {
		return failed(fitnessResp.PermissionDenied, "只有小组管理员可以移除成员")
	}
	member, err := shared.EntClient.FitnessGroupMember.Query().
		Where(fitnessgroupmember.IDEQ(memberID), fitnessgroupmember.GroupIDEQ(groupID)).
		Only(ctx)
	if err != nil {
		return failed(fitnessResp.ResourceNotFound, "成员不存在")
	}
	if member.UserID == group.OwnerID {
		return failed(fitnessResp.PermissionDenied, "不能移除小组管理员")
	}
	if err = shared.EntClient.FitnessGroupMember.DeleteOneID(member.ID).Exec(ctx); err != nil {
		return failed(ginResp.OperationFailed, "成员移除失败")
	}
	return success(nil)
}

func (s *Service) DissolveGroup(ctx context.Context, userID, groupID string) *ginResp.Response {
	if _, err := ensureGroupAdmin(ctx, groupID, userID); err != nil {
		return failed(fitnessResp.PermissionDenied, "只有小组管理员可以解散")
	}
	if err := dissolveGroup(ctx, groupID); err != nil {
		return failed(ginResp.OperationFailed, "小组解散失败")
	}
	return success(nil)
}

func (s *Service) GetGroup(ctx context.Context, userID, groupID string) *ginResp.Response {
	group, err := shared.EntClient.FitnessGroup.Get(ctx, groupID)
	if err != nil || group.Status != fitnessgroup.StatusActive {
		return failed(fitnessResp.ResourceNotFound, "小组不存在")
	}
	currentMembership, err := shared.EntClient.FitnessGroupMember.Query().
		Where(fitnessgroupmember.GroupIDEQ(groupID), fitnessgroupmember.UserIDEQ(userID)).
		Only(ctx)
	if err != nil || currentMembership.Status == fitnessgroupmember.StatusRejected {
		return failed(fitnessResp.PermissionDenied, "你还不是该小组成员")
	}
	if currentMembership.Status == fitnessgroupmember.StatusPending {
		count, countErr := shared.EntClient.FitnessGroupMember.Query().
			Where(
				fitnessgroupmember.GroupIDEQ(groupID),
				fitnessgroupmember.StatusEQ(fitnessgroupmember.StatusActive),
			).
			Count(ctx)
		if countErr != nil {
			return failed(ginResp.OperationFailed, "小组人数读取失败")
		}
		return success(fitnessResp.GroupDetail{
			GroupSummary: fitnessResp.GroupSummary{
				ID: group.ID, Name: support.PublicGroupName(group.Name),
				MemberCount: count,
				Role:        string(currentMembership.Role), MembershipStatus: string(currentMembership.Status),
			},
			WeeklyTarget: group.WeeklyTarget, ReminderTime: group.ReminderTime,
			RequireApproval: group.RequireApproval, MemberLimit: group.MemberLimit,
			CheckedMembers: []fitnessResp.MemberStatus{}, PendingMembers: []fitnessResp.MemberStatus{},
			UncheckedMembers: []fitnessResp.MemberStatus{},
		})
	}
	memberQuery := shared.EntClient.FitnessGroupMember.Query().
		Where(fitnessgroupmember.GroupIDEQ(groupID), fitnessgroupmember.StatusEQ(fitnessgroupmember.StatusActive))
	if currentMembership.Role == fitnessgroupmember.RoleAdmin {
		memberQuery = shared.EntClient.FitnessGroupMember.Query().
			Where(
				fitnessgroupmember.GroupIDEQ(groupID),
				fitnessgroupmember.StatusIn(fitnessgroupmember.StatusActive, fitnessgroupmember.StatusPending),
			)
	}
	members, err := memberQuery.Order(ent.Asc(fitnessgroupmember.FieldJoinedAt)).All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "成员列表读取失败")
	}
	userIDs := make([]string, 0, len(members))
	for _, member := range members {
		userIDs = append(userIDs, member.UserID)
	}
	today := support.Today(support.Location(global.Cfg.System.Timezone))
	todayCheckins, err := shared.EntClient.FitnessCheckin.Query().
		Where(
			fitnesscheckin.UserIDIn(userIDs...),
			fitnesscheckin.CheckinDateEQ(today),
			fitnesscheckin.AuditStatusNEQ(fitnesscheckin.AuditStatusRejected),
		).
		All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "今日完成状态读取失败")
	}
	todayByID := make(map[string]*ent.FitnessCheckin)
	todayIDs := make([]string, 0, len(todayCheckins))
	for _, checkin := range todayCheckins {
		todayByID[checkin.ID] = checkin
		todayIDs = append(todayIDs, checkin.ID)
	}
	publishedByUser := make(map[string]*ent.FitnessCheckin)
	if len(todayIDs) > 0 {
		relations, queryErr := shared.EntClient.FitnessCheckinGroup.Query().
			Where(fitnesscheckingroup.CheckinIDIn(todayIDs...), fitnesscheckingroup.GroupIDEQ(groupID)).
			All(ctx)
		if queryErr != nil {
			return failed(ginResp.OperationFailed, "今日发布状态读取失败")
		}
		for _, relation := range relations {
			checkin := todayByID[relation.CheckinID]
			if checkin != nil {
				publishedByUser[checkin.UserID] = checkin
			}
		}
	}
	checked := make([]fitnessResp.MemberStatus, 0)
	unchecked := make([]fitnessResp.MemberStatus, 0)
	pending := make([]fitnessResp.MemberStatus, 0)
	for index, member := range members {
		label := "成员 " + strconv.Itoa(index+1)
		if member.UserID == userID {
			label = "我"
		}
		item := fitnessResp.MemberStatus{
			MemberID: member.ID, Label: label,
			Role: string(member.Role), Status: string(member.Status), IsCurrent: member.UserID == userID,
		}
		if member.Status == fitnessgroupmember.StatusPending {
			pending = append(pending, item)
			continue
		}
		if publishedByUser[member.UserID] != nil {
			item.Checked = true
			checked = append(checked, item)
		} else {
			unchecked = append(unchecked, item)
		}
	}
	summary := fitnessResp.GroupSummary{
		ID: group.ID, Name: support.PublicGroupName(group.Name),
		MemberCount:  len(checked) + len(unchecked),
		CheckedCount: len(checked), CurrentChecked: publishedByUser[userID] != nil,
		Role: string(currentMembership.Role), MembershipStatus: string(currentMembership.Status),
	}
	return success(fitnessResp.GroupDetail{
		GroupSummary: summary,
		WeeklyTarget: group.WeeklyTarget, ReminderTime: group.ReminderTime,
		RequireApproval: group.RequireApproval, MemberLimit: group.MemberLimit,
		CheckedMembers: checked, PendingMembers: pending, UncheckedMembers: unchecked,
	})
}

func (s *Service) UpsertTodayCheckin(ctx context.Context, userID string, req *fitnessReq.UpsertCheckin) *ginResp.Response {
	groupIDs := uniqueNonEmpty(req.GroupIDs)
	if len(groupIDs) > 0 {
		memberCount, err := shared.EntClient.FitnessGroupMember.Query().
			Where(
				fitnessgroupmember.UserIDEQ(userID),
				fitnessgroupmember.GroupIDIn(groupIDs...),
				fitnessgroupmember.StatusEQ(fitnessgroupmember.StatusActive),
			).Count(ctx)
		if err != nil || memberCount != len(groupIDs) {
			return failed(fitnessResp.PermissionDenied, "只能发布到已加入的小组")
		}
	}
	imageIDs := uniqueNonEmpty(req.ImageFileIDs)
	imageFiles := make([]*ent.SysStorageFile, 0, len(imageIDs))
	var err error
	if len(imageIDs) > 0 {
		imageFiles, err = shared.EntClient.SysStorageFile.Query().
			Where(
				sysstoragefile.IDIn(imageIDs...),
				sysstoragefile.OwnerUserIDEQ(userID),
				sysstoragefile.PurposeEQ(sysstoragefile.PurposeCheckin),
				sysstoragefile.AuditStatusNEQ(sysstoragefile.AuditStatusRejected),
			).All(ctx)
		if err != nil || len(imageFiles) != len(imageIDs) {
			return failed(fitnessResp.ResourceNotFound, "部分打卡图片不存在或未通过审核")
		}
	}
	today := support.Today(support.Location(global.Cfg.System.Timezone))
	existing, err := shared.EntClient.FitnessCheckin.Query().
		Where(fitnesscheckin.UserIDEQ(userID), fitnesscheckin.CheckinDateEQ(today)).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return failed(ginResp.OperationFailed, "今日打卡读取失败")
	}
	tx, err := shared.EntClient.Tx(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "打卡事务启动失败")
	}
	defer func() { _ = tx.Rollback() }()
	var checkin *ent.FitnessCheckin
	if existing == nil {
		create := tx.FitnessCheckin.Create().
			SetUserID(userID).
			SetCheckinDate(today).
			SetExerciseType(fitnesscheckin.ExerciseType(req.ExerciseType)).
			SetDurationMinutes(req.DurationMinutes).
			SetContent("").
			SetWeightPublic(false).
			SetMood("").
			SetAuditStatus(fitnesscheckin.AuditStatusApproved)
		checkin, err = create.Save(ctx)
	} else {
		checkin, err = tx.FitnessCheckin.UpdateOneID(existing.ID).
			SetExerciseType(fitnesscheckin.ExerciseType(req.ExerciseType)).
			SetDurationMinutes(req.DurationMinutes).
			SetContent("").
			ClearCalories().
			ClearWeight().
			SetWeightPublic(false).
			SetMood("").
			SetAuditStatus(fitnesscheckin.AuditStatusApproved).
			Save(ctx)
	}
	if err != nil {
		return failed(ginResp.OperationFailed, "打卡保存失败")
	}
	if _, err = tx.FitnessCheckinGroup.Delete().Where(fitnesscheckingroup.CheckinIDEQ(checkin.ID)).Exec(ctx); err != nil {
		return failed(ginResp.OperationFailed, "发布小组更新失败")
	}
	relationCreates := make([]*ent.FitnessCheckinGroupCreate, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		relationCreates = append(relationCreates, tx.FitnessCheckinGroup.Create().SetCheckinID(checkin.ID).SetGroupID(groupID))
	}
	if len(relationCreates) > 0 {
		if _, err = tx.FitnessCheckinGroup.CreateBulk(relationCreates...).Save(ctx); err != nil {
			return failed(ginResp.OperationFailed, "发布小组保存失败")
		}
	}
	oldImages, err := tx.FitnessCheckinImage.Query().Where(fitnesscheckinimage.CheckinIDEQ(checkin.ID)).All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "旧图片读取失败")
	}
	if _, err = tx.FitnessCheckinImage.Delete().Where(fitnesscheckinimage.CheckinIDEQ(checkin.ID)).Exec(ctx); err != nil {
		return failed(ginResp.OperationFailed, "旧图片关联清理失败")
	}
	imageCreates := make([]*ent.FitnessCheckinImageCreate, 0, len(imageFiles))
	for index, file := range imageFiles {
		imageCreates = append(imageCreates, tx.FitnessCheckinImage.Create().
			SetCheckinID(checkin.ID).
			SetStorageFileID(file.ID).
			SetSort(index).
			SetAuditStatus(fitnesscheckinimage.AuditStatus(file.AuditStatus)).
			SetAuditTraceID(file.AuditTraceID))
	}
	if len(imageCreates) > 0 {
		if _, err = tx.FitnessCheckinImage.CreateBulk(imageCreates...).Save(ctx); err != nil {
			return failed(ginResp.OperationFailed, "打卡图片关联失败")
		}
	}
	if err = tx.Commit(); err != nil {
		return failed(ginResp.OperationFailed, "打卡提交失败")
	}
	kept := make(map[string]struct{}, len(imageIDs))
	for _, id := range imageIDs {
		kept[id] = struct{}{}
	}
	unused := make([]string, 0)
	for _, old := range oldImages {
		if _, exists := kept[old.StorageFileID]; !exists {
			unused = append(unused, old.StorageFileID)
		}
	}
	if len(unused) > 0 {
		if deleteErr := systemUpload.DeleteFiles(ctx, unused); deleteErr != nil {
			shared.Logger.Errorf("delete replaced checkin images failed: %s", deleteErr)
		}
	}
	return s.GetCheckin(ctx, userID, checkin.ID)
}

func (s *Service) GetTodayCheckin(ctx context.Context, userID string) *ginResp.Response {
	today := support.Today(support.Location(global.Cfg.System.Timezone))
	checkin, err := shared.EntClient.FitnessCheckin.Query().
		Where(fitnesscheckin.UserIDEQ(userID), fitnesscheckin.CheckinDateEQ(today)).
		Only(ctx)
	if ent.IsNotFound(err) {
		return success(nil)
	}
	if err != nil {
		return failed(ginResp.OperationFailed, "今日打卡读取失败")
	}
	return s.GetCheckin(ctx, userID, checkin.ID)
}

func (s *Service) GetCheckin(ctx context.Context, viewerID, checkinID string) *ginResp.Response {
	checkin, err := shared.EntClient.FitnessCheckin.Get(ctx, checkinID)
	if err != nil || checkin.UserID != viewerID {
		return failed(fitnessResp.ResourceNotFound, "打卡不存在")
	}
	items, err := s.privateCheckinResponses(ctx, []*ent.FitnessCheckin{checkin})
	if err != nil {
		return failed(ginResp.OperationFailed, "打卡详情读取失败")
	}
	items[0].CanManage = true
	return success(items[0])
}

func (s *Service) Calendar(ctx context.Context, userID, month string) *ginResp.Response {
	location := support.Location(global.Cfg.System.Timezone)
	if month == "" {
		month = time.Now().In(location).Format("2006-01")
	}
	start, end, err := support.MonthBounds(month, location)
	if err != nil {
		return failed(ginResp.ReqParameterException, "月份格式应为 YYYY-MM")
	}
	items, err := shared.EntClient.FitnessCheckin.Query().
		Where(
			fitnesscheckin.UserIDEQ(userID),
			fitnesscheckin.CheckinDateGTE(start.Format(time.DateOnly)),
			fitnesscheckin.CheckinDateLT(end.Format(time.DateOnly)),
		).
		Order(ent.Asc(fitnesscheckin.FieldCheckinDate)).
		All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "月度打卡读取失败")
	}
	responses, err := s.privateCheckinResponses(ctx, items)
	if err != nil {
		return failed(ginResp.OperationFailed, "月度打卡详情读取失败")
	}
	for index := range responses {
		responses[index].CanManage = true
	}
	all, err := shared.EntClient.FitnessCheckin.Query().
		Where(fitnesscheckin.UserIDEQ(userID)).
		Order(ent.Asc(fitnesscheckin.FieldCheckinDate)).
		All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "累计统计读取失败")
	}
	dates := make([]string, 0, len(all))
	durations := make([]int, 0, len(all))
	for _, checkin := range all {
		dates = append(dates, checkin.CheckinDate)
		durations = append(durations, checkin.DurationMinutes)
	}
	return success(fitnessResp.Calendar{
		Month: month,
		Stats: support.CalculateStats(dates, durations, time.Now().In(location)),
		Items: responses,
	})
}

func (s *Service) DeleteCheckin(ctx context.Context, userID, checkinID string) *ginResp.Response {
	checkin, err := shared.EntClient.FitnessCheckin.Get(ctx, checkinID)
	if err != nil || checkin.UserID != userID {
		return failed(fitnessResp.ResourceNotFound, "打卡不存在")
	}
	images, err := shared.EntClient.FitnessCheckinImage.Query().Where(fitnesscheckinimage.CheckinIDEQ(checkinID)).All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "打卡图片读取失败")
	}
	tx, err := shared.EntClient.Tx(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "删除事务启动失败")
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.FitnessCheckinGroup.Delete().Where(fitnesscheckingroup.CheckinIDEQ(checkinID)).Exec(ctx); err != nil {
		return failed(ginResp.OperationFailed, "打卡发布关系删除失败")
	}
	if _, err = tx.FitnessCheckinImage.Delete().Where(fitnesscheckinimage.CheckinIDEQ(checkinID)).Exec(ctx); err != nil {
		return failed(ginResp.OperationFailed, "打卡图片关系删除失败")
	}
	if _, err = tx.FitnessContentReport.Delete().Where(fitnesscontentreport.CheckinIDEQ(checkinID)).Exec(ctx); err != nil {
		return failed(ginResp.OperationFailed, "打卡举报删除失败")
	}
	if err = tx.FitnessCheckin.DeleteOneID(checkinID).Exec(ctx); err != nil {
		return failed(ginResp.OperationFailed, "打卡删除失败")
	}
	if err = tx.Commit(); err != nil {
		return failed(ginResp.OperationFailed, "打卡删除提交失败")
	}
	fileIDs := make([]string, 0, len(images))
	for _, image := range images {
		fileIDs = append(fileIDs, image.StorageFileID)
	}
	if err = systemUpload.DeleteFiles(ctx, fileIDs); err != nil {
		shared.Logger.Errorf("delete checkin storage files failed: %s", err)
	}
	return success(nil)
}

func (s *Service) privateCheckinResponses(ctx context.Context, checkins []*ent.FitnessCheckin) ([]fitnessResp.Checkin, error) {
	if len(checkins) == 0 {
		return []fitnessResp.Checkin{}, nil
	}
	ids := make([]string, 0, len(checkins))
	for _, item := range checkins {
		ids = append(ids, item.ID)
	}
	relations, err := shared.EntClient.FitnessCheckinGroup.Query().
		Where(fitnesscheckingroup.CheckinIDIn(ids...)).All(ctx)
	if err != nil {
		return nil, err
	}
	groupIDs := make([]string, 0, len(relations))
	groupIDsByCheckin := make(map[string][]string)
	for _, relation := range relations {
		groupIDs = append(groupIDs, relation.GroupID)
		groupIDsByCheckin[relation.CheckinID] = append(groupIDsByCheckin[relation.CheckinID], relation.GroupID)
	}
	groups, err := shared.EntClient.FitnessGroup.Query().Where(fitnessgroup.IDIn(uniqueNonEmpty(groupIDs)...)).All(ctx)
	if err != nil {
		return nil, err
	}
	groupNames := make(map[string]string, len(groups))
	for _, group := range groups {
		groupNames[group.ID] = support.PublicGroupName(group.Name)
	}
	imageQuery := shared.EntClient.FitnessCheckinImage.Query().Where(fitnesscheckinimage.CheckinIDIn(ids...))
	images, err := imageQuery.Order(ent.Asc(fitnesscheckinimage.FieldSort)).All(ctx)
	if err != nil {
		return nil, err
	}
	imageIDs := make([]string, 0, len(images))
	imagesByCheckin := make(map[string][]*ent.FitnessCheckinImage)
	for _, image := range images {
		imageIDs = append(imageIDs, image.StorageFileID)
		imagesByCheckin[image.CheckinID] = append(imagesByCheckin[image.CheckinID], image)
	}
	files, err := loadFiles(ctx, imageIDs)
	if err != nil {
		return nil, err
	}
	result := make([]fitnessResp.Checkin, 0, len(checkins))
	for _, item := range checkins {
		responseItem := fitnessResp.Checkin{
			ID: item.ID, Date: item.CheckinDate, ExerciseType: string(item.ExerciseType),
			DurationMinutes: item.DurationMinutes, AuditStatus: string(item.AuditStatus),
			AuditDetail: item.AuditDetail,
			Images:      []fitnessResp.File{},
			GroupIDs:    groupIDsByCheckin[item.ID], GroupNames: []string{},
		}
		responseItem.ImageAuditSummary = map[string]int{"approved": 0, "pending": 0, "rejected": 0}
		for _, groupID := range responseItem.GroupIDs {
			if name := groupNames[groupID]; name != "" {
				responseItem.GroupNames = append(responseItem.GroupNames, name)
			}
		}
		for _, image := range imagesByCheckin[item.ID] {
			responseItem.ImageAuditSummary[string(image.AuditStatus)]++
			var file *fitnessResp.File
			if image.AuditStatus != fitnesscheckinimage.AuditStatusRejected {
				file = privateFileResponse(files[image.StorageFileID])
			}
			if file != nil {
				responseItem.Images = append(responseItem.Images, *file)
			}
		}
		result = append(result, responseItem)
	}
	return result, nil
}

func (s *Service) Home(ctx context.Context, userID string) *ginResp.Response {
	location := support.Location(global.Cfg.System.Timezone)
	today := support.Today(location)
	all, err := shared.EntClient.FitnessCheckin.Query().
		Where(fitnesscheckin.UserIDEQ(userID), fitnesscheckin.AuditStatusNEQ(fitnesscheckin.AuditStatusRejected)).
		Order(ent.Asc(fitnesscheckin.FieldCheckinDate)).
		All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "首页统计读取失败")
	}
	dates := make([]string, 0, len(all))
	durations := make([]int, 0, len(all))
	todayChecked := false
	for _, item := range all {
		dates = append(dates, item.CheckinDate)
		durations = append(durations, item.DurationMinutes)
		if item.CheckinDate == today {
			todayChecked = true
		}
	}
	groupResult := s.ListGroups(ctx, userID)
	if groupResult.Code != ginResp.OperationSuccess {
		return groupResult
	}
	groups, _ := groupResult.Data.([]fitnessResp.GroupSummary)
	return success(fitnessResp.Home{
		TodayChecked: todayChecked,
		Stats:        support.CalculateStats(dates, durations, time.Now().In(location)),
		Groups:       groups,
	})
}

func (s *Service) AuthorizeSubscription(ctx context.Context, userID string, req *fitnessReq.AuthorizeSubscription) *ginResp.Response {
	if req.TemplateID != global.Cfg.System.Wechat.ReminderTemplateID {
		return failed(ginResp.ReqParameterException, "订阅模板不匹配")
	}
	item, err := shared.EntClient.FitnessSubscriptionAuthorization.Query().
		Where(
			fitnesssubscriptionauthorization.UserIDEQ(userID),
			fitnesssubscriptionauthorization.TemplateIDEQ(req.TemplateID),
		).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return failed(ginResp.OperationFailed, "订阅授权读取失败")
	}
	if item == nil {
		count := 0
		if req.Accepted {
			count = 1
		}
		_, err = shared.EntClient.FitnessSubscriptionAuthorization.Create().
			SetUserID(userID).SetTemplateID(req.TemplateID).SetEnabled(req.Accepted).
			SetAvailableCount(count).SetAuthorizedAt(time.Now()).Save(ctx)
	} else {
		update := shared.EntClient.FitnessSubscriptionAuthorization.UpdateOneID(item.ID).
			SetEnabled(req.Accepted).SetAuthorizedAt(time.Now())
		if req.Accepted {
			update.AddAvailableCount(1)
		} else {
			update.SetAvailableCount(0)
		}
		_, err = update.Save(ctx)
	}
	if err != nil {
		return failed(ginResp.OperationFailed, "订阅授权保存失败")
	}
	return success(nil)
}

func (s *Service) ManualReminder(ctx context.Context, userID string, req *fitnessReq.ManualReminder) *ginResp.Response {
	membership, err := shared.EntClient.FitnessGroupMember.Query().
		Where(
			fitnessgroupmember.GroupIDEQ(req.GroupID),
			fitnessgroupmember.UserIDEQ(userID),
			fitnessgroupmember.StatusEQ(fitnessgroupmember.StatusActive),
		).Only(ctx)
	if err != nil {
		return failed(fitnessResp.PermissionDenied, "你还不是该小组成员")
	}
	group, err := shared.EntClient.FitnessGroup.Get(ctx, req.GroupID)
	if err != nil || group.Status != fitnessgroup.StatusActive {
		return failed(fitnessResp.ResourceNotFound, "小组不存在")
	}
	unchecked, err := uncheckedUserIDs(ctx, group.ID)
	if err != nil {
		return failed(ginResp.OperationFailed, "未打卡名单读取失败")
	}
	targets := make([]support.ReminderTarget, 0)
	if req.AllPending {
		if membership.Role != fitnessgroupmember.RoleAdmin {
			return failed(fitnessResp.PermissionDenied, "只有管理员可以提醒全部")
		}
		for _, id := range unchecked {
			if id != userID {
				targets = append(targets, support.ReminderTarget{UserID: id})
			}
		}
	} else {
		if req.TargetMemberID == "" {
			return failed(ginResp.ReqParameterException, "请选择提醒成员")
		}
		targetMember, queryErr := shared.EntClient.FitnessGroupMember.Query().
			Where(
				fitnessgroupmember.IDEQ(req.TargetMemberID),
				fitnessgroupmember.GroupIDEQ(group.ID),
				fitnessgroupmember.StatusEQ(fitnessgroupmember.StatusActive),
			).
			Only(ctx)
		if queryErr != nil || targetMember.UserID == userID {
			return failed(fitnessResp.ResourceNotFound, "提醒成员不存在")
		}
		found := false
		for _, id := range unchecked {
			if id == targetMember.UserID {
				found = true
				break
			}
		}
		if !found {
			return failed(fitnessResp.ResourceNotFound, "该成员已打卡或不在小组中")
		}
		targets = append(targets, support.ReminderTarget{UserID: targetMember.UserID})
	}
	outcomes, err := support.SendReminders(ctx, group.ID, support.PublicGroupName(group.Name), support.ReminderDeadline(group.ReminderTime), userID, "manual", targets)
	if err != nil {
		shared.Logger.Errorf("manual reminder failed: %s", err)
		return failed(ginResp.OperationFailed, "提醒发送失败")
	}
	return success(outcomes)
}

func uncheckedUserIDs(ctx context.Context, groupID string) ([]string, error) {
	members, err := shared.EntClient.FitnessGroupMember.Query().
		Where(fitnessgroupmember.GroupIDEQ(groupID), fitnessgroupmember.StatusEQ(fitnessgroupmember.StatusActive)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	today := support.Today(support.Location(global.Cfg.System.Timezone))
	checkins, err := shared.EntClient.FitnessCheckin.Query().
		Where(fitnesscheckin.CheckinDateEQ(today), fitnesscheckin.AuditStatusNEQ(fitnesscheckin.AuditStatusRejected)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	checkinIDs := make([]string, 0, len(checkins))
	userByCheckin := make(map[string]string, len(checkins))
	for _, checkin := range checkins {
		checkinIDs = append(checkinIDs, checkin.ID)
		userByCheckin[checkin.ID] = checkin.UserID
	}
	checked := make(map[string]bool)
	if len(checkinIDs) > 0 {
		relations, queryErr := shared.EntClient.FitnessCheckinGroup.Query().
			Where(fitnesscheckingroup.GroupIDEQ(groupID), fitnesscheckingroup.CheckinIDIn(checkinIDs...)).
			All(ctx)
		if queryErr != nil {
			return nil, queryErr
		}
		for _, relation := range relations {
			checked[userByCheckin[relation.CheckinID]] = true
		}
	}
	result := make([]string, 0, len(members))
	for _, member := range members {
		if !checked[member.UserID] {
			result = append(result, member.UserID)
		}
	}
	return result, nil
}

func (s *Service) PublicConfig(ctx context.Context) *ginResp.Response {
	items, err := shared.EntClient.FitnessConfig.Query().All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "产品配置读取失败")
	}
	publicKeys := map[string]bool{
		"defaultReminderTime": true,
		"exerciseTypes":       true,
		"userAgreement":       true,
		"privacyPolicy":       true,
	}
	result := make(map[string]string, len(publicKeys)+1)
	for _, item := range items {
		if publicKeys[item.Key] {
			result[item.Key] = item.Value
		}
	}
	result["reminderTemplateId"] = global.Cfg.System.Wechat.ReminderTemplateID
	return success(result)
}

func pageValues(page, pageSize int) (int, int, int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize, pageSize, (page - 1) * pageSize
}

func (s *Service) AdminDashboard(ctx context.Context) *ginResp.Response {
	today := support.Today(support.Location(global.Cfg.System.Timezone))
	users, err := shared.EntClient.SysUser.Query().Where(sysuser.StatusNEQ(sysuser.StatusCancelled)).Count(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "用户统计失败")
	}
	groups, err := shared.EntClient.FitnessGroup.Query().Where(fitnessgroup.StatusEQ(fitnessgroup.StatusActive)).Count(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "小组统计失败")
	}
	checkins, err := shared.EntClient.FitnessCheckin.Query().Where(fitnesscheckin.CheckinDateEQ(today)).Count(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "今日打卡统计失败")
	}
	reminders, err := shared.EntClient.FitnessReminderLog.Query().
		Where(fitnessreminderlog.ReminderDateEQ(today), fitnessreminderlog.StatusEQ(fitnessreminderlog.StatusSent)).Count(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "提醒统计失败")
	}
	return success(map[string]int{
		"users": users, "groups": groups, "todayCheckins": checkins,
		"todayReminders": reminders,
	})
}

func (s *Service) AdminUsers(ctx context.Context, req *fitnessReq.AdminPage) *ginResp.Response {
	page, pageSize, limit, offset := pageValues(req.Page, req.PageSize)
	query := shared.EntClient.SysUser.Query().Where(sysuser.StatusNEQ(sysuser.StatusCancelled))
	if req.Keyword != "" {
		query = query.Where(sysuser.Or(sysuser.NicknameContains(req.Keyword), sysuser.OpenidContains(req.Keyword)))
	}
	if req.Status != "" {
		query = query.Where(sysuser.StatusEQ(sysuser.Status(req.Status)))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "用户数量读取失败")
	}
	users, err := query.Order(ent.Desc(sysuser.FieldCreatedAt)).Offset(offset).Limit(limit).All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "用户列表读取失败")
	}
	userIDs := make([]string, 0, len(users))
	avatarIDs := make([]string, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
		avatarIDs = append(avatarIDs, user.AvatarFileID)
	}
	groups, err := shared.EntClient.FitnessGroup.Query().Where(fitnessgroup.OwnerIDIn(userIDs...)).All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "用户小组统计失败")
	}
	groupCounts := make(map[string]int)
	for _, group := range groups {
		groupCounts[group.OwnerID]++
	}
	files, _ := loadFiles(ctx, avatarIDs)
	items := make([]map[string]any, 0, len(users))
	for _, user := range users {
		items = append(items, map[string]any{
			"id": user.ID, "openid": user.Openid, "nickname": user.Nickname,
			"avatar": fileResponse(files[user.AvatarFileID]), "status": string(user.Status),
			"reminderEnabled": user.ReminderEnabled, "createdGroupCount": groupCounts[user.ID],
			"createdAt": user.CreatedAt.Unix(),
		})
	}
	return success(fitnessResp.Page[map[string]any]{List: items, Total: total, Page: page, PageSize: pageSize})
}

func (s *Service) AdminUpdateUserStatus(ctx context.Context, userID string, req *fitnessReq.AdminUserStatus) *ginResp.Response {
	user, err := shared.EntClient.SysUser.Get(ctx, userID)
	if err != nil || user.Status == sysuser.StatusCancelled {
		return failed(fitnessResp.ResourceNotFound, "用户不存在")
	}
	if _, err = shared.EntClient.SysUser.UpdateOneID(userID).SetStatus(sysuser.Status(req.Status)).Save(ctx); err != nil {
		return failed(ginResp.OperationFailed, "用户状态保存失败")
	}
	if req.Status == string(sysuser.StatusDisabled) {
		support.RevokeUserSessions(ctx, userID)
	}
	return success(nil)
}

func (s *Service) AdminGroups(ctx context.Context, req *fitnessReq.AdminPage) *ginResp.Response {
	page, pageSize, limit, offset := pageValues(req.Page, req.PageSize)
	query := shared.EntClient.FitnessGroup.Query()
	if req.Keyword != "" {
		query = query.Where(fitnessgroup.NameContains(req.Keyword))
	}
	if req.Status != "" {
		query = query.Where(fitnessgroup.StatusEQ(fitnessgroup.Status(req.Status)))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "小组数量读取失败")
	}
	groups, err := query.Order(ent.Desc(fitnessgroup.FieldCreatedAt)).Offset(offset).Limit(limit).All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "小组列表读取失败")
	}
	groupIDs := make([]string, 0, len(groups))
	ownerIDs := make([]string, 0, len(groups))
	for _, group := range groups {
		groupIDs = append(groupIDs, group.ID)
		ownerIDs = append(ownerIDs, group.OwnerID)
	}
	members, err := shared.EntClient.FitnessGroupMember.Query().
		Where(fitnessgroupmember.GroupIDIn(groupIDs...), fitnessgroupmember.StatusEQ(fitnessgroupmember.StatusActive)).All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "小组成员统计失败")
	}
	memberCounts := make(map[string]int)
	for _, member := range members {
		memberCounts[member.GroupID]++
	}
	owners, _ := loadUsers(ctx, ownerIDs)
	items := make([]map[string]any, 0, len(groups))
	for _, group := range groups {
		ownerName := ""
		if owner := owners[group.OwnerID]; owner != nil {
			ownerName = owner.Nickname
		}
		items = append(items, map[string]any{
			"id": group.ID, "name": support.PublicGroupName(group.Name),
			"ownerId": group.OwnerID, "ownerName": ownerName,
			"memberCount": memberCounts[group.ID], "memberLimit": group.MemberLimit,
			"weeklyTarget": group.WeeklyTarget, "status": string(group.Status), "createdAt": group.CreatedAt.Unix(),
		})
	}
	return success(fitnessResp.Page[map[string]any]{List: items, Total: total, Page: page, PageSize: pageSize})
}

func (s *Service) AdminDissolveGroup(ctx context.Context, groupID string) *ginResp.Response {
	group, err := shared.EntClient.FitnessGroup.Get(ctx, groupID)
	if err != nil {
		return failed(fitnessResp.ResourceNotFound, "小组不存在")
	}
	if group.Status == fitnessgroup.StatusDissolved {
		return success(nil)
	}
	if err = dissolveGroup(ctx, groupID); err != nil {
		return failed(ginResp.OperationFailed, "小组解散失败")
	}
	return success(nil)
}

func dissolveGroup(ctx context.Context, groupID string) error {
	invitations, err := shared.EntClient.FitnessGroupInvitation.Query().
		Where(fitnessgroupinvitation.GroupIDEQ(groupID)).
		All(ctx)
	if err != nil {
		return err
	}
	qrFileIDs := make([]string, 0, len(invitations))
	for _, invitation := range invitations {
		qrFileIDs = append(qrFileIDs, invitation.QrFileID)
	}
	tx, err := shared.EntClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.FitnessGroup.UpdateOneID(groupID).SetStatus(fitnessgroup.StatusDissolved).Save(ctx); err != nil {
		return err
	}
	if _, err = tx.FitnessGroupInvitation.Update().
		Where(fitnessgroupinvitation.GroupIDEQ(groupID)).
		SetActive(false).
		ClearQrFileID().
		Save(ctx); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if err = systemUpload.DeleteFiles(ctx, qrFileIDs); err != nil {
		shared.Logger.Errorf("delete dissolved group invitation QR files failed: %s", err)
	}
	return nil
}

func (s *Service) AdminCheckins(ctx context.Context, req *fitnessReq.AdminPage) *ginResp.Response {
	page, pageSize, limit, offset := pageValues(req.Page, req.PageSize)
	query := shared.EntClient.FitnessCheckin.Query()
	if req.Status != "" {
		query = query.Where(fitnesscheckin.AuditStatusEQ(fitnesscheckin.AuditStatus(req.Status)))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "打卡数量读取失败")
	}
	checkins, err := query.Order(ent.Desc(fitnesscheckin.FieldCreatedAt)).Offset(offset).Limit(limit).All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "打卡列表读取失败")
	}
	responses, err := s.privateCheckinResponses(ctx, checkins)
	if err != nil {
		return failed(ginResp.OperationFailed, "打卡详情读取失败")
	}
	userIDs := make([]string, 0, len(checkins))
	for _, checkin := range checkins {
		userIDs = append(userIDs, checkin.UserID)
	}
	users, _ := loadUsers(ctx, userIDs)
	items := make([]map[string]any, 0, len(checkins))
	for index, checkin := range checkins {
		nickname := ""
		if user := users[checkin.UserID]; user != nil {
			nickname = user.Nickname
		}
		items = append(items, map[string]any{
			"checkin": responses[index], "userId": checkin.UserID, "nickname": nickname,
		})
	}
	return success(fitnessResp.Page[map[string]any]{List: items, Total: total, Page: page, PageSize: pageSize})
}

func (s *Service) AdminDeleteCheckin(ctx context.Context, checkinID string) *ginResp.Response {
	checkin, err := shared.EntClient.FitnessCheckin.Get(ctx, checkinID)
	if err != nil {
		return failed(fitnessResp.ResourceNotFound, "打卡不存在")
	}
	return s.DeleteCheckin(ctx, checkin.UserID, checkinID)
}

func (s *Service) AdminAuditCheckin(ctx context.Context, checkinID string, req *fitnessReq.AdminCheckinAudit) *ginResp.Response {
	if _, err := shared.EntClient.FitnessCheckin.UpdateOneID(checkinID).
		SetAuditStatus(fitnesscheckin.AuditStatus(req.Status)).
		SetAuditDetail(strings.TrimSpace(req.Detail)).
		Save(ctx); err != nil {
		if ent.IsNotFound(err) {
			return failed(fitnessResp.ResourceNotFound, "打卡不存在")
		}
		return failed(ginResp.OperationFailed, "打卡审核状态保存失败")
	}
	return success(nil)
}

func (s *Service) AdminReminders(ctx context.Context, req *fitnessReq.AdminPage) *ginResp.Response {
	page, pageSize, limit, offset := pageValues(req.Page, req.PageSize)
	query := shared.EntClient.FitnessReminderLog.Query()
	if req.Status != "" {
		query = query.Where(fitnessreminderlog.StatusEQ(fitnessreminderlog.Status(req.Status)))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "提醒数量读取失败")
	}
	logs, err := query.Order(ent.Desc(fitnessreminderlog.FieldCreatedAt)).Offset(offset).Limit(limit).All(ctx)
	if err != nil {
		return failed(ginResp.OperationFailed, "提醒列表读取失败")
	}
	userIDs := make([]string, 0, len(logs))
	groupIDs := make([]string, 0, len(logs))
	for _, log := range logs {
		userIDs = append(userIDs, log.TargetUserID)
		groupIDs = append(groupIDs, log.GroupID)
	}
	users, _ := loadUsers(ctx, userIDs)
	groups, _ := shared.EntClient.FitnessGroup.Query().Where(fitnessgroup.IDIn(uniqueNonEmpty(groupIDs)...)).All(ctx)
	groupNames := make(map[string]string, len(groups))
	for _, group := range groups {
		groupNames[group.ID] = group.Name
	}
	items := make([]map[string]any, 0, len(logs))
	for _, log := range logs {
		nickname := ""
		if user := users[log.TargetUserID]; user != nil {
			nickname = user.Nickname
		}
		items = append(items, map[string]any{
			"id": log.ID, "groupId": log.GroupID, "groupName": groupNames[log.GroupID],
			"targetUserId": log.TargetUserID, "targetName": nickname, "type": string(log.Type),
			"status": string(log.Status), "failureReason": log.FailureReason,
			"reminderDate": log.ReminderDate, "createdAt": log.CreatedAt.Unix(),
		})
	}
	return success(fitnessResp.Page[map[string]any]{List: items, Total: total, Page: page, PageSize: pageSize})
}

func (s *Service) AdminUpdateConfig(ctx context.Context, key string, req *fitnessReq.UpdateConfig) *ginResp.Response {
	allowed := map[string]bool{
		"defaultReminderTime": true, "exerciseTypes": true,
		"userAgreement": true, "privacyPolicy": true,
	}
	if !allowed[key] {
		return failed(ginResp.ReqParameterException, "不支持的配置项")
	}
	value := strings.TrimSpace(req.Value)
	if len(value) > 20000 {
		return failed(ginResp.ReqParameterException, "配置内容不能超过 20000 字符")
	}
	if (key == "userAgreement" || key == "privacyPolicy") && value == "" {
		return failed(ginResp.ReqParameterException, "协议内容不能为空")
	}
	if key == "defaultReminderTime" {
		if _, err := time.Parse("15:04", value); err != nil || len(value) != 5 {
			return failed(ginResp.ReqParameterException, "默认提醒时间格式应为 HH:mm")
		}
	}
	if key == "exerciseTypes" {
		var items []struct {
			Value string `json:"value"`
			Label string `json:"label"`
		}
		if err := json.Unmarshal([]byte(value), &items); err != nil || len(items) == 0 {
			return failed(ginResp.ReqParameterException, "运动类型必须是非空 JSON 数组")
		}
		supported := map[string]bool{
			"running": true, "walking": true, "cycling": true, "strength": true,
			"swimming": true, "yoga": true, "ball": true, "rope": true, "other": true,
		}
		seen := make(map[string]bool, len(items))
		for _, item := range items {
			if !supported[item.Value] || seen[item.Value] || strings.TrimSpace(item.Label) == "" {
				return failed(ginResp.ReqParameterException, "运动类型包含无效、重复或空白选项")
			}
			seen[item.Value] = true
		}
	}
	item, err := shared.EntClient.FitnessConfig.Query().Where(fitnessconfig.KeyEQ(key)).Only(ctx)
	if ent.IsNotFound(err) {
		_, err = shared.EntClient.FitnessConfig.Create().SetKey(key).SetValue(value).Save(ctx)
	} else if err == nil {
		_, err = shared.EntClient.FitnessConfig.UpdateOneID(item.ID).SetValue(value).Save(ctx)
	}
	if err != nil {
		return failed(ginResp.OperationFailed, "配置保存失败")
	}
	return success(nil)
}
