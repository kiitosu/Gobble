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
