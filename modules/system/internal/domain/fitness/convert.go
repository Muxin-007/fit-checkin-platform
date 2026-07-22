package fitness

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"platform/ent"
	"platform/ent/sysstoragefile"
	"platform/modules/system/internal/domain/fitness/support"
	"platform/modules/system/internal/global"
	fitnessResp "platform/modules/system/internal/models/response"
)

func fileResponse(file *ent.SysStorageFile) *fitnessResp.File {
	if file == nil || file.AuditStatus != sysstoragefile.AuditStatusApproved {
		return nil
	}
	return scopedFileResponse(file, "public", time.Hour)
}

func privateFileResponse(file *ent.SysStorageFile) *fitnessResp.File {
	if file == nil || file.AuditStatus == sysstoragefile.AuditStatusRejected {
		return nil
	}
	return scopedFileResponse(file, "private", 15*time.Minute)
}

func auditFileURL(file *ent.SysStorageFile) string {
	response := scopedFileResponse(file, "audit", time.Hour)
	if response == nil {
		return ""
	}
	return response.URL
}

func scopedFileResponse(file *ent.SysStorageFile, scope string, ttl time.Duration) *fitnessResp.File {
	if file == nil {
		return nil
	}
	base := strings.TrimRight(systemPublicURL(), "/")
	expiresAt := time.Now().Add(ttl).Unix()
	return &fitnessResp.File{
		ID:          file.ID,
		Name:        file.Name,
		AuditStatus: string(file.AuditStatus),
		URL: base + "/api/fitness/files/" + url.PathEscape(file.ID) +
			"?scope=" + url.QueryEscape(scope) +
			"&expires=" + strconv.FormatInt(expiresAt, 10) +
			"&signature=" + support.SignFileID(file.ID, scope, expiresAt),
	}
}

func userResponse(user *ent.SysUser, files map[string]*ent.SysStorageFile) fitnessResp.User {
	var avatar *fitnessResp.File
	if user.AvatarFileID != "" {
		avatar = privateFileResponse(files[user.AvatarFileID])
	}
	return fitnessResp.User{
		ID:              user.ID,
		Nickname:        user.Nickname,
		Avatar:          avatar,
		ReminderEnabled: user.ReminderEnabled,
		WeightPublic:    user.WeightPublic,
		Status:          string(user.Status),
	}
}

func systemPublicURL() string {
	return global.Cfg.System.PublicURL
}
