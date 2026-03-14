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

    return (
      <div ref={containerRef} className="relative w-full h-full">
        {symbols.map((num, idx) => {
          const p = positions.placed[idx];
          if (!p) return null;
          return (
            <button
              key={idx}
              className="absolute flex items-center justify-center cursor-pointer
                         hover:scale-125 transition-transform duration-150 disabled:opacity-40
                         select-none p-0 bg-transparent border-none"
              style={{
                top: p.cy - p.size / 2,
                left: p.cx - p.size / 2,
                width: p.size,
                height: p.size,
                fontSize: `${p.size * 0.75}px`,
                transform: `rotate(${positions.rotations[idx]}deg)`,
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

  const myScore = props.scores.find(
    (s) => s.player_id === props.player?.id
  )?.score;
  const otherScores = props.scores.filter(
    (s) => s.player_id !== props.player?.id
  );

  return (
    <div className="min-h-screen bg-bg p-4">
      {/* スコアボード（右上固定） */}
      <div className="fixed top-4 right-4 bg-card rounded-2xl shadow-lg p-4 min-w-44 z-10 border border-gray-100">
        <h3 className="text-xs font-bold text-text-muted uppercase tracking-wide mb-3">
          スコアボード
        </h3>
        {myScore !== undefined && (
          <div className="text-xl font-bold text-primary mb-2">
            あなた: {myScore}点
          </div>
        )}
        {otherScores.map((s) => (
          <div key={s.player_id} className="text-sm text-text-muted">
            プレイヤー {s.player_id}: {s.score}点
          </div>
        ))}
        {props.scores.length === 0 && (
          <div className="text-sm text-text-muted">スコア情報なし</div>
        )}
      </div>

      {/* 回答ダイアログ */}
      {showDialog && props.answer && props.player && (
        <>
          <div
            className="fixed inset-0 bg-black/30 z-40"
            onClick={() => setShowDialog(false)}
          />
          <div className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 z-50 bg-card rounded-2xl shadow-2xl p-8 min-w-80 text-center border border-gray-100">
            <button
              onClick={() => setShowDialog(false)}
              className="absolute top-3 right-4 text-text-muted hover:text-text text-2xl bg-transparent border-none cursor-pointer"
              aria-label="閉じる"
            >
              &times;
            </button>

            <div
              className={`text-2xl font-bold mb-4 ${
                props.answer.isCorrect ? "text-success" : "text-danger"
              }`}
            >
              {props.answer.playerId === props.player.id
                ? props.answer.isCorrect
                  ? "正解！"
                  : "不正解..."
                : `プレイヤー ${props.answer.playerId} が${
                    props.answer.isCorrect ? "正解" : "不正解"
                  }`}
            </div>

            <div className="flex justify-center gap-8 mb-6">
              <div>
                <div className="text-xs text-text-muted mb-1">正解</div>
                <div className="text-4xl">
                  {toIcon(String(props.answer.answer))}
                </div>
              </div>
              <div>
                <div className="text-xs text-text-muted mb-1">選択</div>
                <div className="text-4xl">
                  {toIcon(String(props.answer.userAnswer))}
                </div>
              </div>
            </div>

            {/* ダイアログ内のカード表示 */}
            <div className="flex flex-col gap-4 items-center">
              {props.cards &&
                props.cards
                  .slice(-2)
                  .reverse()
                  .map((card) => {
                    const symbolNums = extractSymbolNumbers(card.text);
                    return (
                      <div key={card.id} className="flex gap-2 flex-wrap justify-center">
                        {symbolNums.map((num, sidx) => {
                          const isCorrect =
                            String(num) === String(props.answer!.answer);
                          return (
                            <div
                              key={sidx}
                              className={`rounded-lg px-3 py-1.5 text-lg font-bold ${
                                isCorrect
                                  ? "bg-success text-white"
                                  : "bg-gray-100 text-text"
                              }`}
                            >
                              {toIcon(num)}
                            </div>
                          );
                        })}
                      </div>
                    );
                  })}
            </div>
          </div>
        </>
      )}

      {/* メインエリア */}
      <div className="flex flex-col items-center pt-8">
        {/* カードを要求するボタン */}
        <button
          onClick={handleReadyClick}
          disabled={props.dealACard !== DEAL_A_CARD}
          className="mb-8 px-8 py-3 bg-accent text-white font-bold rounded-xl text-lg
                     hover:brightness-110 transition shadow-lg
                     disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {props.dealACard}
        </button>

        {/* カード表示エリア */}
        <div className="flex flex-col md:flex-row gap-8 items-center justify-center">
          {props.cards &&
            [...props.cards]
              .reverse()
              .slice(0, 2)
              .map((card) => {
                const symbolNums = extractSymbolNumbers(card.text);
                return (
                  <div
                    key={card.id}
                    className="w-56 h-56 md:w-64 md:h-64 rounded-full bg-card border-4 border-gray-200 shadow-xl relative overflow-hidden"
                  >
                    <SymbolRandomLayout
                      symbols={symbolNums}
                      cardId={card.id}
                    />
                  </div>
                );
              })}
        </div>
      </div>
    </div>
  );
};

export default GameComponent;
