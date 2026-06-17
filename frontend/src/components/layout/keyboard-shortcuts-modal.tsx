import { Keyboard } from "lucide-react";
import { useIntl } from "react-intl";
import { Modal } from "@/components/ui/modal";
import { Icon } from "@/components/ui";
import {
  useKeyboardShortcutRegistry,
  type ShortcutEntry,
} from "@/hooks/use-keyboard-shortcut-registry";

const GLOBAL_SHORTCUTS: ShortcutEntry[] = [
  { key: "?", description: "Open keyboard shortcuts" },
  { key: "/", description: "Focus search bar" },
  { key: "Esc", description: "Close modal / dismiss overlays" },
];

function KbdKey({ value }: { value: string }) {
  return (
    <kbd className="inline-flex items-center justify-center min-w-[1.75rem] px-2 py-0.5 rounded-sm bg-surface-container-high text-on-surface type-label-sm font-mono border border-outline-variant/40 select-none">
      {value}
    </kbd>
  );
}

function ShortcutRow({ shortcut }: { shortcut: ShortcutEntry }) {
  return (
    <div className="flex items-center justify-between gap-4 py-2.5 border-b border-outline-variant/20 last:border-b-0">
      <span className="type-body-sm text-on-surface-variant">
        {shortcut.description}
      </span>
      <KbdKey value={shortcut.key} />
    </div>
  );
}

export function KeyboardShortcutsModal() {
  const intl = useIntl();
  const { isOpen, closeShortcuts, pageShortcuts } =
    useKeyboardShortcutRegistry();

  return (
    <Modal
      open={isOpen}
      onClose={closeShortcuts}
      title={intl.formatMessage({
        id: "shortcuts.modal.title",
        defaultMessage: "Keyboard shortcuts",
      })}
    >
      <div className="flex flex-col gap-6">
        <div className="flex items-center gap-3">
          <Icon icon={Keyboard} size="md" className="text-primary shrink-0" />
          <h2 className="type-title-lg text-on-surface">
            {intl.formatMessage({
              id: "shortcuts.modal.title",
              defaultMessage: "Keyboard shortcuts",
            })}
          </h2>
        </div>

        <section aria-label={intl.formatMessage({ id: "shortcuts.section.global", defaultMessage: "Global" })}>
          <h3 className="type-label-lg text-on-surface-variant mb-2">
            {intl.formatMessage({
              id: "shortcuts.section.global",
              defaultMessage: "Global",
            })}
          </h3>
          <div className="rounded-md bg-surface-container px-3">
            {GLOBAL_SHORTCUTS.map((s) => (
              <ShortcutRow key={s.key} shortcut={s} />
            ))}
          </div>
        </section>

        {pageShortcuts.length > 0 && (
          <section aria-label={intl.formatMessage({ id: "shortcuts.section.page", defaultMessage: "This page" })}>
            <h3 className="type-label-lg text-on-surface-variant mb-2">
              {intl.formatMessage({
                id: "shortcuts.section.page",
                defaultMessage: "This page",
              })}
            </h3>
            <div className="rounded-md bg-surface-container px-3">
              {pageShortcuts.map((s) => (
                <ShortcutRow key={s.key} shortcut={s} />
              ))}
            </div>
          </section>
        )}
      </div>
    </Modal>
  );
}
