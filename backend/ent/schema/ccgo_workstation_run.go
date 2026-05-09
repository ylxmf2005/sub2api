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

// CcgoWorkstationRun stores disposable Claude Code process lifecycle state.
type CcgoWorkstationRun struct {
	ent.Schema
}

func (CcgoWorkstationRun) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "ccgo_workstation_runs"},
	}
}

func (CcgoWorkstationRun) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (CcgoWorkstationRun) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("workspace_id"),
		field.Int64("user_id"),
		field.String("run_id").
			MaxLen(64).
			NotEmpty().
			Unique(),
		field.String("status").
			MaxLen(32).
			Default("starting").
			Validate(validateCcgoRunStatus),
		field.String("server_pid").
			MaxLen(64).
			Default(""),
		field.Time("started_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("stopped_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("stop_reason").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.Time("last_heartbeat_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (CcgoWorkstationRun) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "status"),
		index.Fields("user_id", "status"),
		index.Fields("last_heartbeat_at"),
	}
}

func validateCcgoRunStatus(value string) error {
	switch value {
	case "starting", "running", "stopping", "stopped", "failed":
		return nil
	default:
		return fmt.Errorf("invalid ccgo workstation run status")
	}
}
