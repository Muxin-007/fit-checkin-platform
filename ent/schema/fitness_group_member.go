package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	mixin "platform/common/db/ent"
	"platform/common/tools/id"
)

type FitnessGroupMember struct{ ent.Schema }

func (FitnessGroupMember) Mixin() []ent.Mixin {
	return mixin.IDAndCommonFieldsMixin(id.GenID, "小组成员关系ID")
}

func (FitnessGroupMember) Annotations() []schema.Annotation {
	return mixin.CommonAnnotations("fitness_group_members", "健身小组成员关系表")
}

func (FitnessGroupMember) Fields() []ent.Field {
	return []ent.Field{
		field.String("group_id").Comment("小组ID"),
		field.String("user_id").Comment("用户ID"),
		field.Enum("role").Values("admin", "member").Default("member").Comment("成员角色"),
		field.Enum("status").Values("pending", "active", "rejected").Default("active").Comment("成员状态"),
		field.Time("joined_at").Default(time.Now).Comment("加入时间"),
	}
}

func (FitnessGroupMember) Indexes() []ent.Index {
	return []ent.Index{index.Fields("group_id", "user_id").Unique()}
}
