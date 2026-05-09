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

// CcgoCommandAudit stores command execution metadata without raw output.
type CcgoCommandAudit struct {
	ent.Schema
}

func (CcgoCommandAudit) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "ccgo_command_audits"},
	}
}

func (CcgoCommandAudit) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (CcgoCommandAudit) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("workspace_id"),
		field.Int64("user_id"),
		field.Int64("run_id").
			Optional().
			Nillable(),
		field.String("request_id").
			MaxLen(64).
			NotEmpty().
			Unique(),
		field.String("command_hash").
			MaxLen(64).
			NotEmpty(),
		field.String("redacted_command").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.String("server_cwd").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			NotEmpty(),
		field.String("local_cwd").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			NotEmpty(),
		field.Int("exit_code").
			Optional().
			Nillable(),
		field.String("status").
			MaxLen(32).
			Default("requested").
			Validate(validateCcgoCommandAuditStatus),
		field.String("failure_reason").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.Time("started_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("finished_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Int64("duration_ms").
			Default(0),
	}
}

func validateCcgoCommandAuditStatus(value string) error {
	switch value {
	case "requested", "succeeded", "failed", "timeout", "not_executed":
		return nil
	default:
		return fmt.Errorf("invalid ccgo command audit status")
	}
}

func (CcgoCommandAudit) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "created_at"),
		index.Fields("user_id", "created_at"),
		index.Fields("status", "created_at"),
	}
}
