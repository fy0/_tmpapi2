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

// SupportTicketMessage stores the conversation for a support ticket.
type SupportTicketMessage struct {
	ent.Schema
}

func (SupportTicketMessage) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "support_ticket_messages"},
	}
}

func (SupportTicketMessage) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("ticket_id").
			Comment("工单 ID"),
		field.Int64("author_id").
			Optional().
			Nillable().
			Comment("作者用户 ID"),
		field.String("author_role").
			MaxLen(20).
			Default("user").
			Comment("作者角色: user, admin"),
		field.String("author_email").
			MaxLen(255).
			Optional().
			Comment("作者邮箱快照"),
		field.String("author_name").
			MaxLen(100).
			Optional().
			Comment("作者名称快照"),
		field.String("content").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			NotEmpty().
			Comment("消息内容"),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (SupportTicketMessage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ticket", SupportTicket.Type).
			Ref("messages").
			Field("ticket_id").
			Required().
			Unique(),
	}
}

func (SupportTicketMessage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("ticket_id"),
		index.Fields("author_id"),
		index.Fields("created_at"),
		index.Fields("ticket_id", "created_at"),
	}
}
