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

type FitnessSubscriptionAuthorization struct{ ent.Schema }

func (FitnessSubscriptionAuthorization) Mixin() []ent.Mixin {
	return mixin.IDAndCommonFieldsMixin(id.GenID, "订阅授权ID")
}

func (FitnessSubscriptionAuthorization) Annotations() []schema.Annotation {
	return mixin.CommonAnnotations("fitness_subscription_authorizations", "小程序订阅消息授权表")
}

func (FitnessSubscriptionAuthorization) Fields() []ent.Field {
	return []ent.Field{
		field.String("user_id").Comment("用户ID"),
		field.String("template_id").Comment("订阅消息模板ID"),
		field.Bool("enabled").Default(true).Comment("是否启用"),
		field.Int("available_count").Default(0).Min(0).Comment("可发送次数"),
		field.Time("authorized_at").Default(time.Now).Comment("最近授权时间"),
	}
}

func (FitnessSubscriptionAuthorization) Indexes() []ent.Index {
	return []ent.Index{index.Fields("user_id", "template_id").Unique()}
}
