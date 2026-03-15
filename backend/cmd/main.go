package main

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

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

	// CREATED状態のゲームのみ参加可能
	gameEnt, err := client.Game.Get(ctx, gameIDInt)
	if err != nil {
		log.Printf("game not found: %v", err)
		return nil, err
	}
	if gameEnt.Status != g.StatusCREATED {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("開始済みのゲームには参加できません"))
	}

	newPlayer, err := client.Player.Create().SetName(player_name).SetParentID(gameIDInt).Save(ctx)
	if err != nil {
		log.Printf("failed creating player: %v", err)
		return nil, err
	}
	log.Printf("player is %v", newPlayer)

	res := connect.NewResponse(&gamev1.JoinGameResponse{
		Player: &gamev1.Player{
			GameId: int32(gameIDInt),
			Id:     int32(newPlayer.ID),
			Name:   player_name,
			Score:  int32(newPlayer.Score),
		},
	})

	totalCards := len(unsentCards[gameIDInt])
	totalRounds := totalCards - 1
	if totalRounds < 0 {
		totalRounds = 0
	}
	// ゲームの全プレイヤー一覧を取得
	allPlayers, _ := client.Player.Query().
		Where(player.HasParentWith(g.IDEQ(gameIDInt))).
		All(ctx)
	var playerList []map[string]interface{}
	for _, p := range allPlayers {
		playerList = append(playerList, map[string]interface{}{
			"player_id": p.ID,
			"name":      p.Name,
			"score":     p.Score,
		})
	}

	msg := map[string]interface{}{
		"event":        "JOINED",
		"total_rounds": totalRounds,
		"players":      playerList,
	}
	b, _ := json.Marshal(msg)
	broadcastToAll(b)
	lobbyMsg := map[string]interface{}{
		"event": "JOINED",
	}
	lb, _ := json.Marshal(lobbyMsg)
	broadcastToLobby(lb)

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

	// 同名のCREATED/STARTEDゲームが存在しないかチェック
	game_name := req.Msg.GameName
	exists, err := client.Game.Query().
		Where(
			g.NameEQ(game_name),
			g.StatusIn(g.StatusCREATED, g.StatusSTARTED),
		).Exist(ctx)
	if err != nil {
		log.Printf("failed checking game name: %v", err)
		return nil, err
	}
	if exists {
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("同名のゲームが既に存在します: %s", game_name))
	}

	// Dobbleカード生成
	generatedCards, _, err := cardgen.GenerateDobbleCards(5)
	if err != nil {
		log.Fatalf("failed to generate cards: %v", err)
	}

	rand.Shuffle(len(generatedCards), func(i, j int) {
		generatedCards[i], generatedCards[j] = generatedCards[j], generatedCards[i]
	})

	// カード枚数制限
	cardCount := int(req.Msg.CardCount)
	log.Printf("requested card_count: %d, generated: %d", cardCount, len(generatedCards))
	if cardCount > 0 && cardCount <= len(generatedCards) {
		generatedCards = generatedCards[:cardCount]
	}
	totalRounds := len(generatedCards) - 1
	if totalRounds < 0 {
		totalRounds = 0
	}
	log.Printf("final card count: %d, total rounds: %d", len(generatedCards), totalRounds)

	// レコード追加
	game, err := client.Game.Create().SetName(game_name).SetTotalRounds(totalRounds).Save(ctx)
	if err != nil {
		log.Printf("failed creating game: %v", err)
		return nil, err
	}

	var cs []Card
	for _, c := range generatedCards {
		cs = append(cs, Card{
			ID:   c.ID,
			Text: "symbols: " + fmt.Sprint(c.Symbols),
		})
	}
	log.Printf("%d cards created", len(cs))
	unsentCards[game.ID] = cs

	res := connect.NewResponse(&gamev1.CreateGameResponse{
		GameId: int32(game.ID),
	})

	msg := map[string]interface{}{
		"event": "CREATED",
	}
	b, _ := json.Marshal(msg)
	broadcastToAll(b)
	broadcastToLobby(b)

	log.Printf("Game %s id of %d created.", game_name, game.ID)
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

	// データ取得（最新10件、プレイヤー数も含めて）
	items, err := client.Game.Query().WithPlayers().Order(game.ByID(sql.OrderDesc())).Limit(10).All(ctx)
	if err != nil {
		log.Printf("failed querying games: %v", err)
		return nil, err
	}
	var games []*gamev1.Game
	for _, t := range items {
		games = append(games, &gamev1.Game{
			Id:          int32(t.ID),
			Status:      string(t.Status),
			Name:        t.Name,
			PlayerCount: int32(len(t.Edges.Players)),
			TotalRounds: int32(t.TotalRounds),
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

	gameId := req.Msg.GameId
	gamaIdInt, err := strconv.Atoi(gameId)
	if err != nil {
		log.Printf("Failed to conv gameId %s", gameId)
	}

	// ゲームのステータスをSTARTEDに更新
	_, err = client.Game.UpdateOneID(gamaIdInt).SetStatus("STARTED").Save(ctx)
	if err != nil {
		log.Printf("Failed to update game status: %v", err)
		return nil, err
	}

	// 全プレイヤーのステータスをSTARTEDに更新
	_, err = client.Player.Update().
		Where(player.HasParentWith(g.IDEQ(gamaIdInt))).
		SetStatus("STARTED").
		Save(ctx)
	if err != nil {
		log.Printf("Failed to update players status: %v", err)
	}

	totalCards := len(unsentCards[gamaIdInt])
	totalRounds := totalCards - 1
	if totalRounds < 0 {
		totalRounds = 0
	}

	// プレイヤー一覧を取得
	players, err := client.Player.Query().
		Where(player.HasParentWith(g.IDEQ(gamaIdInt))).
		All(ctx)
	if err != nil {
		log.Printf("Failed to query players: %v", err)
	}
	var playerList []map[string]interface{}
	for _, p := range players {
		playerList = append(playerList, map[string]interface{}{
			"player_id": p.ID,
			"name":      p.Name,
			"score":     p.Score,
		})
	}

	msg := map[string]interface{}{
		"event":        "STARTED",
		"game_id":      gamaIdInt,
		"total_rounds": totalRounds,
		"players":      playerList,
	}
	b, _ := json.Marshal(msg)
	broadcastToGame(gamaIdInt, b)

	// ロビーに通知
	lobbyMsg := map[string]interface{}{
		"event": "STARTED",
	}
	lb, _ := json.Marshal(lobbyMsg)
	broadcastToLobby(lb)

	res := connect.NewResponse(&gamev1.StartGameResponse{})

	log.Printf("Game %s is STARTED", gameId)

	return res, nil

}

func DistributeCard(client *ent.Client, gameId int) {
	cards := unsentCards[gameId]
	log.Printf("%d cards remaining with game id %d", len(cards), gameId)
	if len(cards) == 0 {
		// カードが無くなったらゲーム終了イベントを送信
		endMsg := map[string]interface{}{
			"event": "GAME_OVER",
		}
		b, _ := json.Marshal(endMsg)
		broadcastToGame(gameId, b)
		return
	}

	if len(cards) > 0 {
		// カード送信前にPLAYINGに更新し、次のREADY要求に備える
		_, err := client.Player.Update().
			Where(player.HasParentWith(g.IDEQ(gameId))).
			SetStatus("PLAYING").
			Save(context.Background())
		if err != nil {
			log.Fatalf("failed updating player status")
		}

		card := cards[0]
		unsentCards[gameId] = cards[1:]
		cardMsg := map[string]interface{}{
			"event":   "card",
			"game_id": gameId,
			"card":    card,
		}
		cb, _ := json.Marshal(cardMsg)
		broadcastToGame(gameId, cb)
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
		return nil, err
	}

	parentIDs, err := client.Player.Query().
		Where(player.IDEQ(playerIdInt)).
		Select("player_parent").
		Ints(ctx)
	if err != nil {
		log.Printf("failed to query parent: %v", err)
		return nil, err
	}

	var gameId = 0
	if len(parentIDs) > 0 {
		gameId = parentIDs[0]
	}

	// ゲームごとのミューテックスでレースコンディションを防止
	mu := getGameMutex(gameId)
	mu.Lock()
	defer mu.Unlock()

	// 現在のプレイヤーステータスを確認（既にREADYなら重複処理しない）
	currentPlayer, err := client.Player.Get(ctx, playerIdInt)
	if err != nil {
		log.Printf("failed to get player: %v", err)
		return nil, err
	}
	if currentPlayer.Status == player.StatusREADY {
		log.Printf("Player %s is already READY, skipping", playerId)
		return connect.NewResponse(&gamev1.ReportReadyResponse{}), nil
	}

	_, err = client.Player.UpdateOneID(playerIdInt).SetStatus(player.StatusREADY).Save(ctx)
	if err != nil {
		log.Printf("Failed to update playerId %d to READY", playerIdInt)
		return nil, err
	}

	notReadyCount, err := client.Player.Query().
		Where(
			player.HasParentWith(g.IDEQ(gameId)),
			player.StatusNEQ(player.StatusREADY),
		).Count(ctx)
	if err != nil {
		log.Printf("failed to count not-ready players: %v", err)
		return nil, err
	}

	if notReadyCount == 0 {
		DistributeCard(client, gameId)
	}

	log.Printf("Player %s is READY (notReadyCount=%d)", playerId, notReadyCount)
	return connect.NewResponse(&gamev1.ReportReadyResponse{}), nil
}

func extractSymbols(text string) []string {
	start := strings.Index(text, "[")
	end := strings.Index(text, "]")
	if start == -1 || end == -1 || start >= end {
		return []string{}
	}
	symbolsStr := text[start+1 : end]
	return strings.Fields(symbolsStr)
}

func findCommonSymbol(card1, card2 Card) string {
	symbols1 := extractSymbols(card1.Text)
	symbols2 := extractSymbols(card2.Text)

	symbolSet := make(map[string]struct{})
	for _, s := range symbols1 {
		symbolSet[s] = struct{}{}
	}

	for _, s := range symbols2 {
		if _, exists := symbolSet[s]; exists {
			return s
		}
	}
	return ""
}

func isAnswerCorrect(card1, card2 Card, answer string) bool {
	log.Printf("card1 %v", card1)
	log.Printf("card2 %v", card2)
	log.Printf("answer %v", answer)

	commonSymbol := findCommonSymbol(card1, card2)
	if commonSymbol == answer {
		log.Printf("answer is correct")
		return true
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
		// スコア加減算処理を追加
		if isCorrect {
			_, err := client.Player.UpdateOneID(playerID).AddScore(1).Save(ctx)
			if err != nil {
				log.Printf("failed to add score: %v", err)
			}
		} else {
			_, err := client.Player.UpdateOneID(playerID).AddScore(-1).Save(ctx)
			if err != nil {
				log.Printf("failed to subtract score: %v", err)
			}
		}
		playerEnt, err := client.Player.Query().Where(player.IDEQ(playerID)).Only(ctx)
		if err != nil {
			log.Printf("failed to query player: %v", err)
		} else {
			gameEnt, err := playerEnt.QueryParent().Only(ctx)
			if err != nil {
				log.Printf("failed to query parent game: %v", err)
			} else {
				// 参加者全員のスコアを取得
				players, err := gameEnt.QueryPlayers().All(ctx)
				scores := []map[string]interface{}{}
				if err == nil {
					for _, p := range players {
						scores = append(scores, map[string]interface{}{
							"player_id": p.ID,
							"score":     p.Score,
							"name":      p.Name,
						})
					}
				}
				// card1.Text, card2.Text から ent.Card を検索し、DBのIDを取得
				correctSymbol := findCommonSymbol(card1, card2)
				msg := map[string]interface{}{
					"event":          "ANSWERED",
					"player_id":      playerID,
					"is_correct":     isCorrect,
					"correct_symbol": correctSymbol,
					"answer":         answer,
					"scores":         scores,
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

func (s *GameServer) DeleteGame(
	ctx context.Context,
	req *connect.Request[gamev1.DeleteGameRequest],
) (*connect.Response[gamev1.DeleteGameResponse], error) {
	endLog := funcCallLog(logFuncName())
	defer endLog()

	client := GetDbClient(ctx)
	defer client.Close()

	gameId := req.Msg.GameId
	gameIdInt, err := strconv.Atoi(gameId)
	if err != nil {
		log.Printf("invalid game_id: %v", err)
		return nil, err
	}

	// CREATED状態のゲームのみ削除可能
	gameEnt, err := client.Game.Get(ctx, gameIdInt)
	if err != nil {
		log.Printf("game not found: %v", err)
		return nil, err
	}
	if gameEnt.Status != g.StatusCREATED {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("開始済みのゲームは削除できません"))
	}

	// 関連プレイヤーを削除
	_, err = client.Player.Delete().Where(player.HasParentWith(g.IDEQ(gameIdInt))).Exec(ctx)
	if err != nil {
		log.Printf("failed deleting players: %v", err)
	}

	// ゲームを削除
	err = client.Game.DeleteOneID(gameIdInt).Exec(ctx)
	if err != nil {
		log.Printf("failed deleting game: %v", err)
		return nil, err
	}

	// カード情報も削除
	delete(unsentCards, gameIdInt)

	// ロビーに通知
	msg := map[string]interface{}{
		"event": "DELETED",
	}
	b, _ := json.Marshal(msg)
	broadcastToLobby(b)

	log.Printf("Game %d deleted.", gameIdInt)
	return connect.NewResponse(&gamev1.DeleteGameResponse{}), nil
}

/* websocket */
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "http://localhost:5173" || origin == "https://gobble-frontend.onrender.com"
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

	// クライアント接続直後にプレイヤー一覧を全員に送信
	{
		ctx := context.Background()
		client := GetDbClient(ctx)
		gamePlayers, _ := client.Player.Query().
			Where(player.HasParentWith(g.IDEQ(initMsg.GameID))).
			All(ctx)
		client.Close()
		var pList []map[string]interface{}
		for _, p := range gamePlayers {
			pList = append(pList, map[string]interface{}{
				"player_id": p.ID,
				"name":      p.Name,
				"score":     p.Score,
			})
		}
		playersEvent := map[string]interface{}{
			"event":   "PLAYERS",
			"players": pList,
		}
		pJSON, _ := json.Marshal(playersEvent)
		broadcastToGame(initMsg.GameID, pJSON)
	}

	log.Printf("gameClients %v", gameClients)

	defer func() {
		log.Printf("websocket disconnected execute defer func")
		defer conn.Close()
		delete(gameClients[initMsg.GameID], conn)

		ctx := context.Background()
		client := GetDbClient(ctx)
		defer client.Close()

		// ゲームの状態を確認
		gameEnt, err := client.Game.Get(ctx, initMsg.GameID)
		if err != nil {
			log.Printf("game %d not found: %v", initMsg.GameID, err)
			endLog()
			return
		}

		if gameEnt.Status == g.StatusSTARTED {
			// 実施中のゲーム: FINISHEDにして他プレイヤーに切断通知
			_, err = client.Game.UpdateOneID(initMsg.GameID).SetStatus("FINISHED").Save(ctx)
			if err != nil {
				log.Printf("failed updating game %d to FINISHED: %v", initMsg.GameID, err)
			}
			delete(unsentCards, initMsg.GameID)

			disconnectMsg := map[string]interface{}{
				"event":     "disconnect",
				"game_id":   initMsg.GameID,
				"player_id": initMsg.PlayerID,
			}
			db, _ := json.Marshal(disconnectMsg)
			broadcastToGame(initMsg.GameID, db)

			log.Printf("Game %d finished due to disconnect.", initMsg.GameID)
		} else {
			// 未開始のゲーム: プレイヤーとゲームを削除
			_, err = client.Player.Delete().
				Where(player.HasParentWith(g.IDEQ(initMsg.GameID))).
				Exec(ctx)
			if err != nil {
				log.Printf("failed deleting players for game %d: %v", initMsg.GameID, err)
			}
			err = client.Game.DeleteOneID(initMsg.GameID).Exec(ctx)
			if err != nil {
				log.Printf("failed deleting game %d: %v", initMsg.GameID, err)
			}
			delete(unsentCards, initMsg.GameID)
			log.Printf("Game %d deleted due to disconnect.", initMsg.GameID)
		}

		// ロビーに通知
		lobbyMsg := map[string]interface{}{
			"event": "DELETED",
		}
		lb, _ := json.Marshal(lobbyMsg)
		broadcastToLobby(lb)
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

/* lobby websocket */
var lobbyClients = make(map[*websocket.Conn]bool)

func lobbyWebsocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Lobby WS upgrade error:", err)
		return
	}
	lobbyClients[conn] = true
	log.Printf("Lobby client connected. Total: %d", len(lobbyClients))

	defer func() {
		delete(lobbyClients, conn)
		conn.Close()
		log.Printf("Lobby client disconnected. Total: %d", len(lobbyClients))
	}()

	// 接続維持（クライアントからの切断を検知）
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func broadcastToLobby(message []byte) {
	for conn := range lobbyClients {
		err := conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("lobby broadcast error: %v", err)
			conn.Close()
			delete(lobbyClients, conn)
		}
	}
}

/* CORS */
func withCORS(h http.Handler) http.Handler {
	return cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173", "https://gobble-frontend.onrender.com"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "X-User-Agent", "Connect-Protocol-Version"},
	}).Handler(h)
}

type Card struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}

var unsentCards = make(map[int][]Card)

// ゲームごとのミューテックス（ReportReady/DistributeCardのレースコンディション防止）
var gameMutexes = make(map[int]*sync.Mutex)
var gameMutexLock sync.Mutex

func getGameMutex(gameId int) *sync.Mutex {
	gameMutexLock.Lock()
	defer gameMutexLock.Unlock()
	if gameMutexes[gameId] == nil {
		gameMutexes[gameId] = &sync.Mutex{}
	}
	return gameMutexes[gameId]
}

/* main */
func main() {
	log.Printf("Starting server on 0.0.0.0:8080")

	// .dbディレクトリがなければ作成する
	dbDir := filepath.Dir("backend/.db/ent.db")
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dbDir, os.ModePerm); err != nil {
			log.Fatalf("failed to create db directory: %v", err)
		}
	}

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
	mux.Handle(gamev1connect.NewDeleteGameServiceHandler(game))

	// WebSocketハンドラの登録
	mux.HandleFunc("/ws", websocketHandler)
	mux.HandleFunc("/ws/lobby", lobbyWebsocketHandler)

	// httpサーバーを起動
	http.ListenAndServe(
		"0.0.0.0:8080",
		// Use h2c so we can serve HTTP/2 without TLS.
		withCORS(h2c.NewHandler(mux, &http2.Server{})),
	)
}
