package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"

	mixin "platform/common/db/ent"
	"platform/common/tools/id"
)

type FitnessGroup struct{ ent.Schema }

func (FitnessGroup) Mixin() []ent.Mixin {
	return mixin.IDAndCommonFieldsMixin(id.GenID, "健身小组ID")
}

func (FitnessGroup) Annotations() []schema.Annotation {
	return mixin.CommonAnnotations("fitness_groups", "健身小组表")
}

func (FitnessGroup) Fields() []ent.Field {
	return []ent.Field{
		field.String("owner_id").Comment("管理员用户ID"),
		field.String("name").MaxLen(30).Comment("小组名称"),
		field.String("avatar_file_id").Optional().Comment("小组头像存储文件ID"),
		field.String("description").MaxLen(200).Optional().Comment("小组简介"),
		field.String("announcement").MaxLen(300).Optional().Comment("小组公告"),
		field.Int("weekly_target").Default(3).Min(1).Max(7).Comment("每周目标天数"),
		field.String("reminder_time").Default("20:00").Comment("每日提醒时间"),
		field.Bool("require_approval").Default(false).Comment("加入是否需要审核"),
		field.Int("member_limit").Default(50).Min(2).Max(200).Comment("人数上限"),
		field.Enum("status").Values("active", "dissolved", "blocked").Default("active").Comment("小组状态"),
	}
}
