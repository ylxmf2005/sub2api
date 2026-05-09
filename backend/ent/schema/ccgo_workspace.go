package schema

import (
	"fmt"

	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CcgoWorkspace stores the stable mapping between one local project root and one server projection root.
type CcgoWorkspace struct {
	ent.Schema
}

func (CcgoWorkspace) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "ccgo_workspaces"},
	}
}

func (CcgoWorkspace) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (CcgoWorkspace) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("workspace_slug").
			MaxLen(64).
			NotEmpty().
			Unique(),
		field.String("server_root").
			MaxLen(512).
			NotEmpty().
			Unique(),
		field.String("local_root_hash").
			MaxLen(64).
			NotEmpty(),
		field.String("local_root_display").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			NotEmpty(),
		field.String("local_root_redacted").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.String("os").
			MaxLen(32).
			Validate(validateCcgoOS),
		field.String("path_style").
			MaxLen(32).
			Validate(validateCcgoPathStyle),
		field.String("device_id").
			MaxLen(128).
			Default(""),
		field.String("status").
			MaxLen(32).
			Default("active").
			Validate(validateCcgoWorkspaceStatus),
		field.Time("last_seen_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (CcgoWorkspace) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "local_root_hash").
			Unique(),
		index.Fields("user_id", "device_id"),
		index.Fields("status"),
	}
}

func validateCcgoOS(value string) error {
	switch value {
	case "darwin", "linux", "windows":
		return nil
	default:
		return fmt.Errorf("invalid ccgo os")
	}
}

func validateCcgoPathStyle(value string) error {
	switch value {
	case "posix", "windows":
		return nil
	default:
		return fmt.Errorf("invalid ccgo path style")
	}
}

func validateCcgoWorkspaceStatus(value string) error {
	switch value {
	case "active", "archived":
		return nil
	default:
		return fmt.Errorf("invalid ccgo workspace status")
	}
}
