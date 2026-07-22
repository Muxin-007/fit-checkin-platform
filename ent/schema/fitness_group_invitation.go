package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	mixin "platform/common/db/ent"
	"platform/common/tools/id"
)

type FitnessGroupInvitation struct{ ent.Schema }

func (FitnessGroupInvitation) Mixin() []ent.Mixin {
	return mixin.IDAndCommonFieldsMixin(id.GenID, "小组邀请ID")
}

func (FitnessGroupInvitation) Annotations() []schema.Annotation {
	return mixin.CommonAnnotations("fitness_group_invitations", "健身小组邀请表")
}

func (FitnessGroupInvitation) Fields() []ent.Field {
	return []ent.Field{
		field.String("group_id").Comment("小组ID"),
		field.String("creator_id").Comment("创建用户ID"),
		field.String("code").Comment("邀请码"),
		field.String("qr_file_id").Optional().Comment("小程序码存储文件ID"),
		field.Time("expires_at").Comment("失效时间"),
		field.Bool("active").Default(true).Comment("是否有效"),
	}
}

func (FitnessGroupInvitation) Indexes() []ent.Index {
	return []ent.Index{index.Fields("code").Unique(), index.Fields("group_id", "active")}
}
