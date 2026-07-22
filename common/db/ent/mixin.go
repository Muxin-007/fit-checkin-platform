package ent

import (
	"context"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
)

var CommonFieldsMixin = []ent.Mixin{
	OperatedTime{},
	SoftDelete{},
}

func IDAndCommonFieldsMixin(genID func() string, comment ...string) []ent.Mixin {
	idComment := "主键ID"
	if len(comment) > 0 {
		idComment = comment[0]
	}
	return []ent.Mixin{
		NewIDFieldMixin(genID, idComment),
		OperatedTime{},
		SoftDelete{},
	}
}

func IDAndOperatedFieldsMixin(genID func() string, comment ...string) []ent.Mixin {
	idComment := "主键ID"
	if len(comment) > 0 {
		idComment = comment[0]
	}
	return []ent.Mixin{
		NewIDFieldMixin(genID, idComment),
		OperatedTime{},
	}
}

func NewIDFieldMixin(genID func() string, comment string) IdFieldMixin {
	return IdFieldMixin{
		comment: comment,
		genID:   genID,
	}
}

func CommonAnnotations(tableName, tableComment string) []schema.Annotation {
	withCommentsEnabled := true
	return []schema.Annotation{
		schema.Comment(tableComment),
		entsql.Annotation{
			Table:        tableName,
			Charset:      "utf8mb4",
			Collation:    "utf8mb4_general_ci",
			WithComments: &withCommentsEnabled,
		},
		edge.Annotation{StructTag: `json:"-"`},
	}
}

type IdFieldMixin struct {
	mixin.Schema
	comment string
	genID   func() string
}

func (i IdFieldMixin) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			DefaultFunc(i.genID).
			Comment(i.comment),
	}
}

func (IdFieldMixin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("id").
			Unique(),
	}
}

type OperatedTime struct {
	mixin.Schema
}

func (OperatedTime) Hooks() []ent.Hook {
	return []ent.Hook{
		func(next ent.Mutator) ent.Mutator {
			return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
				if m.Op().Is(ent.OpUpdate | ent.OpUpdateOne) {
					if m, ok := m.(interface{ SetUpdatedAt(time.Time) }); ok {
						m.SetUpdatedAt(time.Now())
					}
				}
				return next.Mutate(ctx, m)
			})
		},
		func(next ent.Mutator) ent.Mutator {
			return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
				if m.Op().Is(ent.OpCreate) {
					if m, ok := m.(interface{ SetCreatedAt(time.Time) }); ok {
						m.SetCreatedAt(time.Now())
					}
				}
				return next.Mutate(ctx, m)
			})
		},
	}
}

func (OperatedTime) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			Default(time.Now).
			Optional().
			Comment("创建时间"),
		field.Time("updated_at").
			Optional().
			Default(nil).
			Comment("更新时间"),
	}
}

type SoftDelete struct {
	mixin.Schema
}

func (SoftDelete) Fields() []ent.Field {
	return []ent.Field{
		field.Time("deleted_at").
			Optional().
			Default(nil).
			Comment("删除时间"),
	}
}

type Status struct {
	mixin.Schema
}

func (Status) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("1", "2").
			Default("1").
			Comment("状态 1:启用 2:禁用"),
	}
}

// The schema definition for the DbVersion entity.
type DbVersion struct {
	ent.Schema
}

// Fields of the DbVersion.
func (DbVersion) Fields() []ent.Field {
	return []ent.Field{
		field.String("version").NotEmpty().Comment("数据库版本号"),
		field.String("description").Optional().Comment("版本描述"),
	}
}
