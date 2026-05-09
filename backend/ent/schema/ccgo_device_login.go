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

// CcgoDeviceLogin stores pending ccgo device-code login attempts.
type CcgoDeviceLogin struct {
	ent.Schema
}

func (CcgoDeviceLogin) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "ccgo_device_logins"},
	}
}

func (CcgoDeviceLogin) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (CcgoDeviceLogin) Fields() []ent.Field {
	return []ent.Field{
		field.String("device_code_hash").
			MaxLen(64).
			NotEmpty().
			Unique(),
		field.String("user_code_hash").
			MaxLen(64).
			NotEmpty().
			Unique(),
		field.Int64("user_id").
			Optional().
			Nillable(),
		field.String("device_id").
			MaxLen(128).
			Default(""),
		field.String("status").
			MaxLen(32).
			Default("pending").
			Validate(validateCcgoDeviceLoginStatus),
		field.Time("expires_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("approved_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("consumed_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (CcgoDeviceLogin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status", "expires_at"),
		index.Fields("user_id", "status"),
	}
}

func validateCcgoDeviceLoginStatus(value string) error {
	switch value {
	case "pending", "approved", "consumed", "expired":
		return nil
	default:
		return fmt.Errorf("invalid ccgo device login status")
	}
}
