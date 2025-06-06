import React, { useEffect, useState, useMemo } from "react";
import { WebSocketProvider } from "./WebSocketContext";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { JoinGameService, CreateGameService, GetGamesService, StartGameService } from "../gen/game/v1/game_pb";
import type { Game as ProtoGame, Player } from "../gen/game/v1/game_pb";

type Game = ProtoGame & {
  localStatus?: string;
};

type LobbyProps = {
  gameStatus: string;
  player?: Player;
  setPlayer: (player: Player) => void;
  setGameStatus: (status: string) => void;
  setStartedGame: (game: Game) => void;
};

const Lobby: React.FC<LobbyProps> = ({
  gameStatus,
  player,
  setPlayer,
  setGameStatus,
  setStartedGame,
}) => {
  const [ws, setWs] = useState<WebSocket | null>(null);

  // player情報が揃ったらWebSocket接続
  useEffect(() => {
    if (player && player.gameId && player.id && !ws) {
      const socket = new WebSocket("ws://localhost:8080/ws");
      socket.onopen = () => {
        socket.send(JSON.stringify({ game_id: player.gameId, player_id: player.id }));
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

  const transport = createConnectTransport({
    baseUrl: "http://localhost:8080"
  });
  const createGameClient = createClient(CreateGameService, transport);
  const joinGameServiceclient = createClient(JoinGameService, transport);
  const startGameServiceclient = createClient(StartGameService, transport);
  const getGamesClient = createClient(GetGamesService, transport);
  const [games, setGames] = useState<Game[]>([])
  const [gameName, setGameName] = useState("")

  useEffect(() => {
    const fetchGames = async () => {
      const response = await getGamesClient.getGames({});
      setGames((prevGames) =>
        response.games.map((newGame: Game) => {
          const old = prevGames.find((g) => g.id === newGame.id);
          return old && old.localStatus
            ? { ...newGame, localStatus: old.localStatus }
            : newGame;
        })
      );
    };
    fetchGames();
  }, [getGamesClient, setGames]);
  
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
            setGameStatus("JOINING");

            setGames((prevGames) =>
              prevGames.map((g) =>
                g.id === item.id ? { ...g, localStatus: "JOINING" } : g
              )
            );

            if (response.player) {
              setPlayer(response.player);
            }
          }}
        >
          参加
        </button>
      </li>
    ));
  } else if (gameStatus === "JOINING") {
    gameList = games
      .filter((item) => item.localStatus === "JOINING")
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
              setStartedGame(item);
            }}
          >
            開始
          </button>
        </li>
      ));
  }

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
      <div>
        {/* ゲーム作成フォーム */}
        {createGameForm}

        {/* ゲーム情報表示 */}
        <div>Games</div>
        <ul>{gameList}</ul>
      </div>
    </WebSocketProvider>
  );
};

export default Lobby;
