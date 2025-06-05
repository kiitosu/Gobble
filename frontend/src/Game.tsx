import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { Request } from "../gen/game/v1/game_pb";

type GameProps = {
    message: string;
};

const Game = ({ message }: GameProps) => {

    const transport = createConnectTransport({
        baseUrl: "http://localhost:8080"
    });
    const createGameClient = createClient(CreateGameService, transport);


    return (
        <>
            <div>サーバーからのメッセージ: {message}</div>
        </>
    );
};
export default Game
