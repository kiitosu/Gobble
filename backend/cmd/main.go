package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"example/ent/player"
	gamev1 "example/gen/game/v1"
	"example/gen/game/v1/gamev1connect"

	"github.com/gorilla/websocket"

	"example/ent"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
)

var clients = make(map[*websocket.Conn]bool)

const DB_FILE = "file:backend/.db/ent.db?_fk=1"

func GetDbClient(
	ctx context.Context,
) *ent.Client {
	// DBにゲーム情報を書き込み
	// ファイルベースのSQLiteデータベースを持つent.Clientを作成します。
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
	return connect.NewResponse(&gamev1.ReportReadyResponse{}), nil

}

func (s *GameServer) SubmitAnswer(
	ctx context.Context,
	req *connect.Request[gamev1.SubmitAnswerRequest],
) (*connect.Response[gamev1.SubmitAnswerResponse], error) {
	// 受け取った回答が正しいか確認する
	// TODO : 実装する

	// 返却値
	message := "correct!!!"

	// websocketHandlerに通知、websocketでブロードキャスト送信する
	broadcast([]byte(message))

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
	clients[conn] = true
	defer delete(clients, conn)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}
		log.Printf("Received: %s", message)

		err = conn.WriteMessage(websocket.TextMessage, []byte("Hello from service!!"))
		if err != nil {
			log.Println("Write error:", err)
			break
		}
		log.Print("Write message")
	}
}

func broadcast(message []byte) {
	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			client.Close()
			delete(clients, client)
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
