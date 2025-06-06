package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	g "example/ent/game"
	"example/ent/player"
	gamev1 "example/gen/game/v1"
	"example/gen/game/v1/gamev1connect"
	"example/internal/cardgen"

	"github.com/gorilla/websocket"

	"example/ent"

	"encoding/json"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
)

type ClientInfo struct {
	GameID   int
	PlayerID int
}

var gameClients = make(map[int]map[*websocket.Conn]ClientInfo)

const DB_FILE = "file:backend/.db/ent.db?_fk=1"

func GetDbClient(
	ctx context.Context,
) *ent.Client {
	client, err := ent.Open(dialect.SQLite, DB_FILE)
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}
	return client
}

/* servers */
type GameServer struct{}

func (s *GameServer) JoinGame(
	ctx context.Context,
	req *connect.Request[gamev1.JoinGameRequest],
) (*connect.Response[gamev1.JoinGameResponse], error) {
	client := GetDbClient(ctx)
	defer client.Close()

	// レコード追加
	game_id := req.Msg.GameId
	player_name := req.Msg.PlayerName
	gameIDInt, err := strconv.Atoi(game_id) // str to int
	if err != nil {
		log.Printf("invalid game_id: %v", err)
		return nil, err
	}
	log.Printf("gameIDInt is %d", gameIDInt)

	player, err := client.Player.Create().SetName(player_name).SetParentID(gameIDInt).Save(ctx)
	if err != nil {
		log.Printf("failed creating a todo: %v", err)
		return nil, err
	}
	log.Printf("player is %v", player)

	res := connect.NewResponse(&gamev1.JoinGameResponse{
		Player: &gamev1.Player{
			GameId: int32(gameIDInt),
			Id:     int32(player.ID),
			Name:   player_name,
		},
	})

	log.Printf("User %s join the game %s", player_name, game_id)
	return res, nil
}

func (s *GameServer) CreateGame(
	ctx context.Context,
	req *connect.Request[gamev1.CreateGameRequest],
) (*connect.Response[gamev1.CreateGameResponse], error) {
	client := GetDbClient(ctx)
	defer client.Close()

	// レコード追加
	game_name := req.Msg.GameName
	game, err := client.Game.Create().SetName(game_name).Save(ctx)
	if err != nil {
		log.Printf("failed creating a todo: %v", err)
		return nil, err
	}
	player_name := req.Msg.PlayerName
	player, err := client.Player.Create().SetName(player_name).SetParent(game).Save(ctx)
	if err != nil {
		log.Printf("failed creating a todo: %v", err)
		return nil, err
	}

	// Dobbleカード生成
	cards, _, err := cardgen.GenerateDobbleCards(3)
	if err != nil {
		log.Printf("failed to generate dobble cards: %v", err)
	} else {
		var cs []Card
		for _, c := range cards {
			cs = append(cs, Card{
				ID:   c.ID,
				Text: "symbols: " + fmt.Sprint(c.Symbols),
			})
		}
		unsentCards[game.ID] = cs
	}

	res := connect.NewResponse(&gamev1.CreateGameResponse{
		Player: &gamev1.Player{
			GameId: int32(game.ID),
			Id:     int32(player.ID),
			Name:   player_name,
		},
	})

	log.Printf("Game %s created by %s.", game_name, player_name)
	return res, nil
}

func (s *GameServer) GetGames(
	ctx context.Context,
	req *connect.Request[gamev1.GetGamesRequest],
) (*connect.Response[gamev1.GetGamesResponse], error) {

	client := GetDbClient(ctx)
	defer client.Close()

	// データ取得
	items, err := client.Game.Query().All(ctx)
	if err != nil {
		log.Printf("failed querying games: %v", err)
		return nil, err
	}
	var games []*gamev1.Game
	for _, t := range items {
		games = append(games, &gamev1.Game{
			Id:     int32(t.ID),
			Status: string(t.Status),
		})
	}

	res := connect.NewResponse(&gamev1.GetGamesResponse{
		Games: games,
	})
	return res, nil
}

