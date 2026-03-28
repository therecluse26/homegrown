import { useState, useCallback } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Link as RouterLink } from "react-router";
import {
  ArrowLeft,
  Plus,
  Trash2,
  GripVertical,
  ArrowUp,
  ArrowDown,
} from "lucide-react";
import {
  Button,
  Card,
  Icon,
  Input,
  FormField,
  Badge,
} from "@/components/ui";
import { PageTitle } from "@/components/common/page-title";

// ─── Types ───────────────────────────────────────────────────────────────────

type QuestionType =
  | "multiple_choice"
  | "true_false"
  | "fill_blank"
  | "short_answer"
  | "matching"
  | "ordering";

interface QuizQuestion {
  id: string;
  type: QuestionType;
  text: string;
  options: string[];
  correct_answer: string;
  points: number;
}

const QUESTION_TYPES: { value: QuestionType; label: string }[] = [
  { value: "multiple_choice", label: "Multiple Choice" },
  { value: "true_false", label: "True / False" },
  { value: "fill_blank", label: "Fill in the Blank" },
  { value: "short_answer", label: "Short Answer" },
  { value: "matching", label: "Matching" },
  { value: "ordering", label: "Ordering" },
];

let nextId = 1;
function genId() {
  return `q-${nextId++}`;
}

// ─── Question editor ─────────────────────────────────────────────────────────

function QuestionEditor({
  question,
  index,
  total,
  onChange,
  onRemove,
  onMoveUp,
  onMoveDown,
}: {
  question: QuizQuestion;
  index: number;
  total: number;
  onChange: (q: QuizQuestion) => void;
  onRemove: () => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
}) {
  const intl = useIntl();

  const updateOption = (i: number, value: string) => {
    const opts = [...question.options];
    opts[i] = value;
    onChange({ ...question, options: opts });
  };

  const addOption = () => {
    onChange({ ...question, options: [...question.options, ""] });
  };

  const removeOption = (i: number) => {
    const opts = question.options.filter((_, idx) => idx !== i);
    onChange({ ...question, options: opts });
  };

  return (
    <Card className="p-card-padding">
      <div className="flex items-center gap-2 mb-3">
        <Icon
          icon={GripVertical}
          size="sm"
          className="text-on-surface-variant cursor-grab"
        />
        <Badge variant="secondary">Q{index + 1}</Badge>
        <span className="type-label-sm text-on-surface-variant flex-1">
          {QUESTION_TYPES.find((t) => t.value === question.type)?.label}
        </span>
        <div className="flex gap-1">
          <button
            onClick={onMoveUp}
            disabled={index === 0}
            className="p-1 rounded-radius-sm hover:bg-surface-container-low disabled:opacity-30"
            aria-label={`Move question ${index + 1} up`}
          >
            <Icon icon={ArrowUp} size="xs" />
          </button>
          <button
            onClick={onMoveDown}
            disabled={index === total - 1}
            className="p-1 rounded-radius-sm hover:bg-surface-container-low disabled:opacity-30"
            aria-label={`Move question ${index + 1} down`}
          >
            <Icon icon={ArrowDown} size="xs" />
          </button>
          <button
            onClick={onRemove}
            className="p-1 rounded-radius-sm text-error hover:bg-error-container/30"
            aria-label="Remove question"
          >
            <Icon icon={Trash2} size="xs" />
          </button>
        </div>
      </div>

      <div className="space-y-3">
        <FormField
          label={intl.formatMessage({
            id: "marketplace.quiz.questionText",
          })}
          required
        >
          {({ id }) => (
            <Input
              id={id}
              value={question.text}
              onChange={(e) => onChange({ ...question, text: e.target.value })}
              required
            />
          )}
        </FormField>

        <div className="grid grid-cols-2 gap-3">
          <FormField
            label={intl.formatMessage({
              id: "marketplace.quiz.questionType",
            })}
          >
            {({ id }) => (
              <select
                id={id}
                value={question.type}
                onChange={(e) =>
                  onChange({
                    ...question,
                    type: e.target.value as QuestionType,
                  })
                }
                className="w-full bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
              >
                {QUESTION_TYPES.map((t) => (
                  <option key={t.value} value={t.value}>
                    {t.label}
                  </option>
                ))}
              </select>
            )}
          </FormField>
          <FormField
            label={intl.formatMessage({
              id: "marketplace.quiz.points",
            })}
          >
            {({ id }) => (
              <Input
                id={id}
                type="number"
                min={1}
                value={question.points}
                onChange={(e) =>
                  onChange({ ...question, points: Number(e.target.value) })
                }
              />
            )}
          </FormField>
        </div>

        {/* Options for multiple choice */}
        {(question.type === "multiple_choice" ||
          question.type === "matching" ||
          question.type === "ordering") && (
          <div>
            <label className="type-label-md text-on-surface block mb-2">
              Options
            </label>
            <div className="space-y-2">
              {question.options.map((opt, i) => (
                <div key={i} className="flex gap-2">
                  <Input
                    value={opt}
                    onChange={(e) => updateOption(i, e.target.value)}
                    placeholder={`Option ${i + 1}`}
                    className="flex-1"
                  />
                  <button
                    type="button"
                    onClick={() => removeOption(i)}
                    className="p-2 text-on-surface-variant hover:text-error"
                  >
                    <Icon icon={Trash2} size="xs" />
                  </button>
                </div>
              ))}
              <Button
                type="button"
                variant="tertiary"
                size="sm"
                onClick={addOption}
              >
                <Icon icon={Plus} size="xs" className="mr-1" />
                Add Option
              </Button>
            </div>
          </div>
        )}

        {question.type === "true_false" && (
          <FormField label="Correct Answer">
            {({ id }) => (
              <select
                id={id}
                value={question.correct_answer}
                onChange={(e) =>
                  onChange({ ...question, correct_answer: e.target.value })
                }
                className="w-full bg-surface-container-highest rounded-radius-md p-3 text-on-surface type-body-md focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset"
              >
                <option value="true">True</option>
                <option value="false">False</option>
              </select>
            )}
          </FormField>
        )}

        {(question.type === "multiple_choice" ||
          question.type === "fill_blank") && (
          <FormField label="Correct Answer">
            {({ id }) => (
              <Input
                id={id}
                value={question.correct_answer}
                onChange={(e) =>
                  onChange({ ...question, correct_answer: e.target.value })
                }
              />
            )}
          </FormField>
        )}
      </div>
    </Card>
  );
}

