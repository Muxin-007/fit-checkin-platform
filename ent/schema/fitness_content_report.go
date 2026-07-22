package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	mixin "platform/common/db/ent"
	"platform/common/tools/id"
)

type FitnessContentReport struct{ ent.Schema }

func (FitnessContentReport) Mixin() []ent.Mixin {
	return mixin.IDAndCommonFieldsMixin(id.GenID, "内容举报ID")
}

func (FitnessContentReport) Annotations() []schema.Annotation {
	return mixin.CommonAnnotations("fitness_content_reports", "健身内容举报表")
}

func (FitnessContentReport) Fields() []ent.Field {
	return []ent.Field{
		field.String("reporter_user_id").Comment("举报用户ID"),
		field.String("checkin_id").Comment("被举报打卡ID"),
		field.String("reason").MaxLen(300).Comment("举报原因"),
		field.Enum("status").Values("pending", "resolved", "rejected").Default("pending").Comment("处理状态"),
		field.String("handler_user_id").Optional().Comment("处理人用户ID"),
		field.String("resolution").MaxLen(300).Optional().Comment("处理说明"),
	}
}

func (FitnessContentReport) Indexes() []ent.Index {
	return []ent.Index{index.Fields("checkin_id"), index.Fields("status")}
}