func (s *GameServer) StartGame(
	ctx context.Context,
	req *connect.Request[gamev1.StartGameRequest],
) (*connect.Response[gamev1.StartGameResponse], error) {
	client := GetDbClient(ctx)
	defer client.Close()

	player_id := req.Msg.UserId
	playerIDInt, err := strconv.Atoi(player_id) // str to int
	if err != nil {
		log.Printf("invalid player_id: %v", err)
		return nil, err
	}

	player, err := client.Player.Query().Where(player.IDEQ(playerIDInt)).Only((ctx))
	if err != nil {
		log.Printf("Failed to query player: %v", err)
		return nil, err
	}
	log.Printf("player is %v", player)

	game, err := player.QueryParent().Only((ctx))
	if err != nil {
		log.Printf("Failed to query parent: %v", err)
		return nil, err
	}

	_, err = client.Player.UpdateOne(player).SetStatus("STARTING").Save(ctx)
	if err != nil {
		log.Printf("Failed to update player status: %v", err)
		return nil, err
	}

	res := connect.NewResponse(&gamev1.StartGameResponse{})

	log.Printf("Player %s of game %s is STARTING", player.Name, game.Name)

	return res, nil

}

func (s *GameServer) ReportReady(
	ctx context.Context,
	req *connect.Request[gamev1.ReportReadyRequest],
) (*connect.Response[gamev1.ReportReadyResponse], error) {
	client := GetDbClient(ctx)
	defer client.Close()

	// カード要求を受けたら要求をしたユーザの状態をREADYにする
	playerId := req.Msg.PlayerId
	playerIdInt, err := strconv.Atoi(playerId)
	if err != nil {
		log.Printf("invalid playerId: %d", playerIdInt)
	}

	status := player.StatusREADY
	_, err = client.Player.UpdateOneID(playerIdInt).SetStatus(status).Save(ctx)
	if err != nil {
		log.Printf("Failed to update playerId %d to %s", playerIdInt, status)
	}

	log.Printf("Player %s is READY", playerId)
	return connect.NewResponse(&gamev1.ReportReadyResponse{}), nil

}

func (s *GameServer) SubmitAnswer(
	ctx context.Context,
	req *connect.Request[gamev1.SubmitAnswerRequest],
) (*connect.Response[gamev1.SubmitAnswerResponse], error) {
	message := "correct!!!"

	// player_idからgame_idを特定し、そのゲームのクライアントにbroadcast
	playerIDStr := req.Msg.PlayerId
	playerID, err := strconv.Atoi(playerIDStr)
	if err != nil {
		log.Printf("invalid playerID: %v", err)
	} else {
		client := GetDbClient(ctx)
		defer client.Close()
		playerEnt, err := client.Player.Query().Where(player.IDEQ(playerID)).Only(ctx)
		if err != nil {
			log.Printf("failed to query player: %v", err)
		} else {
			gameEnt, err := playerEnt.QueryParent().Only(ctx)
			if err != nil {
				log.Printf("failed to query parent game: %v", err)
			} else {
				broadcastToGame(gameEnt.ID, []byte(message))
			}
		}
	}

	// 戻り値を定義
	res := connect.NewResponse(&gamev1.SubmitAnswerResponse{
		IsCorrect: message,
	})

	return res, nil
}

/* websocket */
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return r.Header.Get("Origin") == "http://localhost:5173"
	},
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	// 初回通信でgame_id, player_idをJSONで受信
	type InitMsg struct {
		GameID   int `json:"game_id"`
		PlayerID int `json:"player_id"`
	}
	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Println("Read error:", err)
		return
	}
	log.Printf("Received: %s", message)
	var initMsg InitMsg
	if err := json.Unmarshal(message, &initMsg); err != nil {
		log.Println("Invalid init message:", err)
		return
	}

	// gameClientsに登録
	if gameClients[initMsg.GameID] == nil {
		gameClients[initMsg.GameID] = make(map[*websocket.Conn]ClientInfo)
	}
	gameClients[initMsg.GameID][conn] = ClientInfo{
		GameID:   initMsg.GameID,
		PlayerID: initMsg.PlayerID,
	}
	defer func() {
		delete(gameClients[initMsg.GameID], conn)
		// 切断通知を同じゲームのクライアントにbroadcast
		disconnectMsg := map[string]interface{}{
			"event":     "disconnect",
			"game_id":   initMsg.GameID,
			"player_id": initMsg.PlayerID,
		}
		b, _ := json.Marshal(disconnectMsg)
		broadcastToGame(initMsg.GameID, b)
	}()

	err = conn.WriteMessage(websocket.TextMessage, []byte("Hello from service!!"))
	if err != nil {
		log.Println("Write error:", err)
	}
	log.Print("Write message")

	// 常時通信
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}
		log.Printf("Received: %s", message)
	}
}

func broadcastToGame(gameID int, message []byte) {
	conns, ok := gameClients[gameID]
	if !ok {
		return
	}
	for conn := range conns {
		err := conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			conn.Close()
			delete(conns, conn)
		}
	}
}

