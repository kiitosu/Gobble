package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"example/ent/game"
	g "example/ent/game"
	"example/ent/player"
	gamev1 "example/gen/game/v1"
	"example/gen/game/v1/gamev1connect"
	"example/internal/cardgen"

	"github.com/gorilla/websocket"

	"example/ent"

	"encoding/json"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
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

func logFuncName() string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return "unknown"
	}
	return runtime.FuncForPC(pc).Name()
}

func funcCallLog(funcName string) func() {
	log.Printf("________")
	log.Printf("START: %s", funcName)

	endLog := func() {
		log.Printf("END: %s", funcName)
		log.Printf("^^^^^^^^ ")
	}
	return endLog
}

func (s *GameServer) JoinGame(
	ctx context.Context,
	req *connect.Request[gamev1.JoinGameRequest],
) (*connect.Response[gamev1.JoinGameResponse], error) {
	endLog := funcCallLog(logFuncName())
	defer endLog()

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

	msg := map[string]interface{}{
		"event": "JOINED",
	}
	b, _ := json.Marshal(msg)
	broadcastToAll(b)

	log.Printf("User %s join the game %s", player_name, game_id)
	return res, nil
}

func (s *GameServer) CreateGame(
	ctx context.Context,
	req *connect.Request[gamev1.CreateGameRequest],
) (*connect.Response[gamev1.CreateGameResponse], error) {
	endLog := funcCallLog(logFuncName())
	defer endLog()

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
		log.Printf("%d cards created", len(cs))
		unsentCards[game.ID] = cs
	}

	res := connect.NewResponse(&gamev1.CreateGameResponse{
		Player: &gamev1.Player{
			GameId: int32(game.ID),
			Id:     int32(player.ID),
			Name:   player_name,
		},
	})

	msg := map[string]interface{}{
		"event": "CREATED",
	}
	b, _ := json.Marshal(msg)
	broadcastToAll(b)

	log.Printf("Game %s id of %d created by %s.", game_name, game.ID, player_name)
	return res, nil
}

func (s *GameServer) GetGames(
	ctx context.Context,
	req *connect.Request[gamev1.GetGamesRequest],
) (*connect.Response[gamev1.GetGamesResponse], error) {
	endLog := funcCallLog(logFuncName())
	defer endLog()

	client := GetDbClient(ctx)
	defer client.Close()

	// データ取得
	items, err := client.Game.Query().Order(game.ByID(sql.OrderDesc())).All(ctx)
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
	endLog := funcCallLog(logFuncName())
	defer endLog()

	client := GetDbClient(ctx)
	defer client.Close()

	player_id := req.Msg.UserId
	playerIDInt, err := strconv.Atoi(player_id) // str to int
	if err != nil {
		log.Printf("invalid player_id: %v", err)
		return nil, err
	}

	_, err = client.Player.UpdateOneID(playerIDInt).SetStatus("STARTED").Save(ctx)
	if err != nil {
		log.Printf("Failed to update player status: %v", err)
		return nil, err
	}
	gameId := req.Msg.GameId
	gamaIdInt, err := strconv.Atoi(gameId)
	if err != nil {
		log.Printf("Failed to conv gameId %s", gameId)
	}

	msg := map[string]interface{}{
		"event":   "STARTED",
		"game_id": gamaIdInt,
	}
	b, _ := json.Marshal(msg)
	broadcastToGame(gamaIdInt, b)

	res := connect.NewResponse(&gamev1.StartGameResponse{})

	log.Printf("Player %s of game %s is STARTED", player_id, gameId)

	return res, nil

}

func DistributeCard(client *ent.Client, gameId int) {
	cards := unsentCards[gameId]
	log.Printf("%d cards remaining with game id %d", len(cards), gameId)
	if len(cards) > 0 {
		card := cards[0]
		unsentCards[gameId] = cards[1:]
		cardMsg := map[string]interface{}{
			"event":   "card",
			"game_id": gameId,
			"card":    card,
		}
		cb, _ := json.Marshal(cardMsg)
		broadcastToGame(gameId, cb)

		// カード送信後はPLAYINGとし、次のREADYを待ち受けられるようにする
		_, err := client.Player.Update().
			Where(player.HasParentWith(g.IDEQ(gameId))).
			SetStatus("PLAYING").
			Save(context.Background())
		if err != nil {
			log.Fatalf("failed updating player status")
		}
	}
}

func (s *GameServer) ReportReady(
	ctx context.Context,
	req *connect.Request[gamev1.ReportReadyRequest],
) (*connect.Response[gamev1.ReportReadyResponse], error) {
	endLog := funcCallLog(logFuncName())
	defer endLog()

	client := GetDbClient(ctx)
	defer client.Close()

	// カード要求を受けたら要求をしたユーザの状態をREADYにする
	playerId := req.Msg.PlayerId
	playerIdInt, err := strconv.Atoi(playerId)
	if err != nil {
		log.Printf("invalid playerId: %d", playerIdInt)
	}

	parentIDs, err := client.Player.Query().
		Where(player.IDEQ(playerIdInt)).
		Select("player_parent").
		Ints(ctx)
	if err != nil {
		// error handling
	}

	var gameId = 0
	if len(parentIDs) > 0 {
		gameId = parentIDs[0]
		// gameIDが該当プレイヤーのgame_id
	}

	status := player.StatusREADY
	_, err = client.Player.UpdateOneID(playerIdInt).SetStatus(status).Save(ctx)
	if err != nil {
		log.Printf("Failed to update playerId %d to %s", playerIdInt, status)
	}

	notReadyCount, err := client.Player.Query().
		Where(
			player.HasParentWith(g.IDEQ(gameId)),
			player.StatusNEQ("READY"),
		).Count(context.Background())
	if err != nil {
		// error handling
	}
	allReady := notReadyCount == 0

	if allReady {
		DistributeCard(client, gameId)
	}

	log.Printf("Player %s is READY", playerId)
	return connect.NewResponse(&gamev1.ReportReadyResponse{}), nil
}

