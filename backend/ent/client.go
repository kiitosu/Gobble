// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"

	"example/ent/migrate"

	"example/ent/card"
	"example/ent/game"
	"example/ent/item"
	"example/ent/player"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
)

// Client is the client that holds all ent builders.
type Client struct {
	config
	// Schema is the client for creating, migrating and dropping schema.
	Schema *migrate.Schema
	// Card is the client for interacting with the Card builders.
	Card *CardClient
	// Game is the client for interacting with the Game builders.
	Game *GameClient
	// Item is the client for interacting with the Item builders.
	Item *ItemClient
	// Player is the client for interacting with the Player builders.
	Player *PlayerClient
}

// NewClient creates a new client configured with the given options.
func NewClient(opts ...Option) *Client {
	client := &Client{config: newConfig(opts...)}
	client.init()
	return client
}

func (c *Client) init() {
	c.Schema = migrate.NewSchema(c.driver)
	c.Card = NewCardClient(c.config)
	c.Game = NewGameClient(c.config)
	c.Item = NewItemClient(c.config)
	c.Player = NewPlayerClient(c.config)
}

type (
	// config is the configuration for the client and its builder.
	config struct {
		// driver used for executing database requests.
		driver dialect.Driver
		// debug enable a debug logging.
		debug bool
		// log used for logging on debug mode.
		log func(...any)
		// hooks to execute on mutations.
		hooks *hooks
		// interceptors to execute on queries.
		inters *inters
	}
	// Option function to configure the client.
	Option func(*config)
)

// newConfig creates a new config for the client.
func newConfig(opts ...Option) config {
	cfg := config{log: log.Println, hooks: &hooks{}, inters: &inters{}}
	cfg.options(opts...)
	return cfg
}

// options applies the options on the config object.
func (c *config) options(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
	if c.debug {
		c.driver = dialect.Debug(c.driver, c.log)
	}
}

// Debug enables debug logging on the ent.Driver.
func Debug() Option {
	return func(c *config) {
		c.debug = true
	}
}

// Log sets the logging function for debug mode.
func Log(fn func(...any)) Option {
	return func(c *config) {
		c.log = fn
	}
}

// Driver configures the client driver.
func Driver(driver dialect.Driver) Option {
	return func(c *config) {
		c.driver = driver
	}
}

// Open opens a database/sql.DB specified by the driver name and
// the data source name, and returns a new client attached to it.
// Optional parameters can be added for configuring the client.
func Open(driverName, dataSourceName string, options ...Option) (*Client, error) {
	switch driverName {
	case dialect.MySQL, dialect.Postgres, dialect.SQLite:
		drv, err := sql.Open(driverName, dataSourceName)
		if err != nil {
			return nil, err
		}
		return NewClient(append(options, Driver(drv))...), nil
	default:
		return nil, fmt.Errorf("unsupported driver: %q", driverName)
	}
}

// ErrTxStarted is returned when trying to start a new transaction from a transactional client.
var ErrTxStarted = errors.New("ent: cannot start a transaction within a transaction")

// Tx returns a new transactional client. The provided context
// is used until the transaction is committed or rolled back.
func (c *Client) Tx(ctx context.Context) (*Tx, error) {
	if _, ok := c.driver.(*txDriver); ok {
		return nil, ErrTxStarted
	}
	tx, err := newTx(ctx, c.driver)
	if err != nil {
		return nil, fmt.Errorf("ent: starting a transaction: %w", err)
	}
	cfg := c.config
	cfg.driver = tx
	return &Tx{
		ctx:    ctx,
		config: cfg,
		Card:   NewCardClient(cfg),
		Game:   NewGameClient(cfg),
		Item:   NewItemClient(cfg),
		Player: NewPlayerClient(cfg),
	}, nil
}

