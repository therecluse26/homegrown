/**
 * WebSocket connection manager with automatic reconnection.
 *
 * Connects to /v1/social/ws (proxied via Vite in dev).
 * Supports message types: new_message, notification, friend_request.
 *
 * @see ARCHITECTURE §11.5 (WebSocket strategy)
 */

export type WsMessageType =
  | "new_message"
  | "notification"
  | "friend_request"
  | "ping"
  | "pong";

export interface WsMessage {
  type: WsMessageType;
  payload: unknown;
}

export type WsMessageHandler = (message: WsMessage) => void;

const WS_BASE =
  import.meta.env["VITE_WS_BASE_URL"] ??
  (typeof window !== "undefined"
    ? window.location.origin.replace(/^http/, "ws")
    : "ws://localhost:5673");

const INITIAL_RECONNECT_DELAY_MS = 1000;
const MAX_RECONNECT_DELAY_MS = 30_000;
const RECONNECT_BACKOFF_FACTOR = 2;

let socket: WebSocket | null = null;
let reconnectDelay = INITIAL_RECONNECT_DELAY_MS;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let isManuallyDisconnected = false;

const handlers = new Set<WsMessageHandler>();

function clearReconnectTimer() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
}

function scheduleReconnect() {
  if (isManuallyDisconnected || reconnectTimer) return;

  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    if (!isManuallyDisconnected) {
      connect();
    }
  }, reconnectDelay);

  // Exponential backoff capped at MAX_RECONNECT_DELAY_MS
  reconnectDelay = Math.min(
    reconnectDelay * RECONNECT_BACKOFF_FACTOR,
    MAX_RECONNECT_DELAY_MS,
  );
}

function connect() {
  if (
    socket &&
    (socket.readyState === WebSocket.OPEN ||
      socket.readyState === WebSocket.CONNECTING)
  ) {
    return;
  }

  socket = new WebSocket(`${WS_BASE}/v1/social/ws`);

  socket.addEventListener("open", () => {
    reconnectDelay = INITIAL_RECONNECT_DELAY_MS;
  });

  socket.addEventListener("message", (event: MessageEvent<string>) => {
    try {
      const message = JSON.parse(event.data) as WsMessage;
      handlers.forEach((h) => h(message));
    } catch {
      // Ignore malformed messages
    }
  });

  socket.addEventListener("close", () => {
    socket = null;
    scheduleReconnect();
  });

  socket.addEventListener("error", () => {
    socket?.close();
    socket = null;
  });
}

/** Subscribe to WebSocket messages. Returns an unsubscribe function. */
export function subscribe(handler: WsMessageHandler): () => void {
  handlers.add(handler);
  return () => handlers.delete(handler);
}

/** Connect to the WebSocket server (idempotent). */
export function wsConnect() {
  isManuallyDisconnected = false;
  connect();
}

/** Disconnect and stop reconnecting. */
export function wsDisconnect() {
  isManuallyDisconnected = true;
  clearReconnectTimer();
  socket?.close();
  socket = null;
}

/** Current connection state (useful for UI indicators). */
export function wsReadyState(): number {
  return socket?.readyState ?? WebSocket.CLOSED;
}
