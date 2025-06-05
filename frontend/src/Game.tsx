import type { Player } from "../gen/game/v1/game_pb";

import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { ReportReadyService } from "../gen/game/v1/game_pb";

type GameProps = {
    message: string;
    player?: Player;
};

const Game = (props: GameProps) => {
    const transport = createConnectTransport({
        baseUrl: "http://localhost:8080"
    });

    const reportReadyServiceclient = createClient(ReportReadyService, transport);
    
    const handleReadyClick = async () => {
        if (props.player) {
            await reportReadyServiceclient.reportReady({
                playerId: String(props.player.id)
            });
        }
    };

    return (
        <>
            <div>サーバーからのメッセージ: {props.message}</div>
            {props.player && (
                <button onClick={handleReadyClick}>
                    I'm READY!!!
                </button>
            )}
        </>
    );
};
export default Game