func isAnswerCorrect(card1, card2 Card, answer string) bool {
	log.Printf("card1 %v", card1)
	log.Printf("card2 %v", card2)
	log.Printf("answer %v", answer)

	// カードのテキストからシンボルを抽出する関数
	extractSymbols := func(text string) []string {
		// "symbols: [a b c]" のような形式を想定
		start := strings.Index(text, "[")
		end := strings.Index(text, "]")
		if start == -1 || end == -1 || start >= end {
			return []string{}
		}
		symbolsStr := text[start+1 : end]
		return strings.Fields(symbolsStr)
	}

	symbols1 := extractSymbols(card1.Text)
	symbols2 := extractSymbols(card2.Text)

	// 共通するシンボルを探す
	symbolSet := make(map[string]struct{})
	for _, s := range symbols1 {
		symbolSet[s] = struct{}{}
	}

	for _, s := range symbols2 {
		if _, exists := symbolSet[s]; exists && s == answer {
			log.Printf("answer is correct")
			return true
		}
	}

	log.Printf("answer is wrong")
	return false
}

func (s *GameServer) SubmitAnswer(
	ctx context.Context,
	req *connect.Request[gamev1.SubmitAnswerRequest],
) (*connect.Response[gamev1.SubmitAnswerResponse], error) {
	endLog := funcCallLog(logFuncName())
	defer endLog()

	message := "correct!!!"
	card1 := Card{
		ID:   int(req.Msg.Card1.Id),
		Text: req.Msg.Card1.Text,
	}
	card2 := Card{
		ID:   int(req.Msg.Card2.Id),
		Text: req.Msg.Card2.Text,
	}
	answer := req.Msg.Answer
	isCorrect := isAnswerCorrect(card1, card2, answer)
	if !isCorrect {
		message = "wrong!!!"
	}

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
				msg := map[string]interface{}{
					"event":      "ANSWERED",
					"player_id":  playerID,
					"is_correct": isCorrect,
				}
				b, _ := json.Marshal(msg)
				broadcastToGame(gameEnt.ID, b)
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
	endLog := funcCallLog(logFuncName())

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

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
	log.Printf("initMsg %v", initMsg)

	// gameClientsに登録
	if gameClients[initMsg.GameID] == nil {
		gameClients[initMsg.GameID] = make(map[*websocket.Conn]ClientInfo)
	}
	gameClients[initMsg.GameID][conn] = ClientInfo{
		GameID:   initMsg.GameID,
		PlayerID: initMsg.PlayerID,
	}

	// クライアント接続直後にDEBUGイベントを送信（デバッグ用途）
	debugEvent := map[string]interface{}{
		"event": "DEBUG",
		"state": "connected",
	}
	debugEventJSON, _ := json.Marshal(debugEvent)
	broadcastToGame(initMsg.GameID, debugEventJSON)

	log.Printf("gameClients %v", gameClients)

	defer func() {
		log.Printf("websocket disconnected execute defer func")
		defer conn.Close()
		delete(gameClients[initMsg.GameID], conn)
		// 切断通知を同じゲームのクライアントにbroadcast
		disconnectMsg := map[string]interface{}{
			"event":     "disconnect",
			"game_id":   initMsg.GameID,
			"player_id": initMsg.PlayerID,
		}
		b, _ := json.Marshal(disconnectMsg)
		broadcastToGame(initMsg.GameID, b)

		client := GetDbClient(context.Background())
		defer client.Close()
		_, err := client.Player.Update().
			Where(player.HasParentWith(g.IDEQ(initMsg.GameID))).
			SetStatus("FINISHED").
			Save(context.Background())
		if err != nil {
			log.Fatalf("failed updating player status")
		}
		_, err = client.Game.UpdateOneID(initMsg.GameID).SetStatus("FINISHED").Save(context.Background())
		if err != nil {
			log.Fatalf("failed updating player status")
		}
		endLog()
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
		log.Printf("ws:Received: %s", message)
	}
}

func broadcastToGame(gameID int, message []byte) {
	log.Printf("broadcast to game")
	conns, ok := gameClients[gameID]
	if !ok {
		log.Printf("gameClients err: %v", ok)
		return
	}
	for conn := range conns {
		err := conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("broadcast %s error: %v", message, err)
			conn.Close()
			delete(conns, conn)
		} else {
			log.Printf("broadcast %s success: %+v", message, conns[conn])
		}
	}
}

func broadcastToAll(message []byte) {
	log.Printf("broadcast to all %v", gameClients)
	for _, conns := range gameClients {
		for conn := range conns {
			err := conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("broadcast error: %v", err)
				conn.Close()
				delete(conns, conn)
			} else {
				log.Printf("broadcast success: %s", message)
			}
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