// BeginTx returns a transactional client with specified options.
func (c *Client) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	if _, ok := c.driver.(*txDriver); ok {
		return nil, errors.New("ent: cannot start a transaction within a transaction")
	}
	tx, err := c.driver.(interface {
		BeginTx(context.Context, *sql.TxOptions) (dialect.Tx, error)
	}).BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("ent: starting a transaction: %w", err)
	}
	cfg := c.config
	cfg.driver = &txDriver{tx: tx, drv: c.driver}
	return &Tx{
		ctx:    ctx,
		config: cfg,
		Card:   NewCardClient(cfg),
		Game:   NewGameClient(cfg),
		Item:   NewItemClient(cfg),
		Player: NewPlayerClient(cfg),
	}, nil
}

// Debug returns a new debug-client. It's used to get verbose logging on specific operations.
//
//	client.Debug().
//		Card.
//		Query().
//		Count(ctx)
func (c *Client) Debug() *Client {
	if c.debug {
		return c
	}
	cfg := c.config
	cfg.driver = dialect.Debug(c.driver, c.log)
	client := &Client{config: cfg}
	client.init()
	return client
}

// Close closes the database connection and prevents new queries from starting.
func (c *Client) Close() error {
	return c.driver.Close()
}

// Use adds the mutation hooks to all the entity clients.
// In order to add hooks to a specific client, call: `client.Node.Use(...)`.
func (c *Client) Use(hooks ...Hook) {
	c.Card.Use(hooks...)
	c.Game.Use(hooks...)
	c.Item.Use(hooks...)
	c.Player.Use(hooks...)
}

// Intercept adds the query interceptors to all the entity clients.
// In order to add interceptors to a specific client, call: `client.Node.Intercept(...)`.
func (c *Client) Intercept(interceptors ...Interceptor) {
	c.Card.Intercept(interceptors...)
	c.Game.Intercept(interceptors...)
	c.Item.Intercept(interceptors...)
	c.Player.Intercept(interceptors...)
}

// Mutate implements the ent.Mutator interface.
func (c *Client) Mutate(ctx context.Context, m Mutation) (Value, error) {
	switch m := m.(type) {
	case *CardMutation:
		return c.Card.mutate(ctx, m)
	case *GameMutation:
		return c.Game.mutate(ctx, m)
	case *ItemMutation:
		return c.Item.mutate(ctx, m)
	case *PlayerMutation:
		return c.Player.mutate(ctx, m)
	default:
		return nil, fmt.Errorf("ent: unknown mutation type %T", m)
	}
}

// CardClient is a client for the Card schema.
type CardClient struct {
	config
}

// NewCardClient returns a client for the Card from the given config.
func NewCardClient(c config) *CardClient {
	return &CardClient{config: c}
}

// Use adds a list of mutation hooks to the hooks stack.
// A call to `Use(f, g, h)` equals to `card.Hooks(f(g(h())))`.
func (c *CardClient) Use(hooks ...Hook) {
	c.hooks.Card = append(c.hooks.Card, hooks...)
}

// Intercept adds a list of query interceptors to the interceptors stack.
// A call to `Intercept(f, g, h)` equals to `card.Intercept(f(g(h())))`.
func (c *CardClient) Intercept(interceptors ...Interceptor) {
	c.inters.Card = append(c.inters.Card, interceptors...)
}

