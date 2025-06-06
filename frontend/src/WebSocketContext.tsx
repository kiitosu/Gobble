import React, { createContext, useContext } from "react";

export const WebSocketContext = createContext<WebSocket | null>(null);

export const useWebSocket = () => useContext(WebSocketContext);

type WebSocketProviderProps = {
  value: WebSocket | null;
  children: React.ReactNode;
};

export const WebSocketProvider: React.FC<WebSocketProviderProps> = ({ value, children }) => (
  <WebSocketContext.Provider value={value}>
    {children}
  </WebSocketContext.Provider>
);