// ─── Quiz builder page ───────────────────────────────────────────────────────

export function QuizBuilder() {
  const intl = useIntl();
  const [quizTitle, setQuizTitle] = useState("");
  const [questions, setQuestions] = useState<QuizQuestion[]>([]);

  const addQuestion = useCallback(() => {
    setQuestions((prev) => [
      ...prev,
      {
        id: genId(),
        type: "multiple_choice",
        text: "",
        options: ["", "", "", ""],
        correct_answer: "",
        points: 1,
      },
    ]);
  }, []);

  const updateQuestion = useCallback((index: number, q: QuizQuestion) => {
    setQuestions((prev) => prev.map((item, i) => (i === index ? q : item)));
  }, []);

  const removeQuestion = useCallback((index: number) => {
    setQuestions((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const moveQuestion = useCallback((from: number, to: number) => {
    setQuestions((prev) => {
      if (to < 0 || to >= prev.length) return prev;
      const arr = [...prev];
      const removed = arr.splice(from, 1);
      if (removed.length === 0) return prev;
      arr.splice(to, 0, removed[0]!);
      return arr;
    });
  }, []);

  const totalPoints = questions.reduce((sum, q) => sum + q.points, 0);

  return (
    <div className="max-w-content-narrow mx-auto">
      <PageTitle
        title={intl.formatMessage({ id: "marketplace.quiz.builder" })}
      />

      <RouterLink
        to="/creator"
        className="inline-flex items-center gap-1 mb-4 type-label-md text-on-surface-variant hover:text-primary transition-colors"
      >
        <Icon icon={ArrowLeft} size="sm" />
        <FormattedMessage id="marketplace.creator.dashboard" />
      </RouterLink>

      {/* Quiz meta */}
      <Card className="p-card-padding mb-6">
        <FormField
          label={intl.formatMessage({ id: "marketplace.quiz.title" })}
          required
        >
          {({ id }) => (
            <Input
              id={id}
              value={quizTitle}
              onChange={(e) => setQuizTitle(e.target.value)}
              placeholder="Quiz title"
              required
            />
          )}
        </FormField>
        <div className="flex items-center gap-4 mt-3">
          <span className="type-label-md text-on-surface-variant">
            {questions.length} questions
          </span>
          <span className="type-label-md text-on-surface-variant">
            {totalPoints} points
          </span>
        </div>
      </Card>

      {/* Questions */}
      <div className="space-y-4 mb-6" role="list" aria-label="Quiz questions">
        {questions.map((q, i) => (
          <QuestionEditor
            key={q.id}
            question={q}
            index={i}
            total={questions.length}
            onChange={(updated) => updateQuestion(i, updated)}
            onRemove={() => removeQuestion(i)}
            onMoveUp={() => moveQuestion(i, i - 1)}
            onMoveDown={() => moveQuestion(i, i + 1)}
          />
        ))}
      </div>

      {/* Add question */}
      <Button variant="secondary" onClick={addQuestion} className="w-full mb-6">
        <Icon icon={Plus} size="sm" className="mr-1" />
        <FormattedMessage id="marketplace.quiz.addQuestion" />
      </Button>

      {/* Save */}
      <div className="flex justify-end">
        <Button
          variant="primary"
          disabled={!quizTitle || questions.length === 0}
        >
          <FormattedMessage id="common.save" />
        </Button>
      </div>
    </div>
  );
}
