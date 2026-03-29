import {
  useState,
  useRef,
  useEffect,
  useCallback,
  type ReactNode,
  type KeyboardEvent,
} from "react";

type DropdownMenuProps = {
  trigger: ReactNode;
  children: ReactNode;
  className?: string;
};

export function DropdownMenu({ trigger, children, className = "" }: DropdownMenuProps) {
  const [open, setOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLDivElement>(null);

  // Close on outside click
  useEffect(() => {
    if (!open) return;

    const handleClick = (e: MouseEvent) => {
      if (
        menuRef.current &&
        !menuRef.current.contains(e.target as Node) &&
        triggerRef.current &&
        !triggerRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    };

    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [open]);

  // Focus first item on open
  useEffect(() => {
    if (!open || !menuRef.current) return;
    const firstItem = menuRef.current.querySelector<HTMLElement>('[role="menuitem"]');
    firstItem?.focus();
  }, [open]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setOpen(false);
        // Return focus to trigger
        const triggerButton = triggerRef.current?.querySelector<HTMLElement>("button");
        triggerButton?.focus();
        return;
      }

      if (!menuRef.current) return;
      const items = Array.from(
        menuRef.current.querySelectorAll<HTMLElement>('[role="menuitem"]'),
      );
      const currentIndex = items.indexOf(document.activeElement as HTMLElement);

      if (e.key === "ArrowDown") {
        e.preventDefault();
        const next = items[(currentIndex + 1) % items.length];
        next?.focus();
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        const prev = items[(currentIndex - 1 + items.length) % items.length];
        prev?.focus();
      }
    },
    [],
  );

  return (
     
    <div className={`relative inline-block ${className}`} onKeyDown={handleKeyDown}>
      <div ref={triggerRef} onClick={() => setOpen((prev) => !prev)}>
        {trigger}
      </div>

      {open && (
        <div
          ref={menuRef}
          role="menu"
          className="absolute right-0 top-full z-popover mt-1 min-w-48 rounded-lg bg-surface-container-lowest py-1 shadow-ambient-md"
        >
          {children}
        </div>
      )}
    </div>
  );
}

type DropdownMenuItemProps = {
  children: ReactNode;
  onClick?: () => void;
  destructive?: boolean;
  className?: string;
};

export function DropdownMenuItem({
  children,
  onClick,
  destructive = false,
  className = "",
}: DropdownMenuItemProps) {
  return (
    <button
      role="menuitem"
      className={`flex w-full items-center gap-3 px-4 py-2.5 type-body-md text-left transition-colors hover:bg-surface-container-low focus-visible:bg-surface-container-low focus-visible:outline-none ${
        destructive ? "text-error" : "text-on-surface"
      } ${className}`}
      onClick={onClick}
      tabIndex={-1}
    >
      {children}
    </button>
  );
}
