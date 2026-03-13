/**
 * Client-only module — MUST NOT be imported in server-rendered components.
 * WebSocket client with auth, subscribe/unsubscribe, and reconnect with backoff.
 */
import { signal } from "@preact/signals";
import { getAccessToken } from "@/lib/api.ts";

export type WSStatus = "connecting" | "connected" | "disconnected";
export const wsStatus = signal<WSStatus>("disconnected");

type EventCallback = (eventType: string, object: unknown) => void;

interface Subscription {
  id: string;
  kind: string;
  namespace: string;
  onEvent: EventCallback;
}

let ws: WebSocket | null = null;
let reconnectAttempt = 0;
let reconnectTimer: number | null = null;
const subscriptions = new Map<string, Subscription>();
let authenticated = false;

const BASE_DELAY = 1000;
const MAX_DELAY = 30000;
const JITTER = 0.2;

function getWsUrl(): string {
  const proto = globalThis.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${globalThis.location.host}/ws/v1/ws/resources`;
}

function reconnectDelay(): number {
  const delay = Math.min(BASE_DELAY * Math.pow(2, reconnectAttempt), MAX_DELAY);
  const jitter = delay * JITTER * (Math.random() * 2 - 1);
  return delay + jitter;
}

export function connectWS(): void {
  if (
    ws &&
    (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)
  ) {
    return;
  }

  wsStatus.value = "connecting";
  authenticated = false;

  try {
    ws = new WebSocket(getWsUrl());
  } catch {
    scheduleReconnect();
    return;
  }

  ws.onopen = () => {
    const token = getAccessToken();
    if (token) {
      ws!.send(JSON.stringify({ type: "auth", token }));
    } else {
      ws!.close();
      wsStatus.value = "disconnected";
    }
  };

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data);
      handleMessage(msg);
    } catch {
      // ignore unparseable messages
    }
  };

  ws.onclose = () => {
    ws = null;
    authenticated = false;
    wsStatus.value = "disconnected";
    scheduleReconnect();
  };

  ws.onerror = () => {
    // onclose will fire after onerror
  };
}

function handleMessage(
  msg: {
    type: string;
    id?: string;
    eventType?: string;
    object?: unknown;
    code?: number;
    message?: string;
  },
) {
  switch (msg.type) {
    case "auth_ok":
      authenticated = true;
      wsStatus.value = "connected";
      reconnectAttempt = 0;
      // Re-subscribe all active subscriptions
      for (const sub of subscriptions.values()) {
        sendSubscribe(sub);
      }
      break;

    case "event":
      if (msg.id) {
        const sub = subscriptions.get(msg.id);
        if (sub && msg.eventType && msg.object !== undefined) {
          sub.onEvent(msg.eventType, msg.object);
        }
      }
      break;

    case "error":
      // For RBAC errors (403), notify the subscriber
      if (msg.id && msg.code === 403) {
        const sub = subscriptions.get(msg.id);
        if (sub) {
          sub.onEvent("RBAC_DENIED", null);
        }
      }
      break;

    case "subscribed":
      // Subscription confirmed
      break;
  }
}

function sendSubscribe(sub: Subscription): void {
  if (ws && ws.readyState === WebSocket.OPEN && authenticated) {
    ws.send(JSON.stringify({
      type: "subscribe",
      id: sub.id,
      kind: sub.kind,
      namespace: sub.namespace,
    }));
  }
}

function scheduleReconnect(): void {
  if (reconnectTimer !== null) return;

  // Visibility-aware: don't reconnect if tab is hidden
  if (typeof document !== "undefined" && document.hidden) {
    const onVisible = () => {
      document.removeEventListener("visibilitychange", onVisible);
      scheduleReconnect();
    };
    document.addEventListener("visibilitychange", onVisible);
    return;
  }

  const delay = reconnectDelay();
  reconnectAttempt++;
  reconnectTimer = globalThis.setTimeout(() => {
    reconnectTimer = null;
    connectWS();
  }, delay) as unknown as number;
}

/**
 * Subscribe to resource events for a given kind and namespace.
 * Returns an unsubscribe function.
 */
export function subscribe(
  id: string,
  kind: string,
  namespace: string,
  onEvent: EventCallback,
): () => void {
  const sub: Subscription = { id, kind, namespace, onEvent };
  subscriptions.set(id, sub);

  // Connect WS if not already connected
  connectWS();

  // Send subscribe if already authenticated
  sendSubscribe(sub);

  return () => {
    subscriptions.delete(id);
    if (ws && ws.readyState === WebSocket.OPEN && authenticated) {
      ws.send(JSON.stringify({ type: "unsubscribe", id }));
    }
  };
}

/** Disconnect and clean up the WebSocket connection. */
export function disconnectWS(): void {
  if (reconnectTimer !== null) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  reconnectAttempt = 0;
  subscriptions.clear();
  if (ws) {
    ws.close();
    ws = null;
  }
  wsStatus.value = "disconnected";
}
