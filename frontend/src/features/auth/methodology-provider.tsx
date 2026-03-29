import { createContext, useContext, useMemo, useCallback, type ReactNode } from "react";
import { useMethodology } from "@/hooks/use-methodology";
import type { components } from "@/api/generated/schema";

type ActiveTool = components["schemas"]["method.ActiveToolResponse"];

type MethodologyContextValue = {
  isLoading: boolean;
  primarySlug: string | null;
  primaryName: string | null;
  secondarySlugs: (string | undefined)[];
  terminology: Record<string, never>;
  tools: ActiveTool[];
  /** Look up the methodology-specific label for a tool slug. Falls back to defaultLabel. */
  toolLabel: (slug: string, defaultLabel: string) => string;
};

const MethodologyContext = createContext<MethodologyContextValue | null>(null);

export function MethodologyProvider({ children }: { children: ReactNode }) {
  const methodology = useMethodology();

  // Build a slug→label lookup from the resolved tools
  const toolLabelMap = useMemo(() => {
    const map = new Map<string, string>();
    for (const tool of methodology.tools) {
      if (tool.slug) {
        // Prefer label (methodology-specific override), fall back to display_name
        map.set(tool.slug, tool.label ?? tool.display_name ?? tool.slug);
      }
    }
    return map;
  }, [methodology.tools]);

  const toolLabel = useCallback(
    (slug: string, defaultLabel: string): string =>
      toolLabelMap.get(slug) ?? defaultLabel,
    [toolLabelMap],
  );

  const value = useMemo(
    () => ({ ...methodology, toolLabel }),
    [methodology, toolLabel],
  );

  return (
    <MethodologyContext.Provider value={value}>
      {children}
    </MethodologyContext.Provider>
  );
}

export function useMethodologyContext() {
  const ctx = useContext(MethodologyContext);
  if (!ctx) {
    throw new Error("useMethodologyContext must be used within <MethodologyProvider>");
  }
  return ctx;
}
