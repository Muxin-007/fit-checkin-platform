package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	mixin "platform/common/db/ent"
	"platform/common/tools/id"
)

type FitnessCheckinImage struct{ ent.Schema }

func (FitnessCheckinImage) Mixin() []ent.Mixin {
	return mixin.IDAndCommonFieldsMixin(id.GenID, "打卡图片ID")
}

func (FitnessCheckinImage) Annotations() []schema.Annotation {
	return mixin.CommonAnnotations("fitness_checkin_images", "打卡图片表")
}

func (FitnessCheckinImage) Fields() []ent.Field {
	return []ent.Field{
		field.String("checkin_id").Comment("打卡ID"),
		field.String("storage_file_id").Comment("存储文件ID"),
		field.Int("sort").Default(0).Comment("排序"),
		field.Enum("audit_status").Values("pending", "approved", "rejected").Default("pending").Comment("图片审核状态"),
		field.String("audit_trace_id").Optional().Comment("微信审核追踪ID"),
	}
}

func (FitnessCheckinImage) Indexes() []ent.Index {
	return []ent.Index{index.Fields("checkin_id"), index.Fields("storage_file_id").Unique()}
}
