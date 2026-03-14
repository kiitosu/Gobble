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

export const DEAL_A_CARD = "新しいカードを要求する";
export const WAIT_FOR_OTHER_PLAYERS = "他のユーザのカード要求を待っています";
export const NEED_ANSWER = "回答してください";

const Lobby: React.FC<LobbyProps> = ({}) => {
  const [dealACard, setDealACard] = useState<string>(DEAL_A_CARD);
  const ws = useRef<WebSocket | null>(null);
  const [gameStatus, setGameStatus] = useState("");
  const [player, setPlayer] = useState<Player>();
  const [games, setGames] = useState<Game[]>([]);
  const [gameName, setGameName] = useState("");
  const [cards, setCards] = useState<{ id: number; text: string }[]>([]);
  const [started, setStarted] = useState<boolean>(false);
  const [answer, setAnswer] = useState<{
    playerId: number;
    isCorrect: boolean;
    answer: string;
    userAnswer: string;
  }>();
  const [scores, setScores] = useState<{ player_id: number; score: number }[]>(
    []
  );
  const [gameOver, setGameOver] = useState<boolean>(false);
  const [countdown, setCountdown] = useState<number>(0);
  const cardsRef = useRef<{ id: number; text: string }[]>([]);

  const transport = useMemo(
    () =>
      createConnectTransport({
        baseUrl: import.meta.env.VITE_BACKEND_URL || "http://localhost:8080",
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
      const backendUrl =
        import.meta.env.VITE_BACKEND_URL || "http://localhost:8080";
      const wsUrl = backendUrl.replace(/^http/, "ws") + "/ws";
      ws.current = new WebSocket(wsUrl);
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
            setGameStatus("STARTED");
          }

          if (msg.event === "ANSWERED") {
            console.log("ANSWERED msg", msg);
            setAnswer({
              playerId: Number(msg.player_id),
              isCorrect: msg.is_correct,
              answer: msg.correct_symbol ?? "",
              userAnswer: String(msg.answer ?? ""),
            });
            setDealACard(DEAL_A_CARD);
            if (msg.scores) {
              setScores(msg.scores);
            }
          }

          if (msg.event === "card" && msg.card) {
            console.log("Received card:", msg.card);
            const pendingCard = msg.card;
            const currentCards = cardsRef.current;

            if (currentCards.length < 1) {
              // 最初のカードは即座に表示
              const next = [...currentCards, pendingCard];
              cardsRef.current = next;
              setCards(next);
              setDealACard(DEAL_A_CARD);
            } else {
              // 2枚目以降: カウントダウン後に表示
              setCountdown(3);
              let count = 3;
              const interval = setInterval(() => {
                count--;
                if (count <= 0) {
                  clearInterval(interval);
                  setCountdown(0);
                  const next = [...cardsRef.current, pendingCard];
                  cardsRef.current = next;
                  setCards(next);
                  setDealACard(NEED_ANSWER);
                } else {
                  setCountdown(count);
                }
              }, 1000);
            }
          }

          if (msg.event === "JOINED") {
            setGameStatus("READY");
          }

          if (msg.event === "GAME_OVER") {
            setGameOver(true);
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

  const createdGames = games
    .filter((item) => item.status === "CREATED")
    .sort((a, b) => b.id - a.id);

  // ゲームオーバー画面
  if (
    (gameStatus === "STARTED" || (gameStatus === "JOINED" && player)) &&
    gameOver
  ) {
    const sorted = [...scores].sort((a, b) => b.score - a.score);
    const isWinner = sorted[0]?.player_id === player?.id;
    return (
      <div className="min-h-screen bg-bg flex flex-col items-center justify-center p-8">
        <h1 className="text-5xl font-bold text-primary mb-2">
          {isWinner ? "You Win!" : "Game Over"}
        </h1>
        <p className="text-2xl text-text mb-8">
          {isWinner ? "おめでとうございます！" : "残念...！"}
        </p>
        <div className="bg-card rounded-2xl shadow-lg p-8 w-full max-w-sm">
          <h2 className="text-lg font-bold text-text-muted mb-4 text-center">
            最終スコア
          </h2>
          <div className="space-y-3">
            {sorted.map((score, idx) => (
              <div
                key={score.player_id}
                className={`flex items-center justify-between p-3 rounded-xl ${
                  score.player_id === player?.id
                    ? "bg-primary/10 border-2 border-primary"
                    : "bg-gray-50"
                }`}
              >
                <div className="flex items-center gap-3">
                  <span className="text-2xl font-bold text-text-muted">
                    {idx + 1}.
                  </span>
                  <span className="font-semibold text-text">
                    {score.player_id === player?.id
                      ? "あなた"
                      : `プレイヤー ${score.player_id}`}
                  </span>
                </div>
                <span className="text-xl font-bold text-primary">
                  {score.score}点
                </span>
              </div>
            ))}
          </div>
        </div>
      </div>
    );
  }

  // ゲーム中画面
  if (gameStatus === "STARTED" || (gameStatus === "JOINED" && player)) {
    return (
      <div className="min-h-screen bg-bg">
        <GameComponent
          message=""
          started={started}
          player={player}
          status={gameStatus}
          cards={cards}
          answer={answer}
          dealACard={dealACard}
          setDealACard={setDealACard}
          scores={scores}
          countdown={countdown}
        />
      </div>
    );
  }

  // ロビー画面
  return (
    <div className="min-h-screen bg-bg text-text">
      <div className="max-w-2xl mx-auto px-4 py-12">
        {/* ヘッダー */}
        <div className="text-center mb-12">
          <h1 className="text-5xl font-bold text-primary mb-2">Gobble</h1>
          <p className="text-text-muted">
            シンボルを見つけて素早くマッチ！
          </p>
        </div>

        {/* ゲーム作成フォーム */}
        {gameStatus === "" && (
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              const response = await createGameClient.createGame({
                gameName: gameName,
                playerName: gameName,
              });
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
            }}
            className="flex flex-col sm:flex-row gap-3 items-center justify-center mb-10"
          >
            <input
              value={gameName}
              onChange={(e) => setGameName(e.target.value)}
              placeholder="プレイヤー名を入力"
              className="px-4 py-2.5 border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-primary bg-card text-text"
            />
            <button
              type="submit"
              className="px-6 py-2.5 bg-primary text-white rounded-xl font-semibold hover:bg-primary-dark transition shadow-sm"
            >
              ゲームを作成
            </button>
          </form>
        )}

        {/* ステータスメッセージ */}
        {gameStatus === "CREATED" && (
          <div className="text-center mb-6 p-4 bg-accent/10 rounded-xl border border-accent/30">
            <p className="text-accent font-semibold">
              他のプレイヤーの参加を待っています...
            </p>
          </div>
        )}

        {/* ゲーム一覧 */}
        <div className="mb-4">
          <h2 className="text-lg font-bold text-text mb-4">ゲーム一覧</h2>
        </div>

        {createdGames.length === 0 ? (
          <div className="text-center py-12 text-text-muted">
            <p className="text-lg">参加できるゲームがありません</p>
            <p className="text-sm mt-1">新しいゲームを作成してください</p>
          </div>
        ) : (
          <div className="space-y-3">
            {createdGames.map((item) => (
              <div
                key={item.id}
                className="flex items-center justify-between p-4 bg-card rounded-xl shadow-sm border border-gray-100"
              >
                <div className="flex items-center gap-3">
                  <span className="text-2xl">🎮</span>
                  <div>
                    <span className="font-semibold text-text">
                      Game #{item.id}
                    </span>
                    <span className="ml-3 text-xs px-2 py-0.5 bg-success/10 text-success rounded-full font-medium">
                      募集中
                    </span>
                  </div>
                </div>

                {gameStatus === "" ? (
                  <button
                    onClick={async () => {
                      const response = await joinGameServiceclient.joinGame({
                        gameId: String(item.id),
                        playerName: "unknown",
                      });
                      setGameStatus("JOINED");
                      setPlayer(response.player);
                    }}
                    className="px-5 py-2 bg-primary text-white rounded-xl font-semibold hover:bg-primary-dark transition text-sm shadow-sm"
                  >
                    参加
                  </button>
                ) : (
                  <button
                    onClick={async () => {
                      const response = await startGameServiceclient.startGame({
                        gameId: String(item.id),
                        userId: String(player?.id),
                      });
                      console.log(response);
                      setGameStatus("STARTED");
                    }}
                    disabled={gameStatus !== "READY"}
                    className="px-5 py-2 bg-success text-white rounded-xl font-semibold hover:brightness-110 transition text-sm shadow-sm disabled:opacity-40 disabled:cursor-not-allowed"
                  >
                    開始
                  </button>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default Lobby;
