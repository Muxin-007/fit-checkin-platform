package support

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"platform/ent"
	"platform/ent/fitnessreminderlog"
	"platform/ent/fitnesssubscriptionauthorization"
	"platform/ent/sysuser"
	"platform/modules/shared"
	"platform/modules/system/internal/global"
)

type ReminderTarget struct {
	UserID string
}

type ReminderBatch struct {
	GroupID   string
	GroupName string
	Deadline  string
	SenderID  string
	Type      string
	Targets   []ReminderTarget
}

type ReminderOutcome struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type reminderLogData struct {
	groupID, targetID, senderID, reminderType, date, status, reason, messageID string
}

func SendReminders(ctx context.Context, groupID, groupName, deadline, senderID, reminderType string, targets []ReminderTarget) ([]ReminderOutcome, error) {
	results, err := SendReminderBatches(ctx, []ReminderBatch{{
		GroupID: groupID, GroupName: groupName, Deadline: deadline,
		SenderID: senderID, Type: reminderType, Targets: targets,
	}})
	outcomes := results[groupID]
	if outcomes == nil {
		outcomes = []ReminderOutcome{}
	}
	return outcomes, err
}

func SendReminderBatches(ctx context.Context, batches []ReminderBatch) (map[string][]ReminderOutcome, error) {
	results := make(map[string][]ReminderOutcome, len(batches))
	userIDs := make([]string, 0)
	groupIDs := make([]string, 0, len(batches))
	types := make([]fitnessreminderlog.Type, 0, len(batches))
	for _, batch := range batches {
		groupIDs = append(groupIDs, batch.GroupID)
		types = append(types, fitnessreminderlog.Type(batch.Type))
		for _, target := range batch.Targets {
			userIDs = append(userIDs, target.UserID)
		}
	}
	userIDs = uniqueStrings(userIDs)
	if len(userIDs) == 0 {
		return results, nil
	}
	users, err := shared.EntClient.SysUser.Query().Where(sysuser.IDIn(userIDs...)).All(ctx)
	if err != nil {
		return nil, err
	}
	usersByID := make(map[string]*ent.SysUser, len(users))
	for _, user := range users {
		usersByID[user.ID] = user
	}
	templateID := global.Cfg.System.Wechat.ReminderTemplateID
	authorizations, err := shared.EntClient.FitnessSubscriptionAuthorization.Query().
		Where(
			fitnesssubscriptionauthorization.UserIDIn(userIDs...),
			fitnesssubscriptionauthorization.TemplateIDEQ(templateID),
			fitnesssubscriptionauthorization.EnabledEQ(true),
			fitnesssubscriptionauthorization.AvailableCountGT(0),
		).All(ctx)
	if err != nil {
		return nil, err
	}
	authorizationByUser := make(map[string]*ent.FitnessSubscriptionAuthorization, len(authorizations))
	for _, authorization := range authorizations {
		authorizationByUser[authorization.UserID] = authorization
	}
	today := Today(Location(global.Cfg.System.Timezone))
	existing, err := shared.EntClient.FitnessReminderLog.Query().
		Where(
			fitnessreminderlog.GroupIDIn(uniqueStrings(groupIDs)...),
			fitnessreminderlog.TargetUserIDIn(userIDs...),
			fitnessreminderlog.ReminderDateEQ(today),
			fitnessreminderlog.TypeIn(uniqueReminderTypes(types)...),
		).All(ctx)
	if err != nil {
		return nil, err
	}
	alreadySent := make(map[string]bool, len(existing))
	for _, item := range existing {
		alreadySent[reminderKey(item.GroupID, item.TargetUserID, string(item.Type))] = true
	}
	logs := make([]reminderLogData, 0, len(userIDs))
	usedAuthorizationIDs := make([]string, 0, len(userIDs))
	for _, batch := range batches {
		outcomes := make([]ReminderOutcome, 0, len(batch.Targets))
		for _, target := range batch.Targets {
			status := fitnessreminderlog.StatusSkipped
			reason := ""
			msgID := ""
			user := usersByID[target.UserID]
			authorization := authorizationByUser[target.UserID]
			key := reminderKey(batch.GroupID, target.UserID, batch.Type)
			switch {
			case alreadySent[key]:
				reason = "今天已提醒过"
			case user == nil || user.Status != sysuser.StatusActive:
				reason = "用户不可用"
			case !user.ReminderEnabled:
				reason = "用户已关闭提醒"
			case authorization == nil:
				reason = "没有可用的订阅授权"
			default:
				wechatMsgID, sendErr := SendReminder(ctx, user.Openid, batch.GroupName, batch.Deadline)
				if sendErr != nil {
					status = fitnessreminderlog.StatusFailed
					reason = sendErr.Error()
				} else {
					status = fitnessreminderlog.StatusSent
					msgID = strconv.FormatInt(wechatMsgID, 10)
					usedAuthorizationIDs = append(usedAuthorizationIDs, authorization.ID)
					delete(authorizationByUser, target.UserID)
				}
			}
			alreadySent[key] = true
			outcomes = append(outcomes, ReminderOutcome{Status: string(status), Reason: reason})
			logs = append(logs, reminderLogData{
				groupID: batch.GroupID, targetID: target.UserID, senderID: batch.SenderID,
				reminderType: batch.Type, date: today, status: string(status),
				reason: reason, messageID: msgID,
			})
		}
		results[batch.GroupID] = outcomes
	}
	tx, err := shared.EntClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if len(logs) > 0 {
		creates := make([]*ent.FitnessReminderLogCreate, 0, len(logs))
		for _, item := range logs {
			creates = append(creates, tx.FitnessReminderLog.Create().
				SetGroupID(item.groupID).
				SetTargetUserID(item.targetID).
				SetNillableSenderUserID(optionalString(item.senderID)).
				SetType(fitnessreminderlog.Type(item.reminderType)).
				SetReminderDate(item.date).
				SetStatus(fitnessreminderlog.Status(item.status)).
				SetFailureReason(item.reason).
				SetWechatMsgID(item.messageID))
		}
		if _, err = tx.FitnessReminderLog.CreateBulk(creates...).Save(ctx); err != nil {
			return nil, err
		}
	}
	if len(usedAuthorizationIDs) > 0 {
		if _, err = tx.FitnessSubscriptionAuthorization.Update().
			Where(fitnesssubscriptionauthorization.IDIn(uniqueStrings(usedAuthorizationIDs)...)).
			AddAvailableCount(-1).
			Save(ctx); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return results, nil
}

func reminderKey(groupID, userID, reminderType string) string {
	return groupID + ":" + userID + ":" + reminderType
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func uniqueReminderTypes(values []fitnessreminderlog.Type) []fitnessreminderlog.Type {
	seen := make(map[fitnessreminderlog.Type]struct{}, len(values))
	result := make([]fitnessreminderlog.Type, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func ReminderDeadline(reminderTime string) string {
	now := time.Now().In(Location(global.Cfg.System.Timezone))
	return fmt.Sprintf("%s 23:59", now.Format("2006-01-02"))
}
