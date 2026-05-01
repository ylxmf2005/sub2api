package schema

import (
	"fmt"
	"math"

	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// ResourceSupplyBalance stores the aggregate supply-credit wallet for an owner.
type ResourceSupplyBalance struct {
	ent.Schema
}

func (ResourceSupplyBalance) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "resource_supply_balances"},
	}
}

func (ResourceSupplyBalance) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (ResourceSupplyBalance) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").
			Unique(),
		field.Float("available_amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0).
			Validate(nonNegativeFloat("available amount")),
		field.Float("lifetime_earned_amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0).
			Validate(nonNegativeFloat("lifetime earned amount")),
		field.Float("lifetime_transferred_amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0).
			Validate(nonNegativeFloat("lifetime transferred amount")),
	}
}

func (ResourceSupplyBalance) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("resource_supply_balance").
			Field("user_id").
			Unique().
			Required(),
	}
}

func nonNegativeFloat(label string) func(float64) error {
	return func(value float64) error {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return fmt.Errorf("%s must be non-negative", label)
		}
		return nil
	}
}
