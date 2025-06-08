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
  answer?: { playerId: number; isCorrect: boolean; answer: string; userAnswer: string };
  dealACard: string;
  setDealACard: React.Dispatch<React.SetStateAction<string>>;
  scores: { player_id: number; score: number }[];
};

import React, { useState, useEffect } from "react";

const GameComponent = (props: GameProps) => {
  const [showDialog, setShowDialog] = useState(false);

  useEffect(() => {
    if (props.answer && props.player) {
      setShowDialog(true);
    }
  }, [props.answer, props.player]);

  const transport = createConnectTransport({
    baseUrl: "http://localhost:8080",
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
  const extractSymbol = (text: string): string[] => {
    const match = text.match(/\[(.*?)\]/);
    return match ? match[1].split(" ") : [];
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
                    正解シンボル: {String(props.answer.answer)}
                  </div>
                  <div
                    style={{
                      color: "#1976d2",
                      fontWeight: "bold",
                      marginBottom: "6px",
                    }}
                  >
                    選択されたシンボル: {String(props.answer.userAnswer)}
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
                    .map((card, idx) => {
                      const symbols = extractSymbol(card.text);
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
                            {symbols.map((symbol, sidx) => {
                              // 最新カード（idx === 0）かつ正解シンボルのみ色変更
                              const isCorrect =
                                props.answer &&
                                String(symbol) === String(props.answer.answer);
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
                                  {symbol}
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

          <div style={{ height: "240px" }}>
            {props.cards &&
              [...props.cards].reverse().map((card, index) => (
                <div key={`${card.id}-${index}`}>
                  <div>
                    カードID: {card.id} 内容: {card.text}
                  </div>
                  {props.cards && props.cards.length >= 2 && (
                    <div>
                      {extractSymbol(card.text).map((symbol, idx) => (
                        <button
                          onClick={() =>
                            handleSubmitAnswer(
                              props.cards![props.cards!.length - 1],
                              props.cards![props.cards!.length - 2],
                              symbol
                            )
                          }
                          key={`${card.id}-symbol-${idx}`}
                          disabled={
                            props.dealACard !== NEED_ANSWER || index !== 0
                          } // 最新のカード（indexが0）のみ有効
                        >
                          {symbol}
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              ))}
          </div>
        </div>
      )}

      {/* ゲーム参加中・開催まち */}
      {props.status == "JOINED" && <>Waiting for game to start...</>}
    </>
  );
};
export default GameComponent;
