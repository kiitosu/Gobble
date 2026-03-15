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
  scores: { player_id: number; score: number; name?: string }[];
  countdown: number;
  roundResults: { playerId: number; isCorrect: boolean }[];
  totalRounds: number;
};

import React, { useState, useEffect } from "react";

import seedrandom from "seedrandom";

const GameComponent = (props: GameProps) => {
  const [showResult, setShowResult] = useState(false);
  const [isMoving, setIsMoving] = useState(false);

  useEffect(() => {
    if (props.answer && props.player) {
      setShowResult(true);
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
      // 移動アニメーション開始
      setIsMoving(true);
      setTimeout(() => {
        setIsMoving(false);
        setShowResult(false);
        props.setDealACard(WAIT_FOR_OTHER_PLAYERS);
        reportReadyServiceclient.reportReady({
          playerId: String(props.player!.id),
        });
      }, 500);
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

  const iconMap: { [key: string]: string } = {
    "0": "🎲", "1": "🌞", "2": "🌙", "3": "⭐", "4": "☁️",
    "5": "⚡", "6": "🔥", "7": "💧", "8": "❄️", "9": "🌈",
    "10": "🍎", "11": "🍌", "12": "🍇", "13": "🍰", "14": "🍕",
    "15": "🐱", "16": "🐶", "17": "🐸", "18": "🐦", "19": "🐘",
    "20": "🐬", "21": "🚗", "22": "🚀", "23": "✈️", "24": "🚲",
    "25": "⛵", "26": "🎈", "27": "🎸", "28": "⌛", "29": "⏰",
    "30": "⚽",
  };

  const extractSymbolNumbers = (text: string): string[] => {
    const match = text.match(/\[(.*?)\]/);
    return match ? match[1].split(" ") : [];
  };

  const toIcon = (num: string) => iconMap[num] || num;

  // カード内のシンボルをランダム配置するためのコンポーネント
  // 円形カードの内側に収まるよう、中心座標ベースで配置
  const SymbolRandomLayout = ({
    symbols,
    cardId,
    highlightSymbol,
    wrongSymbol,
  }: {
    symbols: string[];
    cardId: number;
    highlightSymbol?: string;
    wrongSymbol?: string;
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

    const rng = React.useMemo(() => {
      const seed = (props.message || "default_seed") + "_" + cardId.toString();
      return seedrandom(seed);
    }, [props.message, cardId]);

    // 事前定義されたスロット（中心からの相対位置 -1〜1、サイズ比率）
    // 6シンボル用: 中央に1つ大きめ + 周囲に5つ（サイズにバリエーション）
    const slotTemplates: { x: number; y: number; size: number }[][] = [
      // 6個用
      [
        { x: 0.0,  y: 0.0,  size: 0.42 },
        { x: 0.0,  y: -0.55, size: 0.28 },
        { x: 0.52, y: -0.17, size: 0.20 },
        { x: 0.32, y: 0.45,  size: 0.14 },
        { x: -0.32, y: 0.45, size: 0.24 },
        { x: -0.52, y: -0.17, size: 0.12 },
      ],
      // 6個用 別パターン
      [
        { x: -0.20, y: -0.10, size: 0.40 },
        { x: 0.35,  y: -0.35, size: 0.22 },
        { x: 0.35,  y: 0.30,  size: 0.28 },
        { x: -0.15, y: 0.50,  size: 0.12 },
        { x: -0.50, y: 0.22,  size: 0.16 },
        { x: 0.05,  y: -0.58, size: 0.14 },
      ],
      // 6個用 さらに別パターン
      [
        { x: 0.18,  y: 0.10,  size: 0.44 },
        { x: -0.38, y: -0.30, size: 0.24 },
        { x: 0.42,  y: -0.35, size: 0.14 },
        { x: -0.48, y: 0.25,  size: 0.18 },
        { x: 0.15,  y: 0.52,  size: 0.12 },
        { x: -0.12, y: -0.55, size: 0.16 },
      ],
    ];

    const positions = React.useMemo(() => {
      const containerR = Math.min(containerSize.width, containerSize.height) / 2;
      const centerX = containerSize.width / 2;
      const centerY = containerSize.height / 2;

      // カードIDに基づいてテンプレートを選択
      const templateIdx = Math.floor(rng() * slotTemplates.length);
      const template = slotTemplates[templateIdx];

      // スロットの割り当てをシャッフル（どのシンボルがどの位置に来るか）
      const indices = symbols.map((_, i) => i);
      for (let i = indices.length - 1; i > 0; i--) {
        const j = Math.floor(rng() * (i + 1));
        [indices[i], indices[j]] = [indices[j], indices[i]];
      }

      const placed = symbols.map((_, symIdx) => {
        const slotIdx = indices[symIdx] % template.length;
        const slot = template[slotIdx];
        return {
          cx: centerX + slot.x * containerR,
          cy: centerY + slot.y * containerR,
          size: slot.size * containerR * 2,
        };
      });

      const rotations = symbols.map(() =>
        Math.floor(rng() * 50) - 25
      );

      return { placed, rotations };
    }, [symbols, containerSize, rng]);

    // サイズが小さいほどz-indexを高くして前面に配置
    const maxSize = Math.max(...positions.placed.map(p => p?.size ?? 0));

    return (
      <div ref={containerRef} className="relative w-full h-full">
        {symbols.map((num, idx) => {
          const p = positions.placed[idx];
          if (!p) return null;
          const isHighlighted = highlightSymbol !== undefined && num === highlightSymbol;
          const isWrong = wrongSymbol !== undefined && num === wrongSymbol && num !== highlightSymbol;
          const zIndex = isHighlighted || isWrong ? 20 : Math.round((1 - p.size / maxSize) * 10) + 1;
          // クリック領域は絵文字サイズに合わせて縮小
          const hitSize = p.size * 0.8;
          return (
            <button
              key={idx}
              className="absolute flex items-center justify-center cursor-pointer
                         hover:scale-125 transition-transform duration-150 disabled:opacity-40
                         select-none p-0 bg-transparent border-none"
              style={{
                top: p.cy - hitSize / 2,
                left: p.cx - hitSize / 2,
                width: hitSize,
                height: hitSize,
                fontSize: `${p.size * 0.75}px`,
                transform: `rotate(${positions.rotations[idx]}deg)`,
                zIndex,
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
              {isHighlighted && (
                <div
                  className="absolute rounded-full border-4 border-danger animate-symbol-pop"
                  style={{
                    width: p.size + 12,
                    height: p.size + 12,
                    top: -6,
                    left: -6,
                  }}
                />
              )}
              {isWrong && (
                <div
                  className="absolute rounded-full border-4 border-gray-900"
                  style={{
                    width: p.size + 12,
                    height: p.size + 12,
                    top: -6,
                    left: -6,
                  }}
                />
              )}
              {toIcon(num)}
            </button>
          );
        })}
      </div>
    );
  };

  if (props.status !== "STARTED") {
    return (
      <div className="flex items-center justify-center min-h-screen text-text-muted text-lg">
        ゲーム開始を待っています...
      </div>
    );
  }

  // scores が空の場合、自分のプレイヤー情報からフォールバック
  // 自分を先頭に、それ以外はIDでソート
  const rawScores = props.scores.length > 0
    ? props.scores
    : props.player
      ? [{ player_id: props.player.id, score: props.player.score, name: props.player.name }]
      : [];
  const displayScores = [
    ...rawScores.filter(s => s.player_id === props.player?.id),
    ...rawScores.filter(s => s.player_id !== props.player?.id).sort((a, b) => a.player_id - b.player_id),
  ];
  // 回答待ち: 場のカード=1つ前、めくったカード=最新
  // それ以外: めくったカードが場に移動し、次のカード待ち
  // 回答中 or 結果表示中は両カード維持、カードを引くまで移動しない
  const isAnswering = (props.dealACard === NEED_ANSWER || showResult) && props.cards && props.cards.length >= 2;
  const fieldCard = isAnswering
    ? props.cards![props.cards!.length - 2]
    : props.cards && props.cards.length >= 1
      ? props.cards[props.cards.length - 1]
      : null;
  const drawnCard = isAnswering
    ? props.cards![props.cards!.length - 1]
    : null;

  return (
    <div className="min-h-screen bg-bg flex flex-col">
      {/* 回答結果テキスト */}
      {showResult && props.answer && props.player && (
        <div className="fixed top-4 left-1/2 -translate-x-1/2 z-20">
          <div
            className={`px-6 py-2 rounded-xl shadow-lg font-bold text-lg text-white ${
              props.answer.isCorrect ? "bg-success" : "bg-danger"
            }`}
          >
            {props.answer.playerId === props.player.id
              ? props.answer.isCorrect
                ? "正解！"
                : "不正解..."
              : `${displayScores.find(s => s.player_id === props.answer!.playerId)?.name || "相手"} が${
                  props.answer.isCorrect ? "正解" : "不正解"
                }`}
          </div>
        </div>
      )}

      {/* メインエリア（画面中央に配置） */}
      <div className="flex-1 flex flex-col items-center justify-center px-4 py-8">
        {/* カード表示エリア: 左=場、右=探索対象 */}
        <div className="flex flex-row items-center justify-center gap-8 md:gap-12 mb-8">
          {/* 左: 場のカード */}
          <div className="flex flex-col items-center gap-2">
            <span className="text-sm font-semibold text-text-muted">場のカード</span>
            {fieldCard ? (
              <div className={`w-40 h-40 sm:w-56 sm:h-56 md:w-72 md:h-72 lg:w-80 lg:h-80 rounded-full bg-card border-4 border-gray-300 shadow-lg relative overflow-hidden ${
                showResult && props.answer
                  ? props.answer.isCorrect ? "animate-glow" : "animate-shake"
                  : ""
              }`}>
                <SymbolRandomLayout
                  symbols={extractSymbolNumbers(fieldCard.text)}
                  cardId={fieldCard.id}
                  highlightSymbol={showResult && props.answer ? props.answer.answer : undefined}
                  wrongSymbol={showResult && props.answer && !props.answer.isCorrect ? props.answer.userAnswer : undefined}
                />
              </div>
            ) : (
              <div className="w-40 h-40 sm:w-56 sm:h-56 md:w-72 md:h-72 lg:w-80 lg:h-80 rounded-full bg-gray-50 border-4 border-dashed border-gray-300 flex items-center justify-center">
                <span className="text-text-muted text-xs sm:text-sm">カードを引いてください</span>
              </div>
            )}
          </div>

          {/* 右: 探索対象のカード */}
          <div className="flex flex-col items-center gap-2">
            <span className="text-sm font-semibold text-primary">探すカード</span>
            <div className="relative w-40 h-40 sm:w-56 sm:h-56 md:w-72 md:h-72 lg:w-80 lg:h-80">
              {/* 空枠（常に背面に表示） */}
              <div className="absolute inset-0 rounded-full bg-gray-50 border-4 border-dashed border-gray-200 flex items-center justify-center">
                {props.countdown > 0 ? (
                  <span className="text-6xl font-bold text-primary animate-bounce">
                    {props.countdown}
                  </span>
                ) : (
                  <span className="text-text-muted text-xs sm:text-sm text-center px-4">
                    {fieldCard ? "カードを引いてください" : ""}
                  </span>
                )}
              </div>
              {/* カード本体（スライドアニメーション対象） */}
              {drawnCard && (
                <div className={`absolute inset-0 rounded-full bg-card border-4 border-primary/40 shadow-xl overflow-hidden transition-transform duration-500 ease-in-out ${
                  isMoving ? "-translate-x-[calc(100%+2rem)] md:-translate-x-[calc(100%+3rem)]" : ""
                } ${
                  showResult && props.answer
                    ? props.answer.isCorrect ? "animate-glow" : "animate-shake"
                    : ""
                }`}>
                  <SymbolRandomLayout
                    symbols={extractSymbolNumbers(drawnCard.text)}
                    cardId={drawnCard.id}
                    highlightSymbol={showResult && props.answer ? props.answer.answer : undefined}
                    wrongSymbol={showResult && props.answer && !props.answer.isCorrect ? props.answer.userAnswer : undefined}
                  />
                </div>
              )}
            </div>
          </div>
        </div>

        {/* カードを要求するボタン */}
        <button
          onClick={handleReadyClick}
          disabled={props.dealACard !== DEAL_A_CARD}
          className="px-10 py-3.5 bg-accent text-white font-bold rounded-xl text-lg
                     hover:brightness-110 transition shadow-lg
                     disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {props.dealACard}
        </button>

        {/* スコアボード */}
        <div className="mt-6 bg-card rounded-2xl shadow-lg p-4 border border-gray-100 w-full max-w-2xl">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-xs font-bold text-text-muted uppercase tracking-wide">
              スコアボード
            </h3>
            <span className="text-xs text-text-muted">
              第{props.roundResults.length}回{props.totalRounds > 0 ? ` / ${props.totalRounds}回` : ""}
            </span>
          </div>
          {displayScores.length === 0 ? (
            <div className="text-sm text-text-muted">スコア情報なし</div>
          ) : (
            <div className="space-y-2">
              {displayScores.map((s) => {
                const isMe = s.player_id === props.player?.id;
                const marks = props.roundResults.map((r) => {
                  if (r.playerId === s.player_id) {
                    return r.isCorrect ? "○" : "×";
                  }
                  if (r.isCorrect) return "-";
                  return "";
                }).filter(Boolean);
                return (
                  <div key={s.player_id} className="flex items-center gap-3">
                    <span className={`text-sm font-semibold min-w-20 shrink-0 ${isMe ? "text-primary" : "text-text-muted"}`}>
                      {isMe ? "あなた" : (s.name || "相手")}
                    </span>
                    <span className={`text-sm font-bold min-w-12 shrink-0 ${isMe ? "text-primary" : "text-text"}`}>
                      {s.score}点
                    </span>
                    <div className="flex flex-wrap gap-0.5">
                      {marks.map((mark, i) => (
                        <span
                          key={i}
                          className={`w-5 h-5 flex items-center justify-center text-xs font-bold rounded ${
                            mark === "○"
                              ? "bg-success/15 text-success"
                              : mark === "×"
                              ? "bg-danger/15 text-danger"
                              : "bg-gray-100 text-text-muted"
                          }`}
                        >
                          {mark}
                        </span>
                      ))}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default GameComponent;
