import { useState, useCallback, useEffect, useRef } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useParams, useNavigate } from "react-router";
import {
  ArrowLeft,
  ArrowRight,
  CheckCircle,
  Clock,
  Save,
  Send,
} from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Input,
  ProgressBar,
  Skeleton,
} from "@/components/ui";
import { useStudents } from "@/hooks/use-family";
import {
  useQuizSession,
  useQuizDef,
  useUpdateQuizSession,
  type QuizQuestionResponse,
  type QuizSessionStatus,
} from "@/hooks/use-quiz";

// ─── Question renderers ──────────────────────────────────────────────────────

interface ChoiceOption {
  id: string;
  text: string;
}

function MultipleChoiceQuestion({
  question,
  answer,
  onAnswer,
  disabled,
}: {
  question: QuizQuestionResponse;
  answer: unknown;
  onAnswer: (value: unknown) => void;
  disabled: boolean;
}) {
  // Support both { choices: [{id, text}] } and { options: string[] } formats
  const raw = question.answer_data?.choices ?? question.answer_data?.options ?? [];
  const options: ChoiceOption[] = (raw as unknown[]).map((item, idx) =>
    typeof item === "string" ? { id: String(idx), text: item } : (item as ChoiceOption),
  );
  const selected = answer as string | undefined;

  return (
    <div className="space-y-2" role="radiogroup" aria-label={question.content}>
      {options.map((option) => (
        <label
          key={option.id}
          className={`flex items-center gap-3 p-3 rounded-xl cursor-pointer transition-colors ${
            selected === option.id
              ? "bg-primary-container text-on-primary-container"
              : "bg-surface-container-low text-on-surface hover:bg-surface-container-high"
          } ${disabled ? "pointer-events-none opacity-60" : ""}`}
        >
          <input
            type="radio"
            name={`q-${question.question_id}`}
            value={option.id}
            checked={selected === option.id}
            onChange={() => onAnswer(option.id)}
            disabled={disabled}
            className="sr-only"
          />
          <span
            className={`w-5 h-5 rounded-full border-2 flex items-center justify-center shrink-0 ${
              selected === option.id
                ? "border-primary bg-primary"
                : "border-outline"
            }`}
          >
            {selected === option.id && (
              <span className="w-2.5 h-2.5 rounded-full bg-on-primary" />
            )}
          </span>
          <span className="type-body-md">{option.text}</span>
        </label>
      ))}
    </div>
  );
}

function TrueFalseQuestion({
  question,
  answer,
  onAnswer,
  disabled,
}: {
  question: QuizQuestionResponse;
  answer: unknown;
  onAnswer: (value: unknown) => void;
  disabled: boolean;
}) {
  const intl = useIntl();
  const selected = answer as boolean | undefined;

  const options = [
    { value: true, label: intl.formatMessage({ id: "quiz.true" }) },
    { value: false, label: intl.formatMessage({ id: "quiz.false" }) },
  ];

  return (
    <div className="flex gap-3" role="radiogroup" aria-label={question.content}>
      {options.map((opt) => (
        <button
          key={String(opt.value)}
          type="button"
          onClick={() => onAnswer(opt.value)}
          disabled={disabled}
          className={`flex-1 p-4 rounded-xl type-title-sm font-medium transition-colors ${
            selected === opt.value
              ? "bg-primary-container text-on-primary-container"
              : "bg-surface-container-low text-on-surface hover:bg-surface-container-high"
          } ${disabled ? "pointer-events-none opacity-60" : ""}`}
        >
          {opt.label}
        </button>
      ))}
    </div>
  );
}

