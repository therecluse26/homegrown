import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import {
  wsConnect,
  wsDisconnect,
  subscribe,
  type WsMessage,
} from "@/lib/websocket";

/**
 * Hook that manages the WebSocket connection lifecycle and dispatches
 * TanStack Query invalidations when real-time messages arrive.
 *
 * Connects on mount, disconnects on unmount. Only used inside the
 * authenticated AppShell layout — not mounted on unauthenticated pages.
 *
 * @see ARCHITECTURE §11.5 (WebSocket strategy)
 */
export function useWebSocket() {
  const queryClient = useQueryClient();

  useEffect(() => {
    wsConnect();

    const unsubscribe = subscribe((message: WsMessage) => {
      switch (message.type) {
        case "new_message": {
          const payload = message.payload as { conversation_id?: string };
          if (payload.conversation_id) {
            void queryClient.invalidateQueries({
              queryKey: ["messages", payload.conversation_id],
            });
          }
          void queryClient.invalidateQueries({
            queryKey: ["messages"],
          });
          break;
        }

        case "notification": {
          void queryClient.invalidateQueries({
            queryKey: ["notifications"],
          });
          break;
        }

        case "friend_request": {
          void queryClient.invalidateQueries({
            queryKey: ["friends", "requests"],
          });
          void queryClient.invalidateQueries({
            queryKey: ["friends"],
          });
          break;
        }

        default:
          break;
      }
    });

    return () => {
      unsubscribe();
      wsDisconnect();
    };
  }, [queryClient]);
}
