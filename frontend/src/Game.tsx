import type { Player } from "../gen/game/v1/game_pb";
import { DEAL_A_CARD, WAIT_FOR_OTHER_PLAYERS, NEED_ANSWER } from "./Lobby";

import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  ReportReadyService,
  SubmitAnswerService,
} from "../gen/game/v1/game_pb";

type Card = { id: number; text: string };

type GameProps = {
  message: string;
  started: boolean;
  player?: Player;
  status?: string;
  cards?: Card[];
  answer?: {
    playerId: number;
    isCorrect: boolean;
    answer: string;
    userAnswer: string;
  };
  dealACard: string;
  setDealACard: React.Dispatch<React.SetStateAction<string>>;
  scores: { player_id: number; score: number }[];
};

import React, { useState, useEffect } from "react";

import seedrandom from "seedrandom";

const GameComponent = (props: GameProps) => {
  const [showDialog, setShowDialog] = useState(false);

  useEffect(() => {
    if (props.answer && props.player) {
      setShowDialog(true);
    }
  }, [props.answer, props.player]);

  const transport = createConnectTransport({
    baseUrl: import.meta.env.VITE_BACKEND_URL || "http://localhost:8080",
  });

  const reportReadyServiceclient = createClient(ReportReadyService, transport);
  const submitAnswerServiceClient = createClient(
    SubmitAnswerService,
    transport
  );

  // カード受け取り準備完了通知をする
  const handleReadyClick = async () => {
    if (props.player) {
      props.setDealACard(WAIT_FOR_OTHER_PLAYERS);
      await reportReadyServiceclient.reportReady({
        playerId: String(props.player.id),
      });
    }
  };

  // 回答を通知する
  const handleSubmitAnswer = async (
    card1: Card,
    card2: Card,
    answer: string
  ) => {
    if (props.player) {
      await submitAnswerServiceClient.submitAnswer({
        playerId: String(props.player.id),
        card1: card1,
        card2: card2,
        answer: answer,
      });
      props.setDealACard(DEAL_A_CARD);
    }
  };

  // カードからシンボルを抽出する
  const iconMap: { [key: string]: string } = {
    "0": "🎲",
    "1": "🌞",
    "2": "🌙",
    "3": "⭐",
    "4": "☁️",
    "5": "⚡",
    "6": "🔥",
    "7": "💧",
    "8": "❄️",
    "9": "🌈",
    "10": "🍎",
    "11": "🍌",
    "12": "🍇",
    "13": "🍰",
    "14": "🍕",
    "15": "🐱",
    "16": "🐶",
    "17": "🐸",
    "18": "🐦",
    "19": "🐘",
    "20": "🐬",
    "21": "🚗",
    "22": "🚀",
    "23": "✈️",
    "24": "🚲",
    "25": "⛵",
    "26": "🎈",
    "27": "🎸",
    "28": "⌛",
    "29": "⏰",
    "30": "⚽",
  };

  // シンボルの数字配列を返す
  const extractSymbolNumbers = (text: string): string[] => {
    const match = text.match(/\[(.*?)\]/);
    return match ? match[1].split(" ") : [];
  };
  // 表示用にアイコンへ変換
  const toIcon = (num: string) => iconMap[num] || num;

  // カード内のシンボルをランダム配置するためのコンポーネント
  const SymbolRandomLayout = ({
    symbols,
    cardId,
  }: {
    symbols: string[];
    cardId: number;
  }) => {
    const containerRef = React.useRef<HTMLDivElement>(null);
    const [containerSize, setContainerSize] = useState({ width: 0, height: 0 });

    useEffect(() => {
      if (containerRef.current) {
        setContainerSize({
          width: containerRef.current.clientWidth,
          height: containerRef.current.clientHeight,
        });
      }
    }, []);

    // ゲーム番号とカード番号をシードにしたPRNGを作成
    const rng = React.useMemo(() => {
      // ゲーム番号を文字列として受け取る想定。なければ固定値
      const seed = (props.message || "default_seed") + "_" + cardId.toString();
      return seedrandom(seed);
    }, [props.message, cardId]);

    // シンボルのランダムな位置とサイズを計算（重ならないように）
    const minSize = 20;
    const maxSize = 70;
    const positions = React.useMemo(() => {
      const posArray: { top: number; left: number; size: number }[] = [];
      const rotations: number[] = [];
      const maxTop = containerSize.height - minSize;
      const maxLeft = containerSize.width - minSize;

      const isOverlap = (
        x1: number,
        y1: number,
        size1: number,
        x2: number,
        y2: number,
        size2: number
      ) => {
        const distance = Math.sqrt((x1 - x2) ** 2 + (y1 - y2) ** 2);
        return distance < (size1 + size2) / 2;
      };

      for (let i = 0; i < symbols.length; i++) {
        let top: number, left: number, size: number;
        let attempts = 0;
        do {
          size = Math.floor(rng() * (maxSize - minSize + 1)) + minSize;
          top = Math.floor(rng() * (maxTop > 0 ? maxTop : 0));
          left = Math.floor(rng() * (maxLeft > 0 ? maxLeft : 0));
          attempts++;
          // 10回試しても重ならなければ強制的に配置
          if (attempts > 10) break;
        } while (
          posArray.some((pos) =>
            isOverlap(pos.left, pos.top, pos.size, left, top, size)
          )
        );
        posArray.push({ top, left, size });
        // 回転角度もランダムに決定（0〜359度）
        rotations.push(Math.floor(rng() * 360));
      }
      return { positions: posArray, rotations };
    }, [symbols, containerSize, rng]);

    return (
      <div
        ref={containerRef}
        style={{
          position: "relative",
          width: "100%",
          height: "80px",
          border: "1px solid #ccc",
          borderRadius: "8px",
          backgroundColor: "#fff",
          marginTop: "8px",
        }}
      >
        {symbols.map((num, idx) => (
          <button
            key={idx}
            style={{
              position: "absolute",
              top: positions.positions[idx]?.top ?? 0,
              left: positions.positions[idx]?.left ?? 0,
              width: positions.positions[idx]?.size ?? 40,
              height: positions.positions[idx]?.size ?? 40,
              borderRadius: "50%",
              border: "1px solid #1976d2",
              backgroundColor: "#1976d2",
              color: "white",
              fontSize: `${(positions.positions[idx]?.size ?? 40) * 0.8}px`,
              lineHeight: 1,
              textAlign: "center",
              userSelect: "none",
              cursor: "pointer",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              padding: 0,
              transform: `rotate(${positions.rotations[idx]}deg)`,
              transition: "transform 0.3s ease",
            }}
            onClick={() =>
              handleSubmitAnswer(
                props.cards![props.cards!.length - 1],
                props.cards![props.cards!.length - 2],
                num
              )
            }
            disabled={props.dealACard !== NEED_ANSWER}
            aria-label={`シンボル ${toIcon(num)}`}
          >
            {toIcon(num)}
          </button>
        ))}
      </div>
    );
  };

  return (
    <>
      {/* ゲーム開催中 */}
      {props.status === "STARTED" && (
        <div>
          {/* 全員分のスコア表示 */}
          <div style={{ fontWeight: "bold", marginBottom: "8px" }}>
            <h3>参加者全員のスコア</h3>
            {props.scores && props.scores.length > 0 ? (
              <>
                {/* 自分のスコアを一番上に表示 */}
                {props.player &&
                  props.scores.find(
                    (s) => s.player_id === props.player!.id
                  ) && (
                    <div style={{ color: "#1976d2" }}>
                      あなたのスコア:{" "}
                      {
                        props.scores.find(
                          (s) => s.player_id === props.player!.id
                        )?.score
                      }
                      点
                    </div>
                  )}
                {/* 他のプレイヤーのスコア */}
                {props.scores
                  .filter(
                    (score) =>
                      !props.player || score.player_id !== props.player.id
                  )
                  .map((score) => (
                    <div key={score.player_id}>
                      プレイヤー {score.player_id}: {score.score}点
                    </div>
                  ))}
              </>
            ) : (
              <div>スコア情報なし</div>
            )}
          </div>

          {showDialog && props.answer && props.player && (
            <div
              style={{
                position: "fixed",
                top: "50%",
                left: "50%",
                transform: "translate(-50%, -50%)",
                background: "#f5faff",
                border: "2px solid #1976d2",
                borderRadius: "12px",
                boxShadow: "0 4px 16px rgba(25,118,210,0.18)",
                padding: "28px 36px 24px 36px",
                zIndex: 1000,
                minWidth: "340px",
                textAlign: "center",
                color: "#1976d2",
              }}
            >
              <button
                onClick={() => setShowDialog(false)}
                style={{
                  position: "absolute",
                  top: 8,
                  right: 12,
                  background: "transparent",
                  border: "none",
                  fontSize: "1.5em",
                  color: "#1976d2",
                  cursor: "pointer",
                  fontWeight: "bold",
                  lineHeight: 1,
                }}
                aria-label="閉じる"
              >
                ×
              </button>
              <div
                style={{
                  fontWeight: "bold",
                  fontSize: "1.3em",
                  marginBottom: "16px",
                  color: "#1976d2",
                }}
              >
                {props.answer.playerId === props.player.id
                  ? props.answer.isCorrect
                    ? "正解！"
                    : "不正解"
                  : `Player ${props.player.id} ${
                      props.answer.isCorrect ? "correct" : "wrong"
                    }!`}
              </div>
              {props.answer && (
                <>
                  <div
                    style={{
                      color: "#43a047",
                      fontWeight: "bold",
                      marginBottom: "6px",
                    }}
                  >
                    正解シンボル: {toIcon(String(props.answer.answer))}
                  </div>
                  <div
                    style={{
                      color: "#1976d2",
                      fontWeight: "bold",
                      marginBottom: "6px",
                    }}
                  >
                    選択されたシンボル:{" "}
                    {toIcon(String(props.answer.userAnswer))}
                  </div>
                </>
              )}
              <div
                style={{
                  display: "flex",
                  flexDirection: "column",
                  gap: "18px",
                  alignItems: "center",
                }}
              >
                {props.cards &&
                  props.cards
                    .slice(-2)
                    .reverse()
                    .map((card) => {
                      const symbolNums = extractSymbolNumbers(card.text);
                      return (
                        <div
                          key={card.id}
                          style={{ marginBottom: "0px", width: "100%" }}
                        >
                          <div
                            style={{
                              display: "flex",
                              gap: "16px",
                              justifyContent: "flex-start",
                            }}
                          >
                            {symbolNums.map((num, sidx) => {
                              // 最新カード（idx === 0）かつ正解シンボルのみ色変更
                              const isCorrect =
                                props.answer &&
                                String(num) === String(props.answer.answer);
                              return (
                                <div
                                  key={sidx}
                                  style={{
                                    background: isCorrect ? "#43a047" : "#222",
                                    color: "#fff",
                                    borderRadius: "10px",
                                    padding: "8px 22px",
                                    fontSize: "1.15em",
                                    fontWeight: "bold",
                                    letterSpacing: "1px",
                                    display: "inline-block",
                                    marginBottom: "2px",
                                  }}
                                >
                                  {toIcon(num)}
                                </div>
                              );
                            })}
                          </div>
                        </div>
                      );
                    })}
              </div>
            </div>
          )}

          <button
            onClick={handleReadyClick}
            disabled={props.dealACard !== DEAL_A_CARD}
          >
            {props.dealACard}
          </button>

          <h3>カード一覧</h3>

          <div
            style={{
              height: "240px",
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
            }}
          >
            {props.cards &&
              [...props.cards].reverse().map((card) => {
                const symbolNums = extractSymbolNumbers(card.text);
                return (
                  <div
                    key={card.id}
                    style={{
                      marginBottom: "16px",
                      border: "2px solid gray",
                      borderRadius: "8px",
                      padding: "12px",
                      width: `${symbolNums.length * 60}px`,
                      backgroundColor: "white",
                    }}
                  >
                    <SymbolRandomLayout symbols={symbolNums} cardId={card.id} />
                  </div>
                );
              })}
          </div>
        </div>
      )}

      {/* ゲーム参加中・開催まち */}
      {props.status == "JOINED" && <>Waiting for game to start...</>}
    </>
  );
};
export default GameComponent;
