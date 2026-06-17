import {
  createContext,
  useCallback,
  useContext,
  useRef,
  useState,
  type ReactNode,
} from "react";

export type ShortcutEntry = {
  /** Display label shown in the shortcuts modal (e.g. "?", "Esc", "Ctrl+K") */
  key: string;
  /** Human-readable description of what the shortcut does */
  description: string;
};

type ShortcutRegistryContextValue = {
  isOpen: boolean;
  openShortcuts: () => void;
  closeShortcuts: () => void;
  toggleShortcuts: () => void;
  /** Page-level shortcuts registered by the currently mounted route */
  pageShortcuts: ShortcutEntry[];
  /**
   * Register page-specific shortcuts. Call inside a `useEffect` — the returned
   * cleanup function unregisters them when the component unmounts.
   */
  registerPageShortcuts: (shortcuts: ShortcutEntry[]) => () => void;
};

const ShortcutRegistryContext =
  createContext<ShortcutRegistryContextValue | null>(null);

export function useKeyboardShortcutRegistry(): ShortcutRegistryContextValue {
  const ctx = useContext(ShortcutRegistryContext);
  if (!ctx) {
    throw new Error(
      "useKeyboardShortcutRegistry must be used within KeyboardShortcutRegistryProvider",
    );
  }
  return ctx;
}

export function KeyboardShortcutRegistryProvider({
  children,
}: {
  children: ReactNode;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const [pageShortcuts, setPageShortcuts] = useState<ShortcutEntry[]>([]);
  const activeRegistrationRef = useRef(0);

  const openShortcuts = useCallback(() => setIsOpen(true), []);
  const closeShortcuts = useCallback(() => setIsOpen(false), []);
  const toggleShortcuts = useCallback(() => setIsOpen((v) => !v), []);

  const registerPageShortcuts = useCallback(
    (shortcuts: ShortcutEntry[]) => {
      const id = ++activeRegistrationRef.current;
      setPageShortcuts(shortcuts);
      return () => {
        if (activeRegistrationRef.current === id) {
          setPageShortcuts([]);
        }
      };
    },
    [],
  );

  return (
    <ShortcutRegistryContext.Provider
      value={{
        isOpen,
        openShortcuts,
        closeShortcuts,
        toggleShortcuts,
        pageShortcuts,
        registerPageShortcuts,
      }}
    >
      {children}
    </ShortcutRegistryContext.Provider>
  );
}