function ShortAnswerQuestion({
  question,
  answer,
  onAnswer,
  disabled,
}: {
  question: QuizQuestionResponse;
  answer: unknown;
  onAnswer: (value: unknown) => void;
  disabled: boolean;
}) {
  const intl = useIntl();
  const value = (answer as string) ?? "";

  return (
    <div>
      <Input
        value={value}
        onChange={(e) => onAnswer(e.target.value)}
        placeholder={intl.formatMessage({ id: "quiz.shortAnswer.placeholder" })}
        disabled={disabled}
      />
      {!question.auto_scorable && (
        <p className="type-label-sm text-on-surface-variant mt-2">
          <FormattedMessage id="quiz.shortAnswer.parentReview" />
        </p>
      )}
    </div>
  );
}

function FillInBlankQuestion({
  answer,
  onAnswer,
  disabled,
}: {
  question: QuizQuestionResponse;
  answer: unknown;
  onAnswer: (value: unknown) => void;
  disabled: boolean;
}) {
  const intl = useIntl();
  const value = (answer as string) ?? "";

  return (
    <Input
      value={value}
      onChange={(e) => onAnswer(e.target.value)}
      placeholder={intl.formatMessage({
        id: "quiz.fillInBlank.placeholder",
      })}
      disabled={disabled}
    />
  );
}

function QuestionRenderer({
  question,
  answer,
  onAnswer,
  disabled,
}: {
  question: QuizQuestionResponse;
  answer: unknown;
  onAnswer: (value: unknown) => void;
  disabled: boolean;
}) {
  switch (question.question_type) {
    case "multiple_choice":
      return (
        <MultipleChoiceQuestion
          question={question}
          answer={answer}
          onAnswer={onAnswer}
          disabled={disabled}
        />
      );
    case "true_false":
      return (
        <TrueFalseQuestion
          question={question}
          answer={answer}
          onAnswer={onAnswer}
          disabled={disabled}
        />
      );
    case "short_answer":
      return (
        <ShortAnswerQuestion
          question={question}
          answer={answer}
          onAnswer={onAnswer}
          disabled={disabled}
        />
      );
    case "fill_in_blank":
      return (
        <FillInBlankQuestion
          question={question}
          answer={answer}
          onAnswer={onAnswer}
          disabled={disabled}
        />
      );
    default:
      return (
        <ShortAnswerQuestion
          question={question}
          answer={answer}
          onAnswer={onAnswer}
          disabled={disabled}
        />
      );
  }
}

// ─── Score display ───────────────────────────────────────────────────────────

function ScoreDisplay({
  score,
  maxScore,
  passed,
}: {
  score: number;
  maxScore: number;
  passed: boolean;
}) {
  const pct = maxScore > 0 ? Math.round((score / maxScore) * 100) : 0;

  return (
    <Card className="text-center space-y-4">
      <div className="mx-auto w-24 h-24 rounded-full flex items-center justify-center bg-primary-container">
        <span className="type-headline-lg text-on-primary-container font-bold">
          {pct}%
        </span>
      </div>
      <p className="type-title-md text-on-surface font-semibold">
        <FormattedMessage
          id="quiz.score.result"
          values={{ score, maxScore }}
        />
      </p>
      <p
        className={`type-title-sm font-medium ${
          passed ? "text-primary" : "text-error"
        }`}
      >
        <FormattedMessage id={passed ? "quiz.score.passed" : "quiz.score.failed"} />
      </p>
    </Card>
  );
}

// ─── Main component ──────────────────────────────────────────────────────────

