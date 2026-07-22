package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	mixin "platform/common/db/ent"
	"platform/common/tools/id"
)

// SysUser holds the schema definition for the SysUser entity.
type SysUser struct {
	ent.Schema
}

func (SysUser) Mixin() []ent.Mixin {
	return mixin.IDAndCommonFieldsMixin(id.GenID, "用户ID")
}

func (SysUser) Annotations() []schema.Annotation {
	return mixin.CommonAnnotations("sys_users", "系统用户表")
}

// Fields of the User.
func (SysUser) Fields() []ent.Field {
	return []ent.Field{
		field.String("openid").Comment("微信OpenID"),
		field.String("unionid").Optional().Comment("微信UnionID"),
		field.String("nickname").Default("微信用户").MaxLen(30).Comment("昵称"),
		field.String("avatar_file_id").
			Optional().
			Comment("头像存储文件ID"),
		field.Bool("reminder_enabled").Default(true).Comment("是否接收提醒"),
		field.Bool("weight_public").Default(false).Comment("体重默认是否公开"),
		field.Enum("status").Values("active", "disabled", "cancelled").Default("active").Comment("账号状态"),
		field.Time("cancelled_at").Optional().Comment("注销时间"),
	}
}

// Indexes of the User.
func (SysUser) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("openid").Unique(),
	}
}
