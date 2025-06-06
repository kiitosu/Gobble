import type { Player } from "../gen/game/v1/game_pb";

import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { ReportReadyService, SubmitAnswerService } from "../gen/game/v1/game_pb";
import { useWebSocket } from "./WebSocketContext";
import React, { useEffect, useState } from "react";

type GameProps = {
    message: string;
    player?: Player;
};

const Game = (props: GameProps) => {
    const transport = createConnectTransport({
        baseUrl: "http://localhost:8080"
    });

    const reportReadyServiceclient = createClient(ReportReadyService, transport);
    const submitAnswerServiceClient = createClient(SubmitAnswerService, transport);

    const ws = useWebSocket();
    const [wsMessage, setWsMessage] = useState<string>("");

    useEffect(() => {
        if (!ws) return;
        ws.onmessage = (e) => {
            setWsMessage(e.data);
        };
        // クリーンアップ
        return () => {
            ws.onmessage = null;
        };
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
            <div>サーバーからのメッセージ: {props.message}</div>
            <div>WebSocket: {wsMessage}</div>
            {props.player && (
                <>
                    <button onClick={handleReadyClick}>
                        I'm READY!!!
                    </button>
                    <button onClick={handleSubmitAnswer}>
                        回答送信
                    </button>
                </>
            )}
        </>
    );
};
export default Game;
