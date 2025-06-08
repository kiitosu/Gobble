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
  answer?: { playerId: number; isCorrect: boolean };
  dealACard: string;
  setDealACard: React.Dispatch<React.SetStateAction<string>>;
  scores: { player_id: number; score: number }[];
};


const GameComponent = (props: GameProps) => {
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
      props.setDealACard(WAIT_FOR_OTHER_PLAYERS)
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
                {props.player && props.scores.find(s => s.player_id === props.player!.id) && (
                  <div style={{ color: "#1976d2" }}>
                    あなたのスコア: {props.scores.find(s => s.player_id === props.player!.id)?.score}点
                  </div>
                )}
                {/* 他のプレイヤーのスコア */}
                {props.scores
                  .filter(score => !props.player || score.player_id !== props.player.id)
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
          <div>
            {props.answer && props.player && (
              <>
                {props.answer.playerId === props.player.id
                  ? `You are ${props.answer.isCorrect ? "correct" : "wrong"}!`
                  : `Player ${props.player.id} ${
                      props.answer.isCorrect ? "correct" : "wrong"
                    }!`}
              </>
            )}
          </div>

          <button
            onClick={handleReadyClick}
            disabled={props.dealACard !== DEAL_A_CARD}
          >
            {props.dealACard}
          </button>

          <h3>受信カード一覧</h3>

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
                        disabled={props.dealACard !== NEED_ANSWER || index !== 0} // 最新のカード（indexが0）のみ有効
                      >
                        {symbol}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            ))}
        </div>
      )}

      {/* ゲーム参加中・開催まち */}
      {props.status == "JOINED" && <>Waiting for game to start...</>}
    </>
  );
};
export default GameComponent;
