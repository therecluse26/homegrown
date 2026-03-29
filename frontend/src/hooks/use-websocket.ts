import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useIntl } from "react-intl";
import {
  wsConnect,
  wsDisconnect,
  subscribe,
  type WsMessage,
} from "@/lib/websocket";
import { useToast } from "@/components/ui/toast";

/**
 * Hook that manages the WebSocket connection lifecycle and dispatches
 * TanStack Query invalidations when real-time messages arrive.
 *
 * Also shows celebration toasts for streak/learning milestone events.
 *
 * Connects on mount, disconnects on unmount. Only used inside the
 * authenticated AppShell layout — not mounted on unauthenticated pages.
 *
 * @see ARCHITECTURE §11.5 (WebSocket strategy)
 */
export function useWebSocket() {
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const intl = useIntl();

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

        case "streak_milestone": {
          const payload = message.payload as {
            days?: number;
            student_name?: string;
          };
          const days = payload.days ?? 0;
          const name = payload.student_name ?? "";
          toast(
            intl.formatMessage(
              { id: "milestone.streak" },
              { days, name },
            ),
            "success",
          );
          // Also refresh progress data
          void queryClient.invalidateQueries({
            queryKey: ["learning", "progress"],
          });
          void queryClient.invalidateQueries({
            queryKey: ["learning", "streak"],
          });
          break;
        }

        case "learning_milestone": {
          const payload = message.payload as {
            milestone_type?: string;
            student_name?: string;
          };
          const milestoneType = payload.milestone_type ?? "";
          const name = payload.student_name ?? "";
          toast(
            intl.formatMessage(
              { id: "milestone.learning" },
              { type: milestoneType, name },
            ),
            "success",
          );
          void queryClient.invalidateQueries({
            queryKey: ["learning", "progress"],
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
  }, [queryClient, toast, intl]);
}
