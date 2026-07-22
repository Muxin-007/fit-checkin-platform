package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	mixin "platform/common/db/ent"
	"platform/common/tools/id"
)

type FitnessCheckin struct{ ent.Schema }

func (FitnessCheckin) Mixin() []ent.Mixin {
	return mixin.IDAndCommonFieldsMixin(id.GenID, "健身打卡ID")
}

func (FitnessCheckin) Annotations() []schema.Annotation {
	return mixin.CommonAnnotations("fitness_checkins", "健身打卡表")
}

func (FitnessCheckin) Fields() []ent.Field {
	return []ent.Field{
		field.String("user_id").Comment("用户ID"),
		field.String("checkin_date").Comment("打卡日期，Asia/Shanghai自然日"),
		field.Enum("exercise_type").Values("running", "walking", "cycling", "strength", "swimming", "yoga", "ball", "rope", "other").Comment("运动类型"),
		field.Int("duration_minutes").Min(1).Max(1440).Comment("运动时长（分钟）"),
		field.String("content").MaxLen(500).Optional().Comment("运动内容"),
		field.Int("calories").Optional().Nillable().Min(0).Max(10000).Comment("消耗热量"),
		field.Float("weight").Optional().Nillable().Positive().Comment("当前体重"),
		field.Bool("weight_public").Default(false).Comment("是否公开体重"),
		field.String("mood").MaxLen(100).Optional().Comment("心情感受"),
		field.Enum("audit_status").Values("pending", "approved", "rejected").Default("pending").Comment("内容审核状态"),
		field.String("audit_detail").Optional().Comment("审核结果说明"),
	}
}

func (FitnessCheckin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "checkin_date").Unique(),
		index.Fields("checkin_date"),
	}
}
