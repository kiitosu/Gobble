import React, { useEffect, useState, useMemo, useRef } from "react";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  JoinGameService,
  CreateGameService,
  GetGamesService,
  StartGameService,
  DeleteGameService,
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
  const [playerName, setPlayerName] = useState(
    () => `プレイヤー${Math.floor(Math.random() * 9000 + 1000)}`
  );
  const [cards, setCards] = useState<{ id: number; text: string }[]>([]);
  const [started, setStarted] = useState<boolean>(false);
  const [answer, setAnswer] = useState<{
    playerId: number;
    isCorrect: boolean;
    answer: string;
    userAnswer: string;
  }>();
  const [scores, setScores] = useState<{ player_id: number; score: number; name?: string }[]>(
    []
  );
  const [gameOver, setGameOver] = useState<boolean>(false);
  const [disconnected, setDisconnected] = useState<boolean>(false);
  const [countdown, setCountdown] = useState<number>(0);
  const cardsRef = useRef<{ id: number; text: string }[]>([]);
  const [roundResults, setRoundResults] = useState<
    { playerId: number; isCorrect: boolean }[]
  >([]);
  const [cardCount, setCardCount] = useState<number>(31);
  const [totalRounds, setTotalRounds] = useState<number>(0);

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
  const deleteGameClient = createClient(DeleteGameService, transport);
  const getGamesClient = useMemo(
    () => createClient(GetGamesService, transport),
    [transport]
  );

  const resetToLobby = () => {
    setPlayer(undefined);
    setGameStatus("");
    setStarted(false);
    setCards([]);
    setScores([]);
    setGameOver(false);
    setDisconnected(false);
    setDealACard(DEAL_A_CARD);
    setAnswer(undefined);
    setRoundResults([]);
    setTotalRounds(0);
    setCountdown(0);
    cardsRef.current = [];
    if (ws.current) {
      ws.current.close();
      ws.current = null;
    }
  };

  const updateGames = async () => {
    try {
      const response = await getGamesClient.getGames({});
      setGames(response.games);
      setGamesLoaded(true);
    } catch (e) {
      console.error("ゲーム一覧の取得に失敗:", e);
      setGamesLoaded(true);
    }
  };

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

      ws.current.onmessage = (e: MessageEvent) => {
        try {
          const msg = JSON.parse(e.data);
          console.log("socket receive msg", msg);

          if (msg.event === "CREATED") {
            updateGames();
          }

          if (msg.event === "PLAYERS" && msg.players) {
            setScores(msg.players.map((p: any) => ({
              player_id: p.player_id,
              score: p.score,
              name: p.name,
            })));
          }

          if (msg.event === "STARTED" && !started) {
            setStarted(true);
            if (msg.players && msg.players.length > 0) {
              setScores(msg.players.map((p: any) => ({
                player_id: p.player_id,
                score: p.score,
                name: p.name,
              })));
            }
            if (msg.total_rounds) {
              setTotalRounds(msg.total_rounds);
            }
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
            setRoundResults((prev) => [
              ...prev,
              { playerId: Number(msg.player_id), isCorrect: msg.is_correct },
            ]);
            setDealACard(DEAL_A_CARD);
            if (msg.scores) {
              setScores((prev) =>
                msg.scores.map((s: any) => ({
                  ...s,
                  name: prev.find((p: any) => p.player_id === s.player_id)?.name ?? s.name,
                }))
              );
            }
          }

          if (msg.event === "card" && msg.card) {
            console.log("Received card:", msg.card);
            const pendingCard = msg.card;
            const currentCards = cardsRef.current;

            if (currentCards.length < 1) {
              const next = [...currentCards, pendingCard];
              cardsRef.current = next;
              setCards(next);
              setDealACard(DEAL_A_CARD);
            } else {
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
            updateGames();
            if (msg.total_rounds) {
              setTotalRounds(msg.total_rounds);
            }
            if (msg.players && msg.players.length > 0) {
              setScores(msg.players.map((p: any) => ({
                player_id: p.player_id,
                score: p.score,
                name: p.name,
              })));
            }
          }

          if (msg.event === "disconnect") {
            setDisconnected(true);
            setGameOver(true);
          }

          if (msg.event === "GAME_OVER") {
            // 最後のラウンドの結果を見せてから結果画面に遷移
            setTimeout(() => {
              setGameOver(true);
            }, 3000);
          }
        } catch {
          // ignore parse error
        }
      };
    }
    return () => {
      if (ws.current) {
        ws.current.close();
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [player]);

  // ロビーWebSocket + ポーリングフォールバック
  const lobbyWs = useRef<WebSocket | null>(null);
  useEffect(() => {
    if (gameStatus === "STARTED") return;

    updateGames();

    // WebSocketでリアルタイム通知
    const backendUrl =
      import.meta.env.VITE_BACKEND_URL || "http://localhost:8080";
    const wsUrl = backendUrl.replace(/^http/, "ws") + "/ws/lobby";
    lobbyWs.current = new WebSocket(wsUrl);
    lobbyWs.current.onmessage = (e: MessageEvent) => {
      try {
        const msg = JSON.parse(e.data);
        if (msg.event === "CREATED" || msg.event === "JOINED" || msg.event === "DELETED" || msg.event === "STARTED") {
          updateGames();
        }
      } catch {}
    };

    // フォールバック: 5秒ごとにポーリング
    const interval = setInterval(updateGames, 5000);

    return () => {
      lobbyWs.current?.close();
      lobbyWs.current = null;
      clearInterval(interval);
    };
  }, [gameStatus]);

  const [gamesLoaded, setGamesLoaded] = useState(false);
  const visibleGames = games
    .sort((a, b) => b.id - a.id);

  // ゲームオーバー画面
  if (gameStatus === "STARTED" && gameOver) {
    if (disconnected) {
      return (
        <div className="min-h-screen bg-bg flex flex-col items-center justify-center p-8">
          <h1 className="text-4xl font-bold text-text mb-4">
            通信が切断されました
          </h1>
          <p className="text-xl text-text-muted mb-8">
            対戦相手の接続が切れたため、ゲームが終了しました。
          </p>
          <button
            onClick={resetToLobby}
            className="px-8 py-3 bg-primary text-white rounded-xl font-semibold hover:bg-primary-dark transition shadow-sm"
          >
            ロビーに戻る
          </button>
        </div>
      );
    }

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
                      : (score.name || `プレイヤー ${score.player_id}`)}
                  </span>
                </div>
                <span className="text-xl font-bold text-primary">
                  {score.score}点
                </span>
              </div>
            ))}
          </div>
        </div>
        <button
          onClick={resetToLobby}
          className="mt-8 px-8 py-3 bg-primary text-white rounded-xl font-semibold hover:bg-primary-dark transition shadow-sm"
        >
          ロビーに戻る
        </button>
      </div>
    );
  }

  // ゲーム中画面
  if (gameStatus === "STARTED") {
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
          roundResults={roundResults}
          totalRounds={totalRounds}
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

        {/* プレイヤー名入力 */}
        <div className="flex items-center justify-center gap-3 mb-8">
          <label className="text-sm font-semibold text-text-muted">あなたの名前:</label>
          <input
            value={playerName}
            onChange={(e) => setPlayerName(e.target.value)}
            placeholder="名前を入力"
            className="px-4 py-2 border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-primary bg-card text-text w-48"
          />
        </div>

        {/* ゲーム作成フォーム */}
        <form
          onSubmit={async (e) => {
            e.preventDefault();
            if (!gameName.trim()) return;
            try {
              await createGameClient.createGame({
                gameName: gameName,
                cardCount: cardCount,
              });
              setGameName("");
              updateGames();
            } catch (err: any) {
              console.error("ゲーム作成に失敗:", err);
              if (err?.code === "already_exists" || err?.message?.includes("同名")) {
                alert("同じ名前のゲームが既に存在します。別の名前にしてください。");
              } else {
                alert("ゲーム作成に失敗しました。バックエンドが起動しているか確認してください。");
              }
            }
          }}
          className="flex flex-col sm:flex-row gap-3 items-center justify-center mb-10"
        >
          <input
            value={gameName}
            onChange={(e) => setGameName(e.target.value)}
            placeholder="ゲーム名を入力"
            className="px-4 py-2.5 border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-primary bg-card text-text"
          />
          <select
            value={cardCount}
            onChange={(e) => setCardCount(Number(e.target.value))}
            className="px-4 py-2.5 border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-primary bg-card text-text"
          >
            <option value={6}>5ラウンド</option>
            <option value={11}>10ラウンド</option>
            <option value={16}>15ラウンド</option>
            <option value={21}>20ラウンド</option>
            <option value={31}>30ラウンド（フル）</option>
          </select>
          <button
            type="submit"
            className="px-6 py-2.5 bg-primary text-white rounded-xl font-semibold hover:bg-primary-dark transition shadow-sm"
          >
            ゲームを作成
          </button>
        </form>

        {/* ゲーム一覧 */}
        <div className="mb-4">
          <h2 className="text-lg font-bold text-text mb-4">ゲーム一覧</h2>
        </div>

        {!gamesLoaded ? (
          <div className="text-center py-12 text-text-muted">
            <p className="text-lg">読み込み中...</p>
          </div>
        ) : visibleGames.length === 0 ? (
          <div className="text-center py-12 text-text-muted">
            <p className="text-lg">参加できるゲームがありません</p>
            <p className="text-sm mt-1">新しいゲームを作成してください</p>
          </div>
        ) : (
          <div className="space-y-3">
            {visibleGames.map((item) => {
              const isStarted = item.status === "STARTED";
              const isFinished = item.status === "FINISHED";
              return (
              <div
                key={item.id}
                className={`flex items-center justify-between p-4 bg-card rounded-xl shadow-sm border border-gray-100 ${isStarted || isFinished ? "opacity-60" : ""}`}
              >
                <div className="flex items-center gap-3">
                  <span className="text-2xl">🎮</span>
                  <div>
                    <span className="font-semibold text-text">
                      {item.name || `Game #${item.id}`}
                    </span>
                    {isFinished ? (
                      <span className="ml-2 text-xs px-2 py-0.5 bg-gray-200 text-text-muted rounded-full font-medium">
                        終了
                      </span>
                    ) : isStarted ? (
                      <span className="ml-2 text-xs px-2 py-0.5 bg-accent/10 text-accent rounded-full font-medium">
                        開始済み
                      </span>
                    ) : (
                      <span className="ml-2 text-xs px-2 py-0.5 bg-gray-100 text-text-muted rounded-full font-medium">
                        {item.playerCount}人参加中
                      </span>
                    )}
                    <span className="ml-1 text-xs px-2 py-0.5 bg-primary/10 text-primary rounded-full font-medium">
                      {item.totalRounds}ラウンド
                    </span>
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  {isStarted || isFinished ? null : !player ? (
                    <>
                      <button
                        onClick={async () => {
                          try {
                            const response = await joinGameServiceclient.joinGame({
                              gameId: String(item.id),
                              playerName: playerName.trim() || "player",
                            });
                            setPlayer(response.player);
                            setGameStatus("JOINED");
                            setTotalRounds(0);
                            updateGames();
                          } catch (err) {
                            console.error("ゲーム参加に失敗:", err);
                            alert("ゲーム参加に失敗しました。");
                          }
                        }}
                        className="px-5 py-2 bg-primary text-white rounded-xl font-semibold hover:bg-primary-dark transition text-sm shadow-sm"
                      >
                        参加
                      </button>
                      {item.playerCount === 0 && (
                        <button
                          onClick={async () => {
                            if (!confirm(`「${item.name || `Game #${item.id}`}」を削除しますか？`)) return;
                            try {
                              await deleteGameClient.deleteGame({ gameId: String(item.id) });
                              updateGames();
                            } catch (err) {
                              console.error("ゲーム削除に失敗:", err);
                              alert("ゲーム削除に失敗しました。");
                            }
                          }}
                          className="px-3 py-2 text-danger hover:bg-danger/10 rounded-xl transition text-sm"
                          title="削除"
                        >
                          削除
                        </button>
                      )}
                    </>
                  ) : player.gameId === item.id ? (
                    <button
                      onClick={async () => {
                        await startGameServiceclient.startGame({
                          gameId: String(item.id),
                          userId: String(player?.id),
                        });
                      }}
                      disabled={item.playerCount < 2}
                      className="px-5 py-2 bg-success text-white rounded-xl font-semibold hover:brightness-110 transition text-sm shadow-sm disabled:opacity-40 disabled:cursor-not-allowed"
                    >
                      {item.playerCount < 2 ? "対戦者を待っています..." : "開始"}
                    </button>
                  ) : null}
                </div>
              </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};

export default Lobby;