// Create returns a builder for creating a Card entity.
func (c *CardClient) Create() *CardCreate {
	mutation := newCardMutation(c.config, OpCreate)
	return &CardCreate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// CreateBulk returns a builder for creating a bulk of Card entities.
func (c *CardClient) CreateBulk(builders ...*CardCreate) *CardCreateBulk {
	return &CardCreateBulk{config: c.config, builders: builders}
}

// MapCreateBulk creates a bulk creation builder from the given slice. For each item in the slice, the function creates
// a builder and applies setFunc on it.
func (c *CardClient) MapCreateBulk(slice any, setFunc func(*CardCreate, int)) *CardCreateBulk {
	rv := reflect.ValueOf(slice)
	if rv.Kind() != reflect.Slice {
		return &CardCreateBulk{err: fmt.Errorf("calling to CardClient.MapCreateBulk with wrong type %T, need slice", slice)}
	}
	builders := make([]*CardCreate, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		builders[i] = c.Create()
		setFunc(builders[i], i)
	}
	return &CardCreateBulk{config: c.config, builders: builders}
}

// Update returns an update builder for Card.
func (c *CardClient) Update() *CardUpdate {
	mutation := newCardMutation(c.config, OpUpdate)
	return &CardUpdate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOne returns an update builder for the given entity.
func (c *CardClient) UpdateOne(ca *Card) *CardUpdateOne {
	mutation := newCardMutation(c.config, OpUpdateOne, withCard(ca))
	return &CardUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOneID returns an update builder for the given id.
func (c *CardClient) UpdateOneID(id int) *CardUpdateOne {
	mutation := newCardMutation(c.config, OpUpdateOne, withCardID(id))
	return &CardUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// Delete returns a delete builder for Card.
func (c *CardClient) Delete() *CardDelete {
	mutation := newCardMutation(c.config, OpDelete)
	return &CardDelete{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// DeleteOne returns a builder for deleting the given entity.
func (c *CardClient) DeleteOne(ca *Card) *CardDeleteOne {
	return c.DeleteOneID(ca.ID)
}

// DeleteOneID returns a builder for deleting the given entity by its id.
func (c *CardClient) DeleteOneID(id int) *CardDeleteOne {
	builder := c.Delete().Where(card.ID(id))
	builder.mutation.id = &id
	builder.mutation.op = OpDeleteOne
	return &CardDeleteOne{builder}
}

// Query returns a query builder for Card.
func (c *CardClient) Query() *CardQuery {
	return &CardQuery{
		config: c.config,
		ctx:    &QueryContext{Type: TypeCard},
		inters: c.Interceptors(),
	}
}

// Get returns a Card entity by its id.
func (c *CardClient) Get(ctx context.Context, id int) (*Card, error) {
	return c.Query().Where(card.ID(id)).Only(ctx)
}

// GetX is like Get, but panics if an error occurs.
func (c *CardClient) GetX(ctx context.Context, id int) *Card {
	obj, err := c.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return obj
}

// QueryItems queries the items edge of a Card.
func (c *CardClient) QueryItems(ca *Card) *ItemQuery {
	query := (&ItemClient{config: c.config}).Query()
	query.path = func(context.Context) (fromV *sql.Selector, _ error) {
		id := ca.ID
		step := sqlgraph.NewStep(
			sqlgraph.From(card.Table, card.FieldID, id),
			sqlgraph.To(item.Table, item.FieldID),
			sqlgraph.Edge(sqlgraph.M2M, true, card.ItemsTable, card.ItemsPrimaryKey...),
		)
		fromV = sqlgraph.Neighbors(ca.driver.Dialect(), step)
		return fromV, nil
	}
	return query
}

// Hooks returns the client hooks.
func (c *CardClient) Hooks() []Hook {
	return c.hooks.Card
}

// Interceptors returns the client interceptors.
func (c *CardClient) Interceptors() []Interceptor {
	return c.inters.Card
}

func (c *CardClient) mutate(ctx context.Context, m *CardMutation) (Value, error) {
	switch m.Op() {
	case OpCreate:
		return (&CardCreate{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpUpdate:
		return (&CardUpdate{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpUpdateOne:
		return (&CardUpdateOne{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpDelete, OpDeleteOne:
		return (&CardDelete{config: c.config, hooks: c.Hooks(), mutation: m}).Exec(ctx)
	default:
		return nil, fmt.Errorf("ent: unknown Card mutation op: %q", m.Op())
	}
}

// GameClient is a client for the Game schema.
type GameClient struct {
	config
}

// NewGameClient returns a client for the Game from the given config.
func NewGameClient(c config) *GameClient {
	return &GameClient{config: c}
}

// Use adds a list of mutation hooks to the hooks stack.
// A call to `Use(f, g, h)` equals to `game.Hooks(f(g(h())))`.
func (c *GameClient) Use(hooks ...Hook) {
	c.hooks.Game = append(c.hooks.Game, hooks...)
}

// Intercept adds a list of query interceptors to the interceptors stack.
// A call to `Intercept(f, g, h)` equals to `game.Intercept(f(g(h())))`.
func (c *GameClient) Intercept(interceptors ...Interceptor) {
	c.inters.Game = append(c.inters.Game, interceptors...)
}

// Create returns a builder for creating a Game entity.
func (c *GameClient) Create() *GameCreate {
	mutation := newGameMutation(c.config, OpCreate)
	return &GameCreate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// CreateBulk returns a builder for creating a bulk of Game entities.
func (c *GameClient) CreateBulk(builders ...*GameCreate) *GameCreateBulk {
	return &GameCreateBulk{config: c.config, builders: builders}
}

// MapCreateBulk creates a bulk creation builder from the given slice. For each item in the slice, the function creates
// a builder and applies setFunc on it.
func (c *GameClient) MapCreateBulk(slice any, setFunc func(*GameCreate, int)) *GameCreateBulk {
	rv := reflect.ValueOf(slice)
	if rv.Kind() != reflect.Slice {
		return &GameCreateBulk{err: fmt.Errorf("calling to GameClient.MapCreateBulk with wrong type %T, need slice", slice)}
	}
	builders := make([]*GameCreate, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		builders[i] = c.Create()
		setFunc(builders[i], i)
	}
	return &GameCreateBulk{config: c.config, builders: builders}
}

// Update returns an update builder for Game.
func (c *GameClient) Update() *GameUpdate {
	mutation := newGameMutation(c.config, OpUpdate)
	return &GameUpdate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOne returns an update builder for the given entity.
func (c *GameClient) UpdateOne(ga *Game) *GameUpdateOne {
	mutation := newGameMutation(c.config, OpUpdateOne, withGame(ga))
	return &GameUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOneID returns an update builder for the given id.
func (c *GameClient) UpdateOneID(id int) *GameUpdateOne {
	mutation := newGameMutation(c.config, OpUpdateOne, withGameID(id))
	return &GameUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// Delete returns a delete builder for Game.
func (c *GameClient) Delete() *GameDelete {
	mutation := newGameMutation(c.config, OpDelete)
	return &GameDelete{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// DeleteOne returns a builder for deleting the given entity.
func (c *GameClient) DeleteOne(ga *Game) *GameDeleteOne {
	return c.DeleteOneID(ga.ID)
}

// DeleteOneID returns a builder for deleting the given entity by its id.
func (c *GameClient) DeleteOneID(id int) *GameDeleteOne {
	builder := c.Delete().Where(game.ID(id))
	builder.mutation.id = &id
	builder.mutation.op = OpDeleteOne
	return &GameDeleteOne{builder}
}

// Query returns a query builder for Game.
func (c *GameClient) Query() *GameQuery {
	return &GameQuery{
		config: c.config,
		ctx:    &QueryContext{Type: TypeGame},
		inters: c.Interceptors(),
	}
}

// Get returns a Game entity by its id.
func (c *GameClient) Get(ctx context.Context, id int) (*Game, error) {
	return c.Query().Where(game.ID(id)).Only(ctx)
}

// GetX is like Get, but panics if an error occurs.
func (c *GameClient) GetX(ctx context.Context, id int) *Game {
	obj, err := c.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return obj
}

// QueryPlayers queries the players edge of a Game.
func (c *GameClient) QueryPlayers(ga *Game) *PlayerQuery {
	query := (&PlayerClient{config: c.config}).Query()
	query.path = func(context.Context) (fromV *sql.Selector, _ error) {
		id := ga.ID
		step := sqlgraph.NewStep(
			sqlgraph.From(game.Table, game.FieldID, id),
			sqlgraph.To(player.Table, player.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, true, game.PlayersTable, game.PlayersColumn),
		)
		fromV = sqlgraph.Neighbors(ga.driver.Dialect(), step)
		return fromV, nil
	}
	return query
}

// Hooks returns the client hooks.
func (c *GameClient) Hooks() []Hook {
	return c.hooks.Game
}

// Interceptors returns the client interceptors.
func (c *GameClient) Interceptors() []Interceptor {
	return c.inters.Game
}

func (c *GameClient) mutate(ctx context.Context, m *GameMutation) (Value, error) {
	switch m.Op() {
	case OpCreate:
		return (&GameCreate{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpUpdate:
		return (&GameUpdate{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpUpdateOne:
		return (&GameUpdateOne{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpDelete, OpDeleteOne:
		return (&GameDelete{config: c.config, hooks: c.Hooks(), mutation: m}).Exec(ctx)
	default:
		return nil, fmt.Errorf("ent: unknown Game mutation op: %q", m.Op())
	}
}

// ItemClient is a client for the Item schema.
type ItemClient struct {
	config
}

// NewItemClient returns a client for the Item from the given config.
func NewItemClient(c config) *ItemClient {
	return &ItemClient{config: c}
}

// Use adds a list of mutation hooks to the hooks stack.
// A call to `Use(f, g, h)` equals to `item.Hooks(f(g(h())))`.
func (c *ItemClient) Use(hooks ...Hook) {
	c.hooks.Item = append(c.hooks.Item, hooks...)
}

// Intercept adds a list of query interceptors to the interceptors stack.
// A call to `Intercept(f, g, h)` equals to `item.Intercept(f(g(h())))`.
func (c *ItemClient) Intercept(interceptors ...Interceptor) {
	c.inters.Item = append(c.inters.Item, interceptors...)
}

// Create returns a builder for creating a Item entity.
func (c *ItemClient) Create() *ItemCreate {
	mutation := newItemMutation(c.config, OpCreate)
	return &ItemCreate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// CreateBulk returns a builder for creating a bulk of Item entities.
func (c *ItemClient) CreateBulk(builders ...*ItemCreate) *ItemCreateBulk {
	return &ItemCreateBulk{config: c.config, builders: builders}
}

// MapCreateBulk creates a bulk creation builder from the given slice. For each item in the slice, the function creates
// a builder and applies setFunc on it.
func (c *ItemClient) MapCreateBulk(slice any, setFunc func(*ItemCreate, int)) *ItemCreateBulk {
	rv := reflect.ValueOf(slice)
	if rv.Kind() != reflect.Slice {
		return &ItemCreateBulk{err: fmt.Errorf("calling to ItemClient.MapCreateBulk with wrong type %T, need slice", slice)}
	}
	builders := make([]*ItemCreate, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		builders[i] = c.Create()
		setFunc(builders[i], i)
	}
	return &ItemCreateBulk{config: c.config, builders: builders}
}

// Update returns an update builder for Item.
func (c *ItemClient) Update() *ItemUpdate {
	mutation := newItemMutation(c.config, OpUpdate)
	return &ItemUpdate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOne returns an update builder for the given entity.
func (c *ItemClient) UpdateOne(i *Item) *ItemUpdateOne {
	mutation := newItemMutation(c.config, OpUpdateOne, withItem(i))
	return &ItemUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOneID returns an update builder for the given id.
func (c *ItemClient) UpdateOneID(id int) *ItemUpdateOne {
	mutation := newItemMutation(c.config, OpUpdateOne, withItemID(id))
	return &ItemUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// Delete returns a delete builder for Item.
func (c *ItemClient) Delete() *ItemDelete {
	mutation := newItemMutation(c.config, OpDelete)
	return &ItemDelete{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// DeleteOne returns a builder for deleting the given entity.
func (c *ItemClient) DeleteOne(i *Item) *ItemDeleteOne {
	return c.DeleteOneID(i.ID)
}

// DeleteOneID returns a builder for deleting the given entity by its id.
func (c *ItemClient) DeleteOneID(id int) *ItemDeleteOne {
	builder := c.Delete().Where(item.ID(id))
	builder.mutation.id = &id
	builder.mutation.op = OpDeleteOne
	return &ItemDeleteOne{builder}
}

// Query returns a query builder for Item.
func (c *ItemClient) Query() *ItemQuery {
	return &ItemQuery{
		config: c.config,
		ctx:    &QueryContext{Type: TypeItem},
		inters: c.Interceptors(),
	}
}

// Get returns a Item entity by its id.
func (c *ItemClient) Get(ctx context.Context, id int) (*Item, error) {
	return c.Query().Where(item.ID(id)).Only(ctx)
}

// GetX is like Get, but panics if an error occurs.
func (c *ItemClient) GetX(ctx context.Context, id int) *Item {
	obj, err := c.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return obj
}

// QueryParent queries the parent edge of a Item.
func (c *ItemClient) QueryParent(i *Item) *CardQuery {
	query := (&CardClient{config: c.config}).Query()
	query.path = func(context.Context) (fromV *sql.Selector, _ error) {
		id := i.ID
		step := sqlgraph.NewStep(
			sqlgraph.From(item.Table, item.FieldID, id),
			sqlgraph.To(card.Table, card.FieldID),
			sqlgraph.Edge(sqlgraph.M2M, false, item.ParentTable, item.ParentPrimaryKey...),
		)
		fromV = sqlgraph.Neighbors(i.driver.Dialect(), step)
		return fromV, nil
	}
	return query
}

// Hooks returns the client hooks.
func (c *ItemClient) Hooks() []Hook {
	return c.hooks.Item
}

// Interceptors returns the client interceptors.
func (c *ItemClient) Interceptors() []Interceptor {
	return c.inters.Item
}

func (c *ItemClient) mutate(ctx context.Context, m *ItemMutation) (Value, error) {
	switch m.Op() {
	case OpCreate:
		return (&ItemCreate{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpUpdate:
		return (&ItemUpdate{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpUpdateOne:
		return (&ItemUpdateOne{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpDelete, OpDeleteOne:
		return (&ItemDelete{config: c.config, hooks: c.Hooks(), mutation: m}).Exec(ctx)
	default:
		return nil, fmt.Errorf("ent: unknown Item mutation op: %q", m.Op())
	}
}

// PlayerClient is a client for the Player schema.
type PlayerClient struct {
	config
}

// NewPlayerClient returns a client for the Player from the given config.
func NewPlayerClient(c config) *PlayerClient {
	return &PlayerClient{config: c}
}

// Use adds a list of mutation hooks to the hooks stack.
// A call to `Use(f, g, h)` equals to `player.Hooks(f(g(h())))`.
func (c *PlayerClient) Use(hooks ...Hook) {
	c.hooks.Player = append(c.hooks.Player, hooks...)
}

// Intercept adds a list of query interceptors to the interceptors stack.
// A call to `Intercept(f, g, h)` equals to `player.Intercept(f(g(h())))`.
func (c *PlayerClient) Intercept(interceptors ...Interceptor) {
	c.inters.Player = append(c.inters.Player, interceptors...)
}

// Create returns a builder for creating a Player entity.
func (c *PlayerClient) Create() *PlayerCreate {
	mutation := newPlayerMutation(c.config, OpCreate)
	return &PlayerCreate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// CreateBulk returns a builder for creating a bulk of Player entities.
func (c *PlayerClient) CreateBulk(builders ...*PlayerCreate) *PlayerCreateBulk {
	return &PlayerCreateBulk{config: c.config, builders: builders}
}

// MapCreateBulk creates a bulk creation builder from the given slice. For each item in the slice, the function creates
// a builder and applies setFunc on it.
func (c *PlayerClient) MapCreateBulk(slice any, setFunc func(*PlayerCreate, int)) *PlayerCreateBulk {
	rv := reflect.ValueOf(slice)
	if rv.Kind() != reflect.Slice {
		return &PlayerCreateBulk{err: fmt.Errorf("calling to PlayerClient.MapCreateBulk with wrong type %T, need slice", slice)}
	}
	builders := make([]*PlayerCreate, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		builders[i] = c.Create()
		setFunc(builders[i], i)
	}
	return &PlayerCreateBulk{config: c.config, builders: builders}
}

// Update returns an update builder for Player.
func (c *PlayerClient) Update() *PlayerUpdate {
	mutation := newPlayerMutation(c.config, OpUpdate)
	return &PlayerUpdate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOne returns an update builder for the given entity.
func (c *PlayerClient) UpdateOne(pl *Player) *PlayerUpdateOne {
	mutation := newPlayerMutation(c.config, OpUpdateOne, withPlayer(pl))
	return &PlayerUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOneID returns an update builder for the given id.
func (c *PlayerClient) UpdateOneID(id int) *PlayerUpdateOne {
	mutation := newPlayerMutation(c.config, OpUpdateOne, withPlayerID(id))
	return &PlayerUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// Delete returns a delete builder for Player.
func (c *PlayerClient) Delete() *PlayerDelete {
	mutation := newPlayerMutation(c.config, OpDelete)
	return &PlayerDelete{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// DeleteOne returns a builder for deleting the given entity.
func (c *PlayerClient) DeleteOne(pl *Player) *PlayerDeleteOne {
	return c.DeleteOneID(pl.ID)
}

// DeleteOneID returns a builder for deleting the given entity by its id.
func (c *PlayerClient) DeleteOneID(id int) *PlayerDeleteOne {
	builder := c.Delete().Where(player.ID(id))
	builder.mutation.id = &id
	builder.mutation.op = OpDeleteOne
	return &PlayerDeleteOne{builder}
}

// Query returns a query builder for Player.
func (c *PlayerClient) Query() *PlayerQuery {
	return &PlayerQuery{
		config: c.config,
		ctx:    &QueryContext{Type: TypePlayer},
		inters: c.Interceptors(),
	}
}

// Get returns a Player entity by its id.
func (c *PlayerClient) Get(ctx context.Context, id int) (*Player, error) {
	return c.Query().Where(player.ID(id)).Only(ctx)
}

// GetX is like Get, but panics if an error occurs.
func (c *PlayerClient) GetX(ctx context.Context, id int) *Player {
	obj, err := c.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return obj
}

// QueryParent queries the parent edge of a Player.
func (c *PlayerClient) QueryParent(pl *Player) *GameQuery {
	query := (&GameClient{config: c.config}).Query()
	query.path = func(context.Context) (fromV *sql.Selector, _ error) {
		id := pl.ID
		step := sqlgraph.NewStep(
			sqlgraph.From(player.Table, player.FieldID, id),
			sqlgraph.To(game.Table, game.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, player.ParentTable, player.ParentColumn),
		)
		fromV = sqlgraph.Neighbors(pl.driver.Dialect(), step)
		return fromV, nil
	}
	return query
}

// Hooks returns the client hooks.
func (c *PlayerClient) Hooks() []Hook {
	return c.hooks.Player
}

// Interceptors returns the client interceptors.
func (c *PlayerClient) Interceptors() []Interceptor {
	return c.inters.Player
}

func (c *PlayerClient) mutate(ctx context.Context, m *PlayerMutation) (Value, error) {
	switch m.Op() {
	case OpCreate:
		return (&PlayerCreate{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpUpdate:
		return (&PlayerUpdate{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpUpdateOne:
		return (&PlayerUpdateOne{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpDelete, OpDeleteOne:
		return (&PlayerDelete{config: c.config, hooks: c.Hooks(), mutation: m}).Exec(ctx)
	default:
		return nil, fmt.Errorf("ent: unknown Player mutation op: %q", m.Op())
	}
}

// hooks and interceptors per client, for fast access.
type (
	hooks struct {
		Card, Game, Item, Player []ent.Hook
	}
	inters struct {
		Card, Game, Item, Player []ent.Interceptor
	}
)
