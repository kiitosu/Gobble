import type { Player } from "../gen/game/v1/game_pb";

import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { ReportReadyService, SubmitAnswerService } from "../gen/game/v1/game_pb";

type GameProps = {
    message: string;
    started: boolean
    player?: Player;
    status?: string;
    cards?: { id: number; text: string }[]
};

const GameComponent = (props: GameProps) => {
    const transport = createConnectTransport({
        baseUrl: "http://localhost:8080"
    });

    const reportReadyServiceclient = createClient(ReportReadyService, transport);
    const submitAnswerServiceClient = createClient(SubmitAnswerService, transport);

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
            {props.status == "STARTED" && props.cards && props.cards.length > 0 && (
                <div>
                    <h3>受信カード一覧</h3>
                    {props.cards && props.cards.map((card, index) => (
                        <div key={`${card.id}-${index}`}>
                            カードID: {card.id} 内容: {card.text}
                        </div>
                    ))}
                </div>
            )}
            {props.status == "STARTED" && props.player && props.started && (
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
