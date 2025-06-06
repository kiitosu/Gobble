import React, { useEffect, useState, useMemo } from "react";
import { WebSocketProvider } from "./WebSocketContext";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  JoinGameService,
  CreateGameService,
  GetGamesService,
  StartGameService,
} from "../gen/game/v1/game_pb";
import type { Game, Player } from "../gen/game/v1/game_pb";
import GameComponent from "./Game";


type LobbyProps = {};

const Lobby: React.FC<LobbyProps> = ({}) => {
  const [ws, setWs] = useState<WebSocket | null>(null);
  const [gameStatus, setGameStatus] = useState("");
  const [player, setPlayer] = useState<Player>();
  const [games, setGames] = useState<Game[]>([]);
  const [gameName, setGameName] = useState("");

    const transport = createConnectTransport({
    baseUrl: "http://localhost:8080",
  });
  const createGameClient = createClient(CreateGameService, transport);
  const joinGameServiceclient = createClient(JoinGameService, transport);
  const startGameServiceclient = createClient(StartGameService, transport);
  const getGamesClient = useMemo(() => createClient(GetGamesService, transport), [transport]);

  // player情報が揃ったらWebSocket接続
  useEffect(() => {
    if (player && player.gameId && player.id && !ws) {
      const socket = new WebSocket("ws://localhost:8080/ws");
      socket.onopen = () => {
        socket.send(
          JSON.stringify({ game_id: player.gameId, player_id: player.id })
        );
      };
      setWs(socket);
    }
    // クリーンアップ: Lobbyアンマウント時にWebSocket切断
    return () => {
      if (ws) {
        ws.close();
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [player]);

  // コンポーネント起動時にゲーム状態を取得
  useEffect(() => {
    const fetchGames = async () => {
      const response = await getGamesClient.getGames({});
      setGames(response.games);
    };
    fetchGames();
  }, [getGamesClient, setGames]);

  // ゲームリスト
  let gameList: React.ReactNode[] = [];
  if (gameStatus === "") {
    gameList = games.map((item) => (
      <li key={item.id}>
        Game id {item.id} is {item.status} status
        <button
          onClick={async () => {
            const response = await joinGameServiceclient.joinGame({
              gameId: String(item.id),
              playerName: "unknown",
            });
            console.log(`Join the game ${response}`);
            setGameStatus("JOINED");
            if (response.player) {
              setPlayer(response.player);
            }
          }}
        >
          参加
        </button>
      </li>
    ));
  } else if (gameStatus === "CREATED") {
    gameList = games
      .filter((item) => item.status === "CREATED")
      .map((item) => (
        <li key={item.id}>
          Game id {item.id} is {item.status}
          <button
            onClick={async () => {
              const response = await startGameServiceclient.startGame({
                gameId: String(item.id),
                userId: String(player?.id),
              });
              console.log(response);
              setGameStatus("STARTED");
            }}
          >
            開始
          </button>
        </li>
      ));
  }

  // ゲーム作成フォーム
  const createGameForm =
    gameStatus === "" ? (
      <form
        onSubmit={async (e) => {
          e.preventDefault();
          const response = await createGameClient.createGame({
            gameName: gameName,
            playerName: gameName,
          });
          if (response.player) {
            setPlayer(response.player);
          }
          console.log(
            `created game is ${response.player?.gameId} player is ${response.player?.id}`
          );

          if (response.player === undefined ) return;
          const p = response.player;
          setPlayer(p)

          setGames((prevGames) => [
            ...prevGames,
            {
              id: p.gameId,
              name: gameName,
              status: "CREATED",
              $typeName: "game.v1.Game"
            }
          ]);

          setGameStatus("CREATED");
        }}
      >
        <input
          value={gameName}
          onChange={(e) => setGameName(e.target.value)}
          placeholder="ゲーム名"
        />
        <button type="submit">createGame</button>
      </form>
    ) : null;

  return (
    <WebSocketProvider value={ws}>
      {(gameStatus === "STARTED") || (gameStatus === "JOINED") && player ? (
        <GameComponent
            message=""
            player={player}
            status={gameStatus}
        />
      ) : (
        <div>
          {/* ゲーム作成フォーム */}
          {createGameForm}

          {/* ゲーム情報表示 */}
          <div>Games</div>
          <ul>{gameList}</ul>
        </div>
      )}
    </WebSocketProvider>
  );
};

export default Lobby;
