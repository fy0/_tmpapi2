package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SupportTicket stores user feedback/work orders.
//
// Keep this schema independent from User to reduce coupling with upstream
// changes. User identity is stored as an ID plus display snapshots.
type SupportTicket struct {
	ent.Schema
}

func (SupportTicket) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "support_tickets"},
	}
}

func (SupportTicket) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").
			Comment("创建用户 ID（快照引用，不建立外键）"),
		field.String("user_email").
			MaxLen(255).
			Optional().
			Comment("创建用户邮箱快照"),
		field.String("user_name").
			MaxLen(100).
			Optional().
			Comment("创建用户名称快照"),
		field.String("title").
			MaxLen(200).
			NotEmpty().
			Comment("工单标题"),
		field.String("category").
			MaxLen(40).
			Default("feedback").
			Comment("分类: feedback, bug, billing, account, other"),
		field.String("status").
			MaxLen(20).
			Default("open").
			Comment("状态: open, pending, resolved, closed"),
		field.String("priority").
			MaxLen(20).
			Default("normal").
			Comment("优先级: low, normal, high"),
		field.Int64("created_by").
			Optional().
			Nillable().
			Comment("创建人用户 ID"),
		field.Int64("updated_by").
			Optional().
			Nillable().
			Comment("最后更新人用户 ID"),
		field.Time("last_message_at").
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("最后消息时间"),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (SupportTicket) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("messages", SupportTicketMessage.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (SupportTicket) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("status"),
		index.Fields("category"),
		index.Fields("last_message_at"),
		index.Fields("created_at"),
		index.Fields("user_id", "last_message_at"),
	}
}
