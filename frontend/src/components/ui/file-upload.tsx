import {
  useRef,
  useState,
  useCallback,
  type DragEvent,
  type ChangeEvent,
} from "react";
import { Upload } from "lucide-react";
import { Icon } from "./icon";

type FileUploadProps = {
  /** Accepted MIME types (e.g., "image/*,.pdf") */
  accept?: string;
  /** Maximum file size in bytes */
  maxSize?: number;
  /** Allow multiple files */
  multiple?: boolean;
  /** Called with selected files */
  onFiles: (files: File[]) => void;
  /** Upload progress (0–100) */
  progress?: number;
  /** Error message */
  error?: string;
  className?: string;
};

export function FileUpload({
  accept,
  maxSize,
  multiple = false,
  onFiles,
  progress,
  error,
  className = "",
}: FileUploadProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [isDragOver, setIsDragOver] = useState(false);
  const [localError, setLocalError] = useState<string | null>(null);

  const displayError = error ?? localError;

  const validateAndEmit = useCallback(
    (files: FileList | null) => {
      setLocalError(null);
      if (!files || files.length === 0) return;

      const fileArray = Array.from(files);

      if (maxSize) {
        const oversized = fileArray.find((f) => f.size > maxSize);
        if (oversized) {
          const maxMB = (maxSize / (1024 * 1024)).toFixed(1);
          setLocalError(`File too large. Maximum size is ${maxMB} MB.`);
          return;
        }
      }

      onFiles(fileArray);
    },
    [maxSize, onFiles],
  );

  const handleDrop = useCallback(
    (e: DragEvent) => {
      e.preventDefault();
      setIsDragOver(false);
      validateAndEmit(e.dataTransfer.files);
    },
    [validateAndEmit],
  );

  const handleChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      validateAndEmit(e.target.files);
      // Reset so the same file can be re-selected
      e.target.value = "";
    },
    [validateAndEmit],
  );

  return (
    <div className={`flex flex-col gap-1.5 ${className}`}>
      <button
        type="button"
        onClick={() => inputRef.current?.click()}
        onDragOver={(e) => {
          e.preventDefault();
          setIsDragOver(true);
        }}
        onDragLeave={() => setIsDragOver(false)}
        onDrop={handleDrop}
        className={`flex flex-col items-center justify-center gap-3 rounded-lg border-2 border-dashed px-6 py-10 transition-colors ${
          isDragOver
            ? "border-primary bg-primary/5"
            : "border-outline-variant bg-surface-container-low hover:bg-surface-container"
        } ${displayError ? "border-error" : ""}`}
      >
        <Icon icon={Upload} size="lg" className="text-on-surface-variant" />
        <div className="text-center">
          <p className="type-body-md text-on-surface">
            Drag & drop or{" "}
            <span className="text-primary font-medium">browse</span>
          </p>
          {accept && (
            <p className="type-body-sm text-on-surface-variant mt-1">
              Accepted formats: {accept}
            </p>
          )}
        </div>
      </button>

      <input
        ref={inputRef}
        type="file"
        accept={accept}
        multiple={multiple}
        onChange={handleChange}
        className="sr-only"
        tabIndex={-1}
        aria-hidden="true"
      />

      {progress !== undefined && progress > 0 && progress < 100 && (
        <div className="h-1.5 w-full overflow-hidden rounded-full bg-surface-container-high">
          <div
            className="h-full rounded-full bg-primary transition-all duration-[var(--duration-normal)]"
            style={{ width: `${String(progress)}%` }}
            role="progressbar"
            aria-valuenow={progress}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-label="Upload progress"
          />
        </div>
      )}

      {displayError && (
        <p className="type-body-sm text-error" role="alert">
          {displayError}
        </p>
      )}
    </div>
  );
}
