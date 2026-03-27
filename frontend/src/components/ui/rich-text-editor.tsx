import { useRef, useCallback } from "react";
import { Bold, Italic, List, ListOrdered, Heading2 } from "lucide-react";
import { Icon } from "./icon";

type RichTextEditorProps = {
  /** HTML content */
  value: string;
  /** Called with updated HTML string */
  onChange: (html: string) => void;
  /** Placeholder text */
  placeholder?: string;
  /** Minimum height class */
  minHeight?: string;
  className?: string;
};

type ToolbarAction = {
  icon: typeof Bold;
  label: string;
  command: string;
  argument?: string;
};

const TOOLBAR_ACTIONS: ToolbarAction[] = [
  { icon: Bold, label: "Bold", command: "bold" },
  { icon: Italic, label: "Italic", command: "italic" },
  { icon: Heading2, label: "Heading", command: "formatBlock", argument: "h2" },
  { icon: List, label: "Bulleted list", command: "insertUnorderedList" },
  { icon: ListOrdered, label: "Numbered list", command: "insertOrderedList" },
];

export function RichTextEditor({
  value,
  onChange,
  placeholder = "Start writing...",
  minHeight = "min-h-32",
  className = "",
}: RichTextEditorProps) {
  const editorRef = useRef<HTMLDivElement>(null);

  const execCommand = useCallback((command: string, argument?: string) => {
    document.execCommand(command, false, argument);
    editorRef.current?.focus();
  }, []);

  const handleInput = useCallback(() => {
    if (editorRef.current) {
      onChange(editorRef.current.innerHTML);
    }
  }, [onChange]);

  return (
    <div
      className={`rounded-button bg-surface-container-highest overflow-hidden focus-within:input-focus ${className}`}
    >
      {/* Toolbar */}
      <div
        className="flex gap-0.5 border-b border-surface-container-high p-1"
        role="toolbar"
        aria-label="Text formatting"
      >
        {TOOLBAR_ACTIONS.map((action) => (
          <button
            key={action.command}
            type="button"
            className="rounded-md p-2 text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring transition-colors"
            onClick={() => execCommand(action.command, action.argument)}
            aria-label={action.label}
            tabIndex={-1}
          >
            <Icon icon={action.icon} size="sm" />
          </button>
        ))}
      </div>

      {/* Editable area */}
      <div
        ref={editorRef}
        contentEditable
        role="textbox"
        aria-multiline="true"
        aria-label="Rich text editor"
        className={`p-4 type-body-md text-on-surface outline-none ${minHeight} [&:empty]:before:content-[attr(data-placeholder)] [&:empty]:before:text-on-surface-variant/60`}
        data-placeholder={placeholder}
        dangerouslySetInnerHTML={{ __html: value }}
        onInput={handleInput}
      />
    </div>
  );
}
