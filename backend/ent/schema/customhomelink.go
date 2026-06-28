package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CustomHomeLink holds the schema definition for homepage custom links.
type CustomHomeLink struct {
	ent.Schema
}

func (CustomHomeLink) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "custom_home_links"},
	}
}

func (CustomHomeLink) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			MaxLen(100).
			NotEmpty(),
		field.String("description").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.String("url").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			NotEmpty(),
		field.Bool("open_in_new_window").
			Default(true),
		field.Int("sort_order").
			Default(0),
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

func (CustomHomeLink) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("sort_order", "id").
			StorageKey("idx_custom_home_links_sort_order_id"),
	}
}
