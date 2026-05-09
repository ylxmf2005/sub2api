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

// CcgoAgentCredential stores hashed, short-lived local-agent credentials.
type CcgoAgentCredential struct {
	ent.Schema
}

func (CcgoAgentCredential) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "ccgo_agent_credentials"},
	}
}

func (CcgoAgentCredential) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (CcgoAgentCredential) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("workspace_id"),
		field.Int64("user_id"),
		field.String("token_hash").
			MaxLen(64).
			NotEmpty().
			Unique(),
		field.String("nonce_hash").
			MaxLen(64).
			Default(""),
		field.Time("expires_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("used_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("revoked_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("status").
			MaxLen(32).
			Default("active").
			Validate(validateCcgoCredentialStatus),
	}
}

func (CcgoAgentCredential) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_id", "status"),
		index.Fields("user_id", "status"),
		index.Fields("expires_at"),
	}
}

func validateCcgoCredentialStatus(value string) error {
	switch value {
	case "active", "used", "revoked", "expired":
		return nil
	default:
		return fmt.Errorf("invalid ccgo credential status")
	}
}
