package support

import (
	"context"
	"encoding/json"
	"time"

	"platform/common/tools"
	"platform/ent"
	"platform/ent/fitnesscheckin"
	"platform/ent/fitnesscheckingroup"
	"platform/ent/fitnesscheckinimage"
	"platform/ent/fitnessconfig"
	"platform/ent/fitnessgroup"
	"platform/ent/fitnessgroupmember"
	"platform/ent/sysstoragefile"
	"platform/ent/sysuser"
	"platform/modules/shared"
	"platform/modules/system/internal/global"
	systemUpload "platform/modules/system/pkg/upload"
)

var defaultConfigs = map[string]string{
	"defaultReminderTime": "20:00",
	"exerciseTypes": mustJSON([]map[string]string{
		{"value": "running", "label": "跑步"},
		{"value": "walking", "label": "走路"},
		{"value": "cycling", "label": "骑行"},
		{"value": "strength", "label": "力量训练"},
		{"value": "swimming", "label": "游泳"},
		{"value": "yoga", "label": "瑜伽"},
		{"value": "ball", "label": "球类运动"},
		{"value": "rope", "label": "跳绳"},
		{"value": "other", "label": "其他"},
	}),
	"userAgreement": "欢迎使用「再鸽一天」。本工具用于记录个人运动类型、时长、日期和私密照片，并在小组内同步当天是否完成。小组成员不能查看他人的运动详情或照片；平台不提供动态、评论、点赞或自由文本发布功能。本工具不提供医疗诊断、治疗或康复建议。",
	"privacyPolicy": "我们仅收集提供服务所必需的微信身份标识、个人运动记录、私密照片、小组关系和订阅授权。运动详情和照片仅本人可查看；小组成员只能看到当天是否完成。我们不向其他用户展示昵称头像、自由文本或个人打卡内容。你可以删除打卡、关闭提醒或注销账号。",
}

func StartScheduler() {
	if err := seedConfigs(context.Background()); err != nil {
		shared.Logger.Errorf("seed fitness configs failed: %s", err)
	}
	interval, err := tools.ParseDuration(global.Cfg.System.Wechat.ReminderScanInterval)
	if err != nil || interval < time.Minute {
		interval = time.Minute
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		runScheduledReminders(context.Background())
		for range ticker.C {
			runScheduledReminders(context.Background())
		}
	}()
	go func() {
		cleanupOrphanedUploads(context.Background())
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			cleanupOrphanedUploads(context.Background())
		}
	}()
}

func seedConfigs(ctx context.Context) error {
	keys := make([]string, 0, len(defaultConfigs))
	for key := range defaultConfigs {
		keys = append(keys, key)
	}
	existing, err := shared.EntClient.FitnessConfig.Query().Where(fitnessconfig.KeyIn(keys...)).All(ctx)
	if err != nil {
		return err
	}
	present := make(map[string]bool, len(existing))
	for _, item := range existing {
		present[item.Key] = true
	}
	creates := make([]*ent.FitnessConfigCreate, 0)
	for key, value := range defaultConfigs {
		if !present[key] {
			creates = append(creates, shared.EntClient.FitnessConfig.Create().SetKey(key).SetValue(value))
		}
	}
	if len(creates) > 0 {
		if _, err = shared.EntClient.FitnessConfig.CreateBulk(creates...).Save(ctx); err != nil {
			return err
		}
	}
	legacyAgreement := "欢迎使用「再鸽一天」。请记录真实、合法的运动内容，不发布违法违规或侵害他人权益的信息。你可以随时删除自己的打卡或注销账号。平台仅提供运动记录与好友监督功能，不提供医疗诊断、治疗或康复建议。"
	if _, err = shared.EntClient.FitnessConfig.Update().
		Where(fitnessconfig.KeyEQ("userAgreement"), fitnessconfig.ValueEQ(legacyAgreement)).
		SetValue(defaultConfigs["userAgreement"]).Save(ctx); err != nil {
		return err
	}
	legacyPrivacy := "我们仅收集提供服务所必需的微信身份标识、昵称头像、运动打卡、小组关系和订阅授权。体重默认仅自己可见；上传内容用于展示与内容安全审核。你可以删除打卡、关闭提醒或注销账号。注销后停止提供服务并按规则清理个人数据。"
	_, err = shared.EntClient.FitnessConfig.Update().
		Where(fitnessconfig.KeyEQ("privacyPolicy"), fitnessconfig.ValueEQ(legacyPrivacy)).
		SetValue(defaultConfigs["privacyPolicy"]).Save(ctx)
	return err
}

