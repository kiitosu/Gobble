import React, { useEffect, useState, useMemo, useRef } from "react";
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
  const ws = useRef<WebSocket | null>(null);
  const [gameStatus, setGameStatus] = useState("");
  const [player, setPlayer] = useState<Player>();
  const [games, setGames] = useState<Game[]>([]);
  const [gameName, setGameName] = useState("");
  const [cards, setCards] = useState<{ id: number; text: string }[]>([]);
  const [started, setStarted] = useState<boolean>(false);

  const transport = useMemo(
    () =>
      createConnectTransport({
        baseUrl: "http://localhost:8080",
      }),
    []
  );
  const createGameClient = createClient(CreateGameService, transport);
  const joinGameServiceclient = createClient(JoinGameService, transport);
  const startGameServiceclient = createClient(StartGameService, transport);
  const getGamesClient = useMemo(
    () => createClient(GetGamesService, transport),
    [transport]
  );

  // player情報が揃ったらWebSocket接続
  useEffect(() => {
    if (player && player.gameId && player.id && !ws.current) {
      ws.current = new WebSocket("ws://localhost:8080/ws");
      ws.current.onopen = () => {
        ws.current?.send(
          JSON.stringify({ game_id: player.gameId, player_id: player.id })
        );
      };

      const updateGames = async () => {
        const response = await getGamesClient.getGames({});
        setGames(response.games);
      };

      ws.current.onmessage = (e: MessageEvent) => {
        try {
          const msg = JSON.parse(e.data);
          console.log("socket receive msg", msg);

          if (msg.event === "CREATED") {
            updateGames();
          }

          if (msg.event === "STARTED" && !started) {
            setStarted(true);
            setGameStatus("STARTED")
          }

          if (msg.event === "card" && msg.card) {
            console.log("Received card:", msg.card);
            setCards((prev) => [...prev, msg.card]);
          }

        } catch {
          // ignore parse error
        }
      };
    }
    // クリーンアップ: Lobbyアンマウント時にWebSocket切断
    return () => {
      if (ws.current) {
        ws.current.close();
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
    gameList = games
      .filter((item) => item.status === "CREATED")
      .sort((a, b) => b.id - a.id) // 降順ソートを追加
      .map((item) => (
        <li key={item.id}>
          Game id {item.id} is {item.status} status
          <button
            disabled={item.status !== "CREATED"}
            onClick={async () => {
              const response = await joinGameServiceclient.joinGame({
                gameId: String(item.id),
                playerName: "unknown",
              });
              console.log(`Join the game ${response}`);
              setGameStatus("JOINED");
              setPlayer(response.player);
            }}
          >
            参加
          </button>
        </li>
      ));
  } else if (gameStatus === "CREATED") {
    gameList = games
      .filter((item) => item.status === "CREATED")
      .sort((a, b) => b.id - a.id) // 降順ソートを追加
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
          // ゲーム作成
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

          if (response.player === undefined) return;
          const p = response.player;
          setPlayer(p);

          setGames((prevGames) => [
            ...prevGames,
            {
              id: p.gameId,
              name: gameName,
              status: "CREATED",
              $typeName: "game.v1.Game",
            },
          ]);

          setGameStatus("CREATED");

          console.log("game status is CREATED");
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
    <>
      {gameStatus === "STARTED" || (gameStatus === "JOINED" && player) ? (
        <GameComponent
          message=""
          started={started}
          player={player}
          status={gameStatus}
          cards={cards}
        />
      ) : (
        <div>
          {/* ゲーム作成フォーム */}
          {createGameForm}

          {/* ゲーム情報表示 */}
          <div>Games</div>
          <div className="scrollable-list">
            <ul>{gameList}</ul>
          </div>
        </div>
      )}
    </>
  );
};

export default Lobby;
