import { type ReactNode } from "react";
import { Modal } from "./modal";
import { Button } from "./button";

type ConfirmationDialogProps = {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  /** Description of what will happen */
  children: ReactNode;
  /** Label for the confirm button */
  confirmLabel?: string;
  /** Whether the action is destructive (red button) */
  destructive?: boolean;
  loading?: boolean;
};

export function ConfirmationDialog({
  open,
  onClose,
  onConfirm,
  title,
  children,
  confirmLabel = "Confirm",
  destructive = false,
  loading = false,
}: ConfirmationDialogProps) {
  return (
    <Modal open={open} onClose={onClose} title={title}>
      <div className="flex flex-col gap-6">
        <div>
          <h2 className="type-title-lg text-on-surface">{title}</h2>
          <div className="mt-2 type-body-md text-on-surface-variant">
            {children}
          </div>
        </div>

        <div className="flex justify-end gap-3">
          <Button variant="tertiary" onClick={onClose} disabled={loading}>
            Cancel
          </Button>
          <Button
            variant={destructive ? "primary" : "primary"}
            onClick={onConfirm}
            loading={loading}
            className={destructive ? "bg-error hover:bg-error/90" : ""}
          >
            {confirmLabel}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