func runScheduledReminders(ctx context.Context) {
	location := Location(global.Cfg.System.Timezone)
	now := time.Now().In(location)
	groups, err := shared.EntClient.FitnessGroup.Query().
		Where(
			fitnessgroup.StatusEQ(fitnessgroup.StatusActive),
			fitnessgroup.ReminderTimeEQ(now.Format("15:04")),
		).All(ctx)
	if err != nil || len(groups) == 0 {
		if err != nil {
			shared.Logger.Errorf("query scheduled reminder groups failed: %s", err)
		}
		return
	}
	groupIDs := make([]string, 0, len(groups))
	groupByID := make(map[string]*ent.FitnessGroup, len(groups))
	for _, group := range groups {
		groupIDs = append(groupIDs, group.ID)
		groupByID[group.ID] = group
	}
	members, err := shared.EntClient.FitnessGroupMember.Query().
		Where(
			fitnessgroupmember.GroupIDIn(groupIDs...),
			fitnessgroupmember.StatusEQ(fitnessgroupmember.StatusActive),
		).All(ctx)
	if err != nil {
		shared.Logger.Errorf("query scheduled reminder members failed: %s", err)
		return
	}
	today := now.Format(time.DateOnly)
	checkins, err := shared.EntClient.FitnessCheckin.Query().
		Where(
			fitnesscheckin.CheckinDateEQ(today),
			fitnesscheckin.AuditStatusNEQ(fitnesscheckin.AuditStatusRejected),
		).All(ctx)
	if err != nil {
		shared.Logger.Errorf("query scheduled reminder checkins failed: %s", err)
		return
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
			Where(
				fitnesscheckingroup.GroupIDIn(groupIDs...),
				fitnesscheckingroup.CheckinIDIn(checkinIDs...),
			).All(ctx)
		if queryErr != nil {
			shared.Logger.Errorf("query scheduled reminder relations failed: %s", queryErr)
			return
		}
		for _, relation := range relations {
			checked[relation.GroupID+":"+userByCheckin[relation.CheckinID]] = true
		}
	}
	targetsByGroup := make(map[string][]ReminderTarget)
	for _, member := range members {
		if !checked[member.GroupID+":"+member.UserID] {
			targetsByGroup[member.GroupID] = append(targetsByGroup[member.GroupID], ReminderTarget{UserID: member.UserID})
		}
	}
	batches := make([]ReminderBatch, 0, len(groups))
	for _, group := range groups {
		batches = append(batches, ReminderBatch{
			GroupID: group.ID, GroupName: PublicGroupName(group.Name), Deadline: ReminderDeadline(group.ReminderTime),
			Type: "scheduled", Targets: targetsByGroup[group.ID],
		})
	}
	if _, err = SendReminderBatches(ctx, batches); err != nil {
		shared.Logger.Errorf("send scheduled reminders failed: %s", err)
	}
}

func mustJSON(value any) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

func cleanupOrphanedUploads(ctx context.Context) {
	candidates, err := shared.EntClient.SysStorageFile.Query().
		Where(
			sysstoragefile.CreatedAtLT(time.Now().Add(-24*time.Hour)),
			sysstoragefile.PurposeIn(
				sysstoragefile.PurposeAvatar,
				sysstoragefile.PurposeGroupAvatar,
				sysstoragefile.PurposeCheckin,
			),
		).
		Order(ent.Asc(sysstoragefile.FieldCreatedAt)).
		Limit(500).
		All(ctx)
	if err != nil || len(candidates) == 0 {
		if err != nil {
			shared.Logger.Errorf("query orphaned uploads failed: %s", err)
		}
		return
	}
	ids := make([]string, 0, len(candidates))
	for _, file := range candidates {
		ids = append(ids, file.ID)
	}
	referenced := make(map[string]bool, len(ids))
	users, userErr := shared.EntClient.SysUser.Query().Where(sysuser.AvatarFileIDIn(ids...)).All(ctx)
	groups, groupErr := shared.EntClient.FitnessGroup.Query().Where(fitnessgroup.AvatarFileIDIn(ids...)).All(ctx)
	images, imageErr := shared.EntClient.FitnessCheckinImage.Query().Where(fitnesscheckinimage.StorageFileIDIn(ids...)).All(ctx)
	if userErr != nil || groupErr != nil || imageErr != nil {
		shared.Logger.Error("query upload references failed")
		return
	}
	for _, user := range users {
		referenced[user.AvatarFileID] = true
	}
	for _, group := range groups {
		referenced[group.AvatarFileID] = true
	}
	for _, image := range images {
		referenced[image.StorageFileID] = true
	}
	orphanIDs := make([]string, 0)
	for _, id := range ids {
		if !referenced[id] {
			orphanIDs = append(orphanIDs, id)
		}
	}
	if err = systemUpload.DeleteFiles(ctx, orphanIDs); err != nil {
		shared.Logger.Errorf("delete orphaned uploads failed: %s", err)
	}
}
