import { type ReactNode } from "react";
import { ProgressBar, Button } from "@/components/ui";
import { Check } from "lucide-react";
import { Icon } from "@/components/ui";

export type QuizOption = {
  id: string;
  label: string;
  emoji?: string;
};

export type QuizQuestion = {
  id: string;
  text: string;
  options: QuizOption[];
  multiSelect?: boolean;
};

type QuizQuestionScreenProps = {
  variant: "parent" | "student";
  question: QuizQuestion;
  /** 0-based */
  questionIndex: number;
  totalQuestions: number;
  selectedIds: string[];
  studentName?: string;
  onSelect: (optionId: string) => void;
  onSkip: () => void;
  onBack: () => void;
  onNext: () => void;
};

function OptionCard({
  option,
  selected,
  multiSelect,
  isStudent,
  onSelect,
}: {
  option: QuizOption;
  selected: boolean;
  multiSelect: boolean;
  isStudent: boolean;
  onSelect: () => void;
}) {
  const role = multiSelect ? "checkbox" : "radio";

  return (
    <button
      type="button"
      role={role}
      aria-checked={selected}
      onClick={onSelect}
      className={[
        "relative flex w-full flex-col items-center gap-2 rounded-lg p-4 text-center transition-all",
        "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring",
        isStudent ? "rounded-xl py-5" : "",
        selected
          ? "bg-primary-container ring-2 ring-primary text-on-primary-container"
          : "bg-surface-container-lowest text-on-surface hover:bg-surface-container-low hover:shadow-ambient-sm",
      ]
        .filter(Boolean)
        .join(" ")}
    >
      {/* Multi-select check indicator */}
      {multiSelect && (
        <span
          aria-hidden
          className={[
            "absolute right-2 top-2 flex h-5 w-5 items-center justify-center rounded-full border-2 transition-colors",
            selected
              ? "border-primary bg-primary text-on-primary"
              : "border-outline bg-surface-container-lowest",
          ].join(" ")}
        >
          {selected && <Icon icon={Check} size="xs" />}
        </span>
      )}

      {/* Emoji for student variant */}
      {isStudent && option.emoji && (
        <span className="text-3xl leading-none" aria-hidden>
          {option.emoji}
        </span>
      )}

      <span
        className={
          isStudent ? "type-body-md font-medium" : "type-body-md font-medium"
        }
      >
        {option.label}
      </span>
    </button>
  );
}

export function QuizQuestionScreen({
  variant,
  question,
  questionIndex,
  totalQuestions,
  selectedIds,
  studentName,
  onSelect,
  onSkip,
  onBack,
  onNext,
}: QuizQuestionScreenProps): ReactNode {
  const isStudent = variant === "student";
  const questionNumber = questionIndex + 1;
  const progressPercent = Math.round((questionIndex / totalQuestions) * 100);
  const isMulti = question.multiSelect ?? false;
  const hasSelection = selectedIds.length > 0;

  const progressLabel = `Question ${String(questionNumber)} of ${String(totalQuestions)}`;

  return (
    <div data-context={isStudent ? "student" : "parent"}>
      {/* Sub-progress within the learner-profile step */}
      <div className="mb-6">
        <div className="mb-2 flex items-center justify-between">
          <span className="type-label-md text-on-surface-variant">
            {progressLabel}
          </span>
          {isStudent && (
            <span className="type-label-sm text-on-surface-variant">
              Almost there!
            </span>
          )}
        </div>
        <ProgressBar
          value={progressPercent}
          label={progressLabel}
          className={isStudent ? "[&>div]:h-3" : ""}
        />
      </div>

      {/* Question text */}
      <h2
        className={[
          "mb-6 text-on-surface",
          isStudent
            ? "type-headline-sm font-bold text-center"
            : "type-headline-sm font-semibold",
        ].join(" ")}
      >
        {question.text}
      </h2>

      {/* Multi-select helper */}
      {isMulti && (
        <p className="mb-4 type-body-sm text-on-surface-variant">
          {isStudent
            ? "Pick as many as you like!"
            : "Select all that apply."}
        </p>
      )}

      {/* Options grid */}
      <div
        role={isMulti ? "group" : "radiogroup"}
        aria-label={question.text}
        className="mb-6 grid grid-cols-2 gap-3"
      >
        {question.options.map((option) => (
          <OptionCard
            key={option.id}
            option={option}
            selected={selectedIds.includes(option.id)}
            multiSelect={isMulti}
            isStudent={isStudent}
            onSelect={() => onSelect(option.id)}
          />
        ))}
      </div>

      {/* Skip question */}
      <div className={`mb-8 ${isStudent ? "text-center" : ""}`}>
        <button
          type="button"
          onClick={onSkip}
          className="type-body-sm text-on-surface-variant underline-offset-2 hover:text-on-surface hover:underline transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring rounded-sm"
        >
          Skip this question
        </button>
      </div>

      {/* Navigation */}
      <div className="flex items-center gap-3">
        <Button type="button" variant="tertiary" onClick={onBack}>
          Back
        </Button>
        <Button
          type="button"
          variant="primary"
          onClick={onNext}
          disabled={!hasSelection}
          className="flex-1"
        >
          {questionIndex === totalQuestions - 1 ? "See my profile →" : "Next →"}
        </Button>
      </div>

      {/* Context note for parent proxy variant */}
      {!isStudent && studentName && (
        <p className="mt-4 type-body-sm text-on-surface-variant text-center">
          Answering on behalf of {studentName}
        </p>
      )}
    </div>
  );
}
