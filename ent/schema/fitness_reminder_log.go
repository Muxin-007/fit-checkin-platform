package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	mixin "platform/common/db/ent"
	"platform/common/tools/id"
)

type FitnessReminderLog struct{ ent.Schema }

func (FitnessReminderLog) Mixin() []ent.Mixin {
	return mixin.IDAndCommonFieldsMixin(id.GenID, "提醒日志ID")
}

func (FitnessReminderLog) Annotations() []schema.Annotation {
	return mixin.CommonAnnotations("fitness_reminder_logs", "健身提醒发送日志表")
}

func (FitnessReminderLog) Fields() []ent.Field {
	return []ent.Field{
		field.String("group_id").Optional().Comment("小组ID"),
		field.String("target_user_id").Comment("接收用户ID"),
		field.String("sender_user_id").Optional().Comment("发起用户ID"),
		field.Enum("type").Values("scheduled", "manual").Comment("提醒类型"),
		field.String("reminder_date").Comment("提醒日期"),
		field.Enum("status").Values("sent", "failed", "skipped").Comment("发送状态"),
		field.String("failure_reason").Optional().Comment("失败原因"),
		field.String("wechat_msg_id").Optional().Comment("微信消息ID"),
	}
}

func (FitnessReminderLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("target_user_id", "group_id", "reminder_date", "type"),
		index.Fields("created_at"),
	}
}
