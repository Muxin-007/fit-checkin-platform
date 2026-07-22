package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"

	mixin "platform/common/db/ent"
	"platform/common/tools/id"
)

// SysStorage holds the schema definition for the SysStorage entity.
type SysStorageFile struct {
	ent.Schema
}

func (SysStorageFile) Mixin() []ent.Mixin {
	return mixin.IDAndCommonFieldsMixin(id.GenID, "存储ID")
}

func (SysStorageFile) Annotations() []schema.Annotation {
	return mixin.CommonAnnotations("sys_storage_files", "系统存储文件表")
}

// Fields of the storage.
func (SysStorageFile) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("文件名称"),
		field.String("owner_user_id").
			Optional().
			Comment("上传用户ID"),
		field.Enum("purpose").
			Values("avatar", "group_avatar", "checkin", "invitation_qr").
			Default("checkin").
			Comment("文件用途"),
		field.Enum("audit_status").
			Values("pending", "approved", "rejected").
			Default("pending").
			Comment("内容安全审核状态"),
		field.String("audit_trace_id").
			Optional().
			Comment("微信内容审核追踪ID"),
		field.String("tag").
			Optional().
			Comment("文件标签"),
		field.Uint64("size").
			Default(0).
			Comment("文件大小: 单位: 字节"),
		field.Enum("type").
			Values("file", "img").
			Default("file").
			Comment("文件类型"),
		field.String("key").
			Comment("文件key"),
	}
}