/* CORS */
func withCORS(h http.Handler) http.Handler {
	return cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "X-User-Agent", "Connect-Protocol-Version"},
	}).Handler(h)
}

type Card struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}

var unsentCards = make(map[int][]Card)

type GameStatus struct {
	AllStarting bool
	AllReady    bool
	PlayerCount int
}

/* 100ms事にデータベースの状態を管理してwsbroadcastするgoroutine */
func monitorGames() {
	log.Printf("monitorGames started")
	lastStatusMap := make(map[int]GameStatus)
	for {
		client := GetDbClient(context.Background())
		games, err := client.Game.Query().All(context.Background())
		if err != nil {
			log.Printf("monitorGames: failed to query games: %v", err)
			client.Close()
			time.Sleep(100 * time.Millisecond)
			continue
		}
		log.Printf("%d games found.", len(games))
		for _, game := range games {
			// log.Printf("Game is %v", game)

			players, err := client.Player.Query().Where(player.HasParentWith(g.IDEQ(game.ID))).All(context.Background())
			if err != nil {
				log.Printf("monitorGames: failed to query players: %v", err)
				continue
			}
			playerCount := len(players)
			allStarting := true
			allReady := true
			log.Printf("%d players found.", playerCount)
			for _, p := range players {
				if p.Status != "STARTING" {
					allStarting = false
				}
				if p.Status != "READY" {
					allReady = false
				}
			}
			prev, ok := lastStatusMap[game.ID]
			// 全員STARTINGになったタイミング
			if allStarting && (!ok || !prev.AllStarting) && playerCount > 0 {
				msg := map[string]interface{}{
					"event":   "all_starting",
					"game_id": game.ID,
				}
				b, _ := json.Marshal(msg)
				broadcastToGame(game.ID, b)
				log.Printf("message %v broadcasted", err)
			}
			// 全員READYになったタイミング
			if allReady && (!ok || !prev.AllReady) && playerCount > 0 {
				msg := map[string]interface{}{
					"event":   "all_ready",
					"game_id": game.ID,
				}
				b, _ := json.Marshal(msg)
				broadcastToGame(game.ID, b)
				log.Printf("message %v broadcasted", err)

				cards := unsentCards[game.ID]
				if len(cards) > 0 {
					card := cards[0]
					unsentCards[game.ID] = cards[1:]
					cardMsg := map[string]interface{}{
						"event":   "card",
						"game_id": game.ID,
						"card":    card,
					}
					cb, _ := json.Marshal(cardMsg)
					broadcastToGame(game.ID, cb)
					log.Printf("message %v broadcasted", err)
				}
			}
			lastStatusMap[game.ID] = GameStatus{
				AllStarting: allStarting,
				AllReady:    allReady,
				PlayerCount: playerCount,
			}
		}
		client.Close()
		time.Sleep(1000 * time.Millisecond)
	}
}

/* main */
func main() {
	// サーバー起動時に一度だけスキーマ作成
	client, err := ent.Open(dialect.SQLite, DB_FILE)
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}
	if err := client.Schema.Create(context.Background()); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			log.Printf("warning: schema resource already exists: %v", err)
		} else {
			log.Fatalf("failed creating schema resources: %v", err)
		}
	}
	client.Close()

	// 100msごとの監視goroutine起動
	go monitorGames()

	// マルチプレクサ(ルータ)を生成
	mux := http.NewServeMux()

	// サービスを生成
	// サービスハンドラにgreeterサービス登録、
	// ルーティング用のpathと関数呼び出し用のハンドラを作成
	game := &GameServer{}

	// マルチプレクサ(ルータ)にパスとハンドラを追加
	mux.Handle(gamev1connect.NewJoinGameServiceHandler(game))
	mux.Handle(gamev1connect.NewCreateGameServiceHandler(game))
	mux.Handle(gamev1connect.NewGetGamesServiceHandler(game))
	mux.Handle(gamev1connect.NewSubmitAnswerServiceHandler(game))
	mux.Handle(gamev1connect.NewStartGameServiceHandler(game))
	mux.Handle(gamev1connect.NewReportReadyServiceHandler(game))

	// WebSocketハンドラの登録
	mux.HandleFunc("/ws", websocketHandler)

	// httpサーバーを起動
	http.ListenAndServe(
		"0.0.0.0:8080",
		// Use h2c so we can serve HTTP/2 without TLS.
		withCORS(h2c.NewHandler(mux, &http2.Server{})),
	)
}