export function QuizPlayer() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { sessionId } = useParams<{ sessionId: string }>();
  const { data: students } = useStudents();
  const [currentIndex, setCurrentIndex] = useState(0);
  const [answers, setAnswers] = useState<Record<string, unknown>>({});
  const autoSaveTimer = useRef<ReturnType<typeof setTimeout>>(undefined);

  // For now we use the first student — in practice, session contains studentId
  const studentId = students?.[0]?.id ?? "";

  const { data: session, isPending: sessionLoading } = useQuizSession(
    studentId,
    sessionId ?? "",
  );
  const { data: quizDef, isPending: defLoading } = useQuizDef(
    session?.quiz_def_id ?? "",
  );
  const updateSession = useUpdateQuizSession(studentId);

  // Initialize answers from session
  useEffect(() => {
    if (session?.answers) {
      setAnswers(session.answers);
    }
  }, [session?.answers]);

  const questions = quizDef?.questions ?? [];
  const currentQuestion = questions[currentIndex];
  const isSubmitted = session?.status === "submitted" || session?.status === "scored";
  const isScored = session?.status === "scored";

  const totalAnswered = questions.filter(
    (q) => answers[q.question_id] !== undefined && answers[q.question_id] !== "",
  ).length;
  const progressPct =
    questions.length > 0
      ? Math.round((totalAnswered / questions.length) * 100)
      : 0;

  // Auto-save answers
  const saveAnswers = useCallback(
    (newAnswers: Record<string, unknown>) => {
      if (!sessionId || isSubmitted) return;
      updateSession.mutate({
        sessionId,
        answers: newAnswers,
      });
    },
    [sessionId, isSubmitted, updateSession],
  );

  function handleAnswer(value: unknown) {
    if (!currentQuestion || isSubmitted) return;
    const newAnswers = { ...answers, [currentQuestion.question_id]: value };
    setAnswers(newAnswers);

    // Debounced auto-save
    if (autoSaveTimer.current) clearTimeout(autoSaveTimer.current);
    autoSaveTimer.current = setTimeout(() => saveAnswers(newAnswers), 1500);
  }

  function handleSubmit() {
    if (!sessionId) return;
    // Save final answers and submit
    updateSession.mutate({
      sessionId,
      answers,
      submit: true,
    });
  }

  function handleSaveAndExit() {
    if (!sessionId) return;
    updateSession.mutate(
      { sessionId, answers },
      { onSuccess: () => void navigate("/learning") },
    );
  }

  if (!sessionId) {
    return (
      <EmptyState message={intl.formatMessage({ id: "quiz.noSession" })} />
    );
  }

  if (sessionLoading || defLoading) {
    return (
      <div className="mx-auto max-w-content-narrow space-y-6">
        <Skeleton height="h-8" />
        <Skeleton height="h-64" />
      </div>
    );
  }

  if (!session || !quizDef) {
    return (
      <EmptyState message={intl.formatMessage({ id: "quiz.notFound" })} />
    );
  }

  const statusLabel = (s: QuizSessionStatus) =>
    intl.formatMessage({ id: `quiz.status.${s}` });

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Button
          variant="tertiary"
          size="sm"
          onClick={() => void navigate("/learning")}
        >
          <Icon icon={ArrowLeft} size="sm" aria-hidden />
          <span className="ml-1">
            <FormattedMessage id="common.back" />
          </span>
        </Button>
        <h1 className="type-headline-md text-on-surface font-semibold">
          {quizDef?.title ?? ""}
        </h1>
      </div>

      {/* Status + progress */}
      <Card className="bg-surface-container-low">
        <div className="flex flex-wrap items-center justify-between gap-3 mb-3">
          <span className="type-label-md text-on-surface-variant">
            {statusLabel(session?.status ?? "not_started")}
          </span>
          {quizDef?.time_limit_minutes && (
            <span className="inline-flex items-center gap-1 type-label-sm text-on-surface-variant">
              <Icon icon={Clock} size="xs" aria-hidden />
              <FormattedMessage
                id="quiz.timeLimit"
                values={{ minutes: quizDef.time_limit_minutes }}
              />
            </span>
          )}
          <span className="type-label-sm text-on-surface-variant">
            <FormattedMessage
              id="quiz.progress"
              values={{
                answered: totalAnswered,
                total: questions.length,
              }}
            />
          </span>
        </div>
        <ProgressBar value={progressPct} />
      </Card>

      {/* Score display if scored */}
      <div aria-live="polite" aria-atomic="true">
        {isScored && session?.score !== undefined && session?.max_score !== undefined && (
          <ScoreDisplay
            score={session.score}
            maxScore={session.max_score}
            passed={session.passed ?? false}
          />
        )}
      </div>

      {/* Question card */}
      {currentQuestion && !isScored && (
        <Card>
          <div className="mb-4">
            <span className="type-label-md text-on-surface-variant">
              <FormattedMessage
                id="quiz.questionNumber"
                values={{
                  current: currentIndex + 1,
                  total: questions.length,
                }}
              />
            </span>
            <span className="ml-2 type-label-sm text-on-surface-variant">
              ({currentQuestion.points}{" "}
              <FormattedMessage id="quiz.points" />)
            </span>
          </div>

          <p className="type-title-md text-on-surface font-medium mb-4">
            {currentQuestion.content}
          </p>

          <QuestionRenderer
            question={currentQuestion}
            answer={answers[currentQuestion.question_id]}
            onAnswer={handleAnswer}
            disabled={isSubmitted}
          />
        </Card>
      )}

      {/* Navigation */}
      {questions.length > 0 && !isScored && (
        <div className="flex items-center justify-between">
          <Button
            variant="tertiary"
            size="sm"
            onClick={() => setCurrentIndex((i) => Math.max(0, i - 1))}
            disabled={currentIndex === 0}
          >
            <Icon icon={ArrowLeft} size="sm" aria-hidden />
            <span className="ml-1">
              <FormattedMessage id="quiz.prev" />
            </span>
          </Button>

          {/* Question dots */}
          <div className="flex gap-1.5 flex-wrap justify-center" role="tablist">
            {questions.map((q, idx) => {
              const hasAnswer =
                answers[q.question_id] !== undefined &&
                answers[q.question_id] !== "";
              return (
                <button
                  key={q.question_id}
                  type="button"
                  role="tab"
                  aria-selected={idx === currentIndex}
                  aria-label={intl.formatMessage(
                    { id: "quiz.goToQuestion" },
                    { number: idx + 1 },
                  )}
                  onClick={() => setCurrentIndex(idx)}
                  className={`w-3 h-3 rounded-full transition-colors ${
                    idx === currentIndex
                      ? "bg-primary"
                      : hasAnswer
                        ? "bg-tertiary-fixed"
                        : "bg-surface-container-high"
                  }`}
                />
              );
            })}
          </div>

          <Button
            variant="tertiary"
            size="sm"
            onClick={() =>
              setCurrentIndex((i) => Math.min(questions.length - 1, i + 1))
            }
            disabled={currentIndex === questions.length - 1}
          >
            <span className="mr-1">
              <FormattedMessage id="quiz.next" />
            </span>
            <Icon icon={ArrowRight} size="sm" aria-hidden />
          </Button>
        </div>
      )}

      {/* Action buttons */}
      {!isScored && (
        <div className="flex gap-2 justify-end">
          {!isSubmitted && (
            <>
              <Button
                variant="tertiary"
                size="sm"
                onClick={handleSaveAndExit}
                loading={updateSession.isPending}
              >
                <Icon icon={Save} size="sm" aria-hidden />
                <span className="ml-1">
                  <FormattedMessage id="quiz.saveAndExit" />
                </span>
              </Button>
              <Button
                variant="primary"
                size="sm"
                onClick={handleSubmit}
                loading={updateSession.isPending}
                disabled={totalAnswered === 0}
              >
                <Icon icon={Send} size="sm" aria-hidden />
                <span className="ml-1">
                  <FormattedMessage id="quiz.submit" />
                </span>
              </Button>
            </>
          )}
          {isSubmitted && !isScored && (
            <div className="flex items-center gap-2 type-body-md text-on-surface-variant">
              <Icon icon={CheckCircle} size="md" className="text-primary" aria-hidden />
              <FormattedMessage id="quiz.submitted.awaiting" />
            </div>
          )}
        </div>
      )}
    </div>
  );
}
