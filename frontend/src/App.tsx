import { useEffect, useRef, useState } from 'react'
import './App.css'

import Game from "./Game"
import Lobby from "./Lobby";

// src/App.tsx

// 接続したいサービスをインポート
import type { Game as ProtoGame, Player } from "../gen/game/v1/game_pb"

// App.tsx専用の拡張型
type Game = ProtoGame & {
  localStatus?: string; // 追加したいプロパティ
};


function App() {
    const [startedGame, setStartedGame] = useState<Game | null>(null)
    const [gameStatus, setGameStatus] = useState("")
    const [player, setPlayer] = useState<Player>()
    const [message, setMessage] = useState("");
    const ws = useRef<WebSocket|null>(null)

    useEffect(()=>{
        ws.current = new WebSocket('ws://localhost:8080/ws')

        ws.current.onopen = () => {
            console.log('Websocket接続が確立されました')
            if (ws.current?.readyState === WebSocket.OPEN){
                ws.current.send('Hello from client!')
            } else {
                console.log('ws.current is not ready as it is Strict mode.')
            }
        };

        ws.current.onmessage = (event) => {
            console.log("サーバーからのメッセージ", event.data);
            setMessage(event.data)
        }

        ws.current.onclose = () => {
            console.log("Websocket接続が閉じられました")
        }

        ws.current.onerror = (error) => {
            console.error("WebSocketエラー:", error)
        }

        return () => {
            if (ws?.current?.readyState === WebSocket.OPEN) {
                ws.current.close();
            }
        }
    }, []);


    // ロビー画面のUI・ロジックはLobbyコンポーネントに移動

    return (
        <>
            {gameStatus === "STARTED" && player && startedGame ? (
                <Game message={message} />
            ) : (
                <Lobby
                    gameStatus={gameStatus}
                    player={player}
                    setPlayer={setPlayer}
                    setGameStatus={setGameStatus}
                    setStartedGame={setStartedGame}
                />
            )}
        </>
    )
}

export default App
