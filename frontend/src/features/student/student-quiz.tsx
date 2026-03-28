import { useState, useCallback, useEffect, useRef } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useParams, useNavigate } from "react-router";
import { ArrowLeft, ArrowRight, CheckCircle, Clock, Send } from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Input,
  ProgressBar,
  Skeleton,
} from "@/components/ui";
import { useStudentSession } from "@/hooks/use-student-session";
import {
  useQuizSession,
  useQuizDef,
  useUpdateQuizSession,
  type QuizQuestionResponse,
} from "@/hooks/use-quiz";

// ─── Simplified question renderers ───────────────────────────────────────────

function QuestionInput({
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

  if (question.question_type === "multiple_choice") {
    const options = (question.answer_data?.options as string[]) ?? [];
    const selected = answer as string | undefined;
    return (
      <div className="space-y-2" role="radiogroup">
        {options.map((option, idx) => (
          <label
            key={idx}
            className={`flex items-center gap-3 p-3 rounded-xl cursor-pointer transition-colors ${
              selected === option
                ? "bg-primary-container text-on-primary-container"
                : "bg-surface-container-low text-on-surface hover:bg-surface-container-high"
            } ${disabled ? "pointer-events-none opacity-60" : ""}`}
          >
            <input
              type="radio"
              name={`q-${question.question_id}`}
              value={option}
              checked={selected === option}
              onChange={() => onAnswer(option)}
              disabled={disabled}
              className="sr-only"
            />
            <span
              className={`w-5 h-5 rounded-full border-2 flex items-center justify-center shrink-0 ${
                selected === option
                  ? "border-primary bg-primary"
                  : "border-outline"
              }`}
            >
              {selected === option && (
                <span className="w-2.5 h-2.5 rounded-full bg-on-primary" />
              )}
            </span>
            <span className="type-body-md">{option}</span>
          </label>
        ))}
      </div>
    );
  }

  if (question.question_type === "true_false") {
    const selected = answer as boolean | undefined;
    return (
      <div className="flex gap-3">
        {[true, false].map((val) => (
          <button
            key={String(val)}
            type="button"
            onClick={() => onAnswer(val)}
            disabled={disabled}
            className={`flex-1 p-4 rounded-xl type-title-sm font-medium transition-colors ${
              selected === val
                ? "bg-primary-container text-on-primary-container"
                : "bg-surface-container-low text-on-surface hover:bg-surface-container-high"
            }`}
          >
            {intl.formatMessage({ id: val ? "quiz.true" : "quiz.false" })}
          </button>
        ))}
      </div>
    );
  }

  // Short answer / fill-in-blank
  return (
    <Input
      value={(answer as string) ?? ""}
      onChange={(e) => onAnswer(e.target.value)}
      placeholder={intl.formatMessage({ id: "quiz.shortAnswer.placeholder" })}
      disabled={disabled}
    />
  );
}

// ─── Main component ──────────────────────────────────────────────────────────

export function StudentQuiz() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { sessionId } = useParams<{ sessionId: string }>();
  const { session: studentSession } = useStudentSession();
  const studentId = studentSession?.studentId ?? "";

  const [currentIndex, setCurrentIndex] = useState(0);
  const [answers, setAnswers] = useState<Record<string, unknown>>({});
  const autoSaveTimer = useRef<ReturnType<typeof setTimeout>>(undefined);

  const { data: session, isPending: sessionLoading } = useQuizSession(
    studentId,
    sessionId ?? "",
  );
  const { data: quizDef, isPending: defLoading } = useQuizDef(
    session?.quiz_def_id ?? "",
  );
  const updateSession = useUpdateQuizSession(studentId);

  useEffect(() => {
    if (session?.answers) setAnswers(session.answers);
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

  const saveAnswers = useCallback(
    (newAnswers: Record<string, unknown>) => {
      if (!sessionId || isSubmitted) return;
      updateSession.mutate({ sessionId, answers: newAnswers });
    },
    [sessionId, isSubmitted, updateSession],
  );

  function handleAnswer(value: unknown) {
    if (!currentQuestion || isSubmitted) return;
    const newAnswers = { ...answers, [currentQuestion.question_id]: value };
    setAnswers(newAnswers);
    if (autoSaveTimer.current) clearTimeout(autoSaveTimer.current);
    autoSaveTimer.current = setTimeout(() => saveAnswers(newAnswers), 1500);
  }

  function handleSubmit() {
    if (!sessionId) return;
    updateSession.mutate({ sessionId, answers, submit: true });
  }

  if (!sessionId) {
    return <EmptyState message={intl.formatMessage({ id: "quiz.noSession" })} />;
  }

  if (sessionLoading || defLoading) {
    return (
      <div className="mx-auto max-w-content-narrow space-y-6">
        <Skeleton height="h-8" />
        <Skeleton height="h-64" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-content-narrow space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Button variant="tertiary" size="sm" onClick={() => void navigate(-1)}>
          <Icon icon={ArrowLeft} size="sm" aria-hidden />
          <span className="ml-1">
            <FormattedMessage id="common.back" />
          </span>
        </Button>
        <h1 className="type-headline-md text-on-surface font-semibold">
          {quizDef?.title ?? ""}
        </h1>
      </div>

      {/* Progress */}
      <Card className="bg-surface-container-low">
        <div className="flex items-center justify-between mb-2">
          {quizDef?.time_limit_minutes && (
            <span className="inline-flex items-center gap-1 type-label-sm text-on-surface-variant">
              <Icon icon={Clock} size="xs" aria-hidden />
              {quizDef.time_limit_minutes} min
            </span>
          )}
          <span className="type-label-sm text-on-surface-variant">
            {totalAnswered}/{questions.length}
          </span>
        </div>
        <ProgressBar value={progressPct} />
      </Card>

      {/* Score result */}
      {isScored && session?.score !== undefined && (
        <Card className="text-center space-y-3">
          <div className="mx-auto w-20 h-20 rounded-full flex items-center justify-center bg-primary-container">
            <span className="type-headline-md text-on-primary-container font-bold">
              {session.max_score
                ? Math.round((session.score / session.max_score) * 100)
                : session.score}
              %
            </span>
          </div>
          <p
            className={`type-title-sm font-medium ${
              session.passed ? "text-primary" : "text-error"
            }`}
          >
            <FormattedMessage
              id={session.passed ? "quiz.score.passed" : "quiz.score.failed"}
            />
          </p>
        </Card>
      )}

      {/* Question */}
      {currentQuestion && !isScored && (
        <Card>
          <p className="type-label-md text-on-surface-variant mb-2">
            <FormattedMessage
              id="quiz.questionNumber"
              values={{ current: currentIndex + 1, total: questions.length }}
            />
          </p>
          <p className="type-title-md text-on-surface font-medium mb-4">
            {currentQuestion.content}
          </p>
          <QuestionInput
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
          </Button>

          {!isSubmitted ? (
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
          ) : (
            <span className="inline-flex items-center gap-1 type-body-sm text-on-surface-variant">
              <Icon icon={CheckCircle} size="sm" className="text-primary" aria-hidden />
              <FormattedMessage id="quiz.submitted.awaiting" />
            </span>
          )}

          <Button
            variant="tertiary"
            size="sm"
            onClick={() =>
              setCurrentIndex((i) => Math.min(questions.length - 1, i + 1))
            }
            disabled={currentIndex === questions.length - 1}
          >
            <Icon icon={ArrowRight} size="sm" aria-hidden />
          </Button>
        </div>
      )}
    </div>
  );
}
