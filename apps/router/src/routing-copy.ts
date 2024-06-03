import { IncomingMessage, Server } from "http";
import WebSocket, { RawData, WebSocketServer } from "ws";

import { isEventShellCreate, isEventShellData } from "./events";

const terminalClients: Record<string, WebSocket> = {};
const instanceClients: Record<string, WebSocket> = {};

const onInstanceClose = (id: string) => () => {
  delete instanceClients[id];
};

/**
 * Return all shell outputs to the terminal.
 */
const onInstanceMessage = (data: RawData) => {
  const event = JSON.parse(data.toString());
  if (!isEventShellData(event)) return;

  const terminal = terminalClients[event.terminalId];
  terminal?.send(event.data);
};

const onTerminalClose = (id: string) => () => {
  delete terminalClients[id];
};

/**
 * Redirect all terminal inputs and shell creation to the instance.
 */
const onTerminalMessage = (data: RawData) => {
  const event = JSON.parse(data.toString());
  if (!isEventShellCreate(event) && !isEventShellData(event)) return;

  const instance = instanceClients[event.instanceId];
  instance?.send(JSON.stringify(event));
};

const onConnect = (ws: WebSocket, request: IncomingMessage) => {
  const { headers } = request;

  const instanceId = headers["x-instance-id"]?.toString();
  if (instanceId != null) {
    instanceClients[instanceId] = ws;
    ws.on("message", onInstanceMessage);
    ws.on("close", onInstanceClose(instanceId));
    return;
  }

  const terminalId = headers["x-terminal-id"]?.toString();
  if (terminalId != null) {
    terminalClients[terminalId] = ws;
    ws.on("message", onTerminalMessage);
    ws.on("close", onTerminalClose(terminalId));
    return;
  }

  ws.close();
};

export const addSocket = (server: Server) => {
  const wss = new WebSocketServer({ server });
  wss.on("connection", onConnect);

  server.on("upgrade", (request, socket, head) => {
    wss.handleUpgrade(request, socket, head, (ws) => {
      wss.emit("connection", ws, request);
    });
  });
};
