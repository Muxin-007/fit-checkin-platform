package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	mixin "platform/common/db/ent"
	"platform/common/tools/id"
)

type FitnessCheckinGroup struct{ ent.Schema }

func (FitnessCheckinGroup) Mixin() []ent.Mixin {
	return mixin.IDAndCommonFieldsMixin(id.GenID, "打卡小组关联ID")
}

func (FitnessCheckinGroup) Annotations() []schema.Annotation {
	return mixin.CommonAnnotations("fitness_checkin_groups", "打卡与小组关联表")
}

func (FitnessCheckinGroup) Fields() []ent.Field {
	return []ent.Field{
		field.String("checkin_id").Comment("打卡ID"),
		field.String("group_id").Comment("小组ID"),
	}
}

func (FitnessCheckinGroup) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("checkin_id", "group_id").Unique(),
		index.Fields("group_id"),
	}
}
