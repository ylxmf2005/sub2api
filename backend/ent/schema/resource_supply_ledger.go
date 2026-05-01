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

// ResourceSupplyLedger stores immutable reward, transfer, and adjustment rows.
type ResourceSupplyLedger struct {
	ent.Schema
}

func (ResourceSupplyLedger) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "resource_supply_ledger"},
	}
}

func (ResourceSupplyLedger) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (ResourceSupplyLedger) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("owner_user_id"),
		field.Int64("caller_user_id").
			Optional().
			Nillable(),
		field.Int64("api_key_id").
			Optional().
			Nillable(),
		field.Int64("group_id").
			Optional().
			Nillable(),
		field.Int64("account_id").
			Optional().
			Nillable(),
		field.Int64("usage_billing_event_id").
			Optional().
			Nillable(),
		field.String("ledger_type").
			MaxLen(32).
			Validate(func(value string) error {
				switch value {
				case "reward", "transfer", "adjustment":
					return nil
				default:
					return fmt.Errorf("invalid resource supply ledger type")
				}
			}),
		field.String("idempotency_key").
			Optional().
			Nillable().
			MaxLen(255),
		field.Float("amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0),
		field.Float("balance_after").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Validate(nonNegativeFloat("balance after")).
			Default(0),
		field.Float("actual_cost").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}),
		field.Float("reward_multiplier").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Int8("billing_type").
			Optional().
			Nillable(),
		field.String("model").
			Optional().
			Nillable().
			MaxLen(255),
		field.String("request_id").
			Optional().
			Nillable().
			MaxLen(255),
		field.Int64("admin_user_id").
			Optional().
			Nillable(),
		field.String("note").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
	}
}

func (ResourceSupplyLedger) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("owner_user_id", "created_at"),
		index.Fields("caller_user_id", "created_at"),
		index.Fields("group_id", "created_at"),
		index.Fields("account_id", "created_at"),
		index.Fields("request_id"),
		index.Fields("usage_billing_event_id").
			Unique().
			Annotations(entsql.IndexWhere("ledger_type = 'reward' AND usage_billing_event_id IS NOT NULL")),
		index.Fields("owner_user_id", "idempotency_key").
			Unique(),
	}
}
