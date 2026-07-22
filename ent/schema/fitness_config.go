package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	mixin "platform/common/db/ent"
	"platform/common/tools/id"
)

type FitnessConfig struct{ ent.Schema }

func (FitnessConfig) Mixin() []ent.Mixin {
	return mixin.IDAndCommonFieldsMixin(id.GenID, "健身配置ID")
}

func (FitnessConfig) Annotations() []schema.Annotation {
	return mixin.CommonAnnotations("fitness_configs", "健身业务配置表")
}

func (FitnessConfig) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").Comment("配置键"),
		field.Text("value").Comment("配置值"),
		field.String("description").Optional().Comment("配置说明"),
	}
}

func (FitnessConfig) Indexes() []ent.Index {
	return []ent.Index{index.Fields("key").Unique()}
}
