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

// UsageBillingEvent is the durable accounting event for an applied billable request.
type UsageBillingEvent struct {
	ent.Schema
}

func (UsageBillingEvent) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "usage_billing_events"},
	}
}

func (UsageBillingEvent) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (UsageBillingEvent) Fields() []ent.Field {
	return []ent.Field{
		field.String("request_id").
			MaxLen(255).
			NotEmpty(),
		field.Int64("api_key_id"),
		field.String("request_fingerprint").
			MaxLen(128).
			NotEmpty(),
		field.String("request_payload_hash").
			Optional().
			Nillable().
			MaxLen(128),
		field.Int64("user_id"),
		field.Int64("group_id").
			Optional().
			Nillable(),
		field.Int64("account_id"),
		field.String("account_type").
			Optional().
			Nillable().
			MaxLen(32),
		field.String("model").
			MaxLen(255).
			Default(""),
		field.String("service_tier").
			MaxLen(64).
			Default(""),
		field.String("reasoning_effort").
			MaxLen(64).
			Default(""),
		field.Int8("billing_type"),
		field.Int("input_tokens").
			Default(0),
		field.Int("output_tokens").
			Default(0),
		field.Int("cache_creation_tokens").
			Default(0),
		field.Int("cache_read_tokens").
			Default(0),
		field.Int("cache_creation_5m_tokens").
			Default(0),
		field.Int("cache_creation_1h_tokens").
			Default(0),
		field.Float("total_cost").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}).
			Default(0).
			Validate(nonNegativeFloat("total cost")),
		field.Float("actual_cost").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}).
			Default(0).
			Validate(nonNegativeFloat("actual cost")),
		field.Float("balance_cost").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}).
			Default(0).
			Validate(nonNegativeFloat("balance cost")),
		field.Float("subscription_cost").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}).
			Default(0).
			Validate(nonNegativeFloat("subscription cost")),
		field.Float("api_key_quota_cost").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}).
			Default(0).
			Validate(nonNegativeFloat("api key quota cost")),
		field.Float("api_key_rate_limit_cost").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}).
			Default(0).
			Validate(nonNegativeFloat("api key rate limit cost")),
		field.Float("account_quota_cost").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}).
			Default(0).
			Validate(nonNegativeFloat("account quota cost")),
		field.Bool("supply_reward_eligible").
			Default(false),
		field.Int64("supply_owner_user_id").
			Optional().
			Nillable(),
		field.Float("supply_reward_multiplier").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(1).
			Validate(nonNegativeFloat("supply reward multiplier")),
		field.String("supply_account_status").
			MaxLen(32).
			Default("none").
			Validate(func(value string) error {
				switch value {
				case "none", "testing", "pending_review", "schedulable", "paused", "rejected", "revoked":
					return nil
				default:
					return fmt.Errorf("invalid supply account status")
				}
			}),
		field.String("supply_source").
			MaxLen(32).
			Default("live").
			Validate(func(value string) error {
				switch value {
				case "live", "backfill":
					return nil
				default:
					return fmt.Errorf("invalid usage billing event source")
				}
			}),
	}
}

func (UsageBillingEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("request_id", "api_key_id").
			Unique(),
		index.Fields("user_id", "created_at"),
		index.Fields("group_id", "user_id", "created_at"),
		index.Fields("billing_type", "created_at"),
		index.Fields("account_id", "created_at"),
	}
}
