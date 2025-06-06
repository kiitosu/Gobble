import type { Player } from "../gen/game/v1/game_pb";

import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { ReportReadyService, SubmitAnswerService } from "../gen/game/v1/game_pb";
import { useWebSocket } from "./WebSocketContext";
import React, { useEffect, useState } from "react";

type GameProps = {
    message: string;
    player?: Player;
    status?: string;
};

const GameComponent = (props: GameProps) => {
    const transport = createConnectTransport({
        baseUrl: "http://localhost:8080"
    });

    const reportReadyServiceclient = createClient(ReportReadyService, transport);
    const submitAnswerServiceClient = createClient(SubmitAnswerService, transport);

    const ws = useWebSocket();
    const [cards, setCards] = useState<{ id: number; text: string }[]>([]);

    useEffect(() => {
        if (!ws) return;
        ws.onmessage = (e) => {
            try {
                const msg = JSON.parse(e.data);
                if (msg.event === "card" && msg.card) {
                    setCards(prev => [...prev, msg.card]);
                }
            } catch {
                // ignore parse error
            }
        };
        return () => { ws.onmessage = null; };
    }, [ws]);


    const handleReadyClick = async () => {
        if (props.player) {
            await reportReadyServiceclient.reportReady({
                playerId: String(props.player.id)
            });
        }
    };

    const handleSubmitAnswer = async () => {
        if (props.player) {
            await submitAnswerServiceClient.submitAnswer({
                playerId: String(props.player.id),
                answer: "test answer"
            });
        }
    };

    return (
        <>
            {props.status == "STARTED" && cards.length > 0 && (
                <div>
                    <h3>受信カード一覧</h3>
                    {cards.map(card => (
                        <div key={card.id}>
                            カードID: {card.id} 内容: {card.text}
                        </div>
                    ))}
                </div>
            )}
            {props.status == "STARTED" && props.player && (
                <>
                    <button onClick={handleReadyClick}>
                        I'm READY!!!
                    </button>
                </>
            )}
            {props.status == "JOINED" && (
                <>
                    Waiting for game to start...
                </>
            )}
        </>
    );
};
export default GameComponent;
