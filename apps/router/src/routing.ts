import { IncomingMessage, Server } from "http";
import WebSocket, { RawData, WebSocketServer } from "ws";

import { isEventShellCreate, isEventShellData } from "./events";

const clients: Record<string, WebSocket> = {};
const instanceClients: Record<string, WebSocket> = {};

const onInstanceClose = (id: string) => () => {
  console.log(`Instance disconnected: ${id}`);
  delete instanceClients[id];
};

const onInstanceMessage = (data: RawData) => {
  const event = JSON.parse(data.toString());
  if (!isEventShellData(event)) return;
  const client = clients[event.clientId];
  client?.send(JSON.stringify(event));
};

const onClientClose = (id: string) => () => {
  console.log(`Client disconnected: ${id}`);
  delete clients[id];
};

const onClientMessage = (data: RawData) => {
  const event = JSON.parse(data.toString());
  if (!isEventShellCreate(event) && !isEventShellData(event)) return;
  const instance = instanceClients[event.instanceId];
  instance?.send(JSON.stringify(event));
};

const onConnect = (ws: WebSocket, request: IncomingMessage) => {
  const { headers } = request;

  const instanceId = headers["x-identifier"]?.toString();
  if (instanceId != null) {
    console.log(`Instance connected: ${instanceId}`);
    instanceClients[instanceId] = ws;
    ws.on("message", onInstanceMessage);
    ws.on("close", onInstanceClose(instanceId));
    return;
  }

  // For some reason you cannot set custom headers in the browser
  const clientId = headers["sec-websocket-protocol"]?.toString();
  if (clientId != null) {
    console.log(`Client connected: ${clientId}`);
    clients[clientId] = ws;
    ws.on("message", onClientMessage);
    ws.on("close", onClientClose(clientId));
    return;
  }

  ws.close();
};

export const addSocket = (server: Server) => {
  const wss = new WebSocketServer({ server, path: "/route" });
  wss.on("connection", onConnect);

  // server.on("upgrade", (request, socket, head) => {
  //   wss.handleUpgrade(request, socket, head, (ws) => {
  //     wss.emit("connection", ws, request);
  //   });
  // });
};
