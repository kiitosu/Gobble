package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

/*********
  Game
*********/
// Game holds the schema definition for the Game entity.
type Game struct {
	ent.Schema
}

// Fields of the Game.
func (Game) Fields() []ent.Field {
	return []ent.Field{
		field.Text("name").NotEmpty(),
		field.Enum("status").
			Values("CREATED", "STARTED", "FINISHED").
			Default("CREATED"),
	}
}

// Edges of the Game.
func (Game) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("players", Player.Type).Ref("parent"),
	}
}

/*********
  Player
*********/
// Player holds the schema definition for the Player entity.
type Player struct {
	ent.Schema
}

// Fields of the Player.
func (Player) Fields() []ent.Field {
	return []ent.Field{
		field.Text("name").NotEmpty(),
		field.Enum("status").
			Values("JOINING", "STARTING", "READY", "PLAYING", "FINISHED").
			Default("JOINING"),
	}
}

// Edges of the Player.
func (Player) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("parent", Game.Type).Unique(),
	}
}

/*********
  Card
*********/

// Card holds the schema definition for the Card entity.
type Card struct {
	ent.Schema
}

// Fields of the Card.
func (Card) Fields() []ent.Field {
	return []ent.Field{}
}

// Edges of the Card.
func (Card) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("items", Item.Type).Ref("parent"),
	}
}

/*********
  ITEM
*********/

type Item struct {
	ent.Schema
}

func (Item) Fields() []ent.Field {
	return []ent.Field{
		field.Text("symbol"),
	}
}

func (Item) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("parent", Card.Type),
	}
}

/* sample
func (Todo) Fields() []ent.Field {
	return []ent.Field{
		field.Text("text").NotEmpty(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Enum("status").NamedValues(
			"InProgress", "IN_PROGRESS",
			"Completed", "COMPLETED",
		).Default("IN_PROGRESS"),
		field.Int("priority").Default(0),
	}
}

// Edges of the Todo.
func (Todo) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("parent", Todo.Type). // Todoを親とするエッジを追加
			Unique(). // 親は単一
			From("children"), //自分自身(今回はTodo)を子として引けるようにする
	}
}
*/
