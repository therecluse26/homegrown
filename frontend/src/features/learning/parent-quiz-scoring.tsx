import { useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useParams, useNavigate } from "react-router";
import { ArrowLeft, Check, CheckCircle, XCircle, Minus } from "lucide-react";
import {
  Button,
  Card,
  EmptyState,
  Icon,
  Skeleton,
} from "@/components/ui";
import { useStudents } from "@/hooks/use-family";
import {
  useQuizSession,
  useQuizDef,
  useScoreQuiz,
  type QuestionScore,
  type QuizQuestionResponse,
} from "@/hooks/use-quiz";

// ─── Score option buttons ────────────────────────────────────────────────────

type ScoreLevel = "correct" | "partial" | "incorrect";

function ScoreButtons({
  value,
  onChange,
  maxPoints,
}: {
  value: ScoreLevel | undefined;
  onChange: (level: ScoreLevel, points: number) => void;
  maxPoints: number;
}) {
  const intl = useIntl();

  const options: { level: ScoreLevel; icon: typeof Check; points: number; colorClass: string }[] = [
    {
      level: "correct",
      icon: CheckCircle,
      points: maxPoints,
      colorClass: "bg-primary-container text-on-primary-container",
    },
    {
      level: "partial",
      icon: Minus,
      points: Math.round(maxPoints / 2),
      colorClass: "bg-tertiary-fixed text-on-tertiary-fixed",
    },
    {
      level: "incorrect",
      icon: XCircle,
      points: 0,
      colorClass: "bg-error-container text-on-error-container",
    },
  ];

  return (
    <div className="flex gap-2">
      {options.map((opt) => (
        <button
          key={opt.level}
          type="button"
          onClick={() => onChange(opt.level, opt.points)}
          className={`flex items-center gap-1.5 px-3 py-1.5 rounded-full type-label-sm font-medium transition-colors ${
            value === opt.level
              ? opt.colorClass
              : "bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high"
          }`}
        >
          <Icon icon={opt.icon} size="xs" aria-hidden />
          {intl.formatMessage({ id: `quiz.scoring.${opt.level}` })}
        </button>
      ))}
    </div>
  );
}

// ─── Main component ──────────────────────────────────────────────────────────

export function ParentQuizScoring() {
  const intl = useIntl();
  const navigate = useNavigate();
  const { sessionId } = useParams<{ sessionId: string }>();
  const { data: students } = useStudents();

  const studentId = students?.[0]?.id ?? "";

  const { data: session, isPending: sessionLoading } = useQuizSession(
    studentId,
    sessionId ?? "",
  );
  const { data: quizDef, isPending: defLoading } = useQuizDef(
    session?.quiz_def_id ?? "",
  );
  const scoreQuiz = useScoreQuiz(studentId);

  const [scores, setScores] = useState<
    Record<string, { level: ScoreLevel; points: number }>
  >({});

  const questions = quizDef?.questions ?? [];
  const pendingQuestions = questions.filter((q) => !q.auto_scorable);
  const allScored = pendingQuestions.every((q) => scores[q.question_id]);

  function handleScoreChange(
    questionId: string,
    level: ScoreLevel,
    points: number,
  ) {
    setScores((prev) => ({ ...prev, [questionId]: { level, points } }));
  }

  function handleScoreAll(level: ScoreLevel) {
    const newScores: Record<string, { level: ScoreLevel; points: number }> = {};
    for (const q of pendingQuestions) {
      const pts =
        level === "correct"
          ? q.points
          : level === "partial"
            ? Math.round(q.points / 2)
            : 0;
      newScores[q.question_id] = { level, points: pts };
    }
    setScores((prev) => ({ ...prev, ...newScores }));
  }

  function handleSubmitScores() {
    if (!sessionId) return;
    const questionScores: QuestionScore[] = Object.entries(scores).map(
      ([question_id, { points }]) => ({
        question_id,
        points_awarded: points,
      }),
    );
    scoreQuiz.mutate(
      { sessionId, scores: questionScores },
      {
        onSuccess: () => void navigate("/learning"),
      },
    );
  }

  function getStudentAnswer(question: QuizQuestionResponse): string {
    const ans = session?.answers?.[question.question_id];
    if (ans === undefined || ans === null) return "—";
    return String(ans);
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
        <Skeleton height="h-40" />
        <Skeleton height="h-40" />
      </div>
    );
  }

  if (session?.status === "scored") {
    return (
      <div className="mx-auto max-w-content-narrow space-y-6">
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
        <EmptyState
          message={intl.formatMessage({ id: "quiz.scoring.alreadyScored" })}
        />
      </div>
    );
  }

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
          <FormattedMessage
            id="quiz.scoring.title"
            values={{ quiz: quizDef?.title ?? "" }}
          />
        </h1>
      </div>

      {/* Batch actions */}
      {pendingQuestions.length > 1 && (
        <Card className="bg-surface-container-low">
          <p className="type-label-md text-on-surface-variant mb-2">
            <FormattedMessage
              id="quiz.scoring.batch"
              values={{ count: pendingQuestions.length }}
            />
          </p>
          <div className="flex gap-2">
            <Button
              variant="tertiary"
              size="sm"
              onClick={() => handleScoreAll("correct")}
            >
              <Icon icon={CheckCircle} size="sm" aria-hidden />
              <span className="ml-1">
                <FormattedMessage id="quiz.scoring.allCorrect" />
              </span>
            </Button>
            <Button
              variant="tertiary"
              size="sm"
              onClick={() => handleScoreAll("incorrect")}
            >
              <Icon icon={XCircle} size="sm" aria-hidden />
              <span className="ml-1">
                <FormattedMessage id="quiz.scoring.allIncorrect" />
              </span>
            </Button>
          </div>
        </Card>
      )}

      {/* Questions to score */}
      {pendingQuestions.length === 0 ? (
        <EmptyState
          message={intl.formatMessage({ id: "quiz.scoring.noPending" })}
        />
      ) : (
        <div className="space-y-4">
          {pendingQuestions.map((question, idx) => (
            <Card key={question.question_id}>
              <div className="mb-3">
                <span className="type-label-md text-on-surface-variant">
                  <FormattedMessage
                    id="quiz.questionNumber"
                    values={{
                      current: idx + 1,
                      total: pendingQuestions.length,
                    }}
                  />
                </span>
                <span className="ml-2 type-label-sm text-on-surface-variant">
                  ({question.points}{" "}
                  <FormattedMessage id="quiz.points" />)
                </span>
              </div>

              <p className="type-title-sm text-on-surface font-medium mb-2">
                {question.content}
              </p>

              <div className="p-3 rounded-xl bg-surface-container-low mb-3">
                <p className="type-label-sm text-on-surface-variant mb-1">
                  <FormattedMessage id="quiz.scoring.studentAnswer" />
                </p>
                <p className="type-body-md text-on-surface">
                  {getStudentAnswer(question)}
                </p>
              </div>

              <ScoreButtons
                value={scores[question.question_id]?.level}
                onChange={(level, points) =>
                  handleScoreChange(question.question_id, level, points)
                }
                maxPoints={question.points}
              />
            </Card>
          ))}
        </div>
      )}

      {/* Submit */}
      {pendingQuestions.length > 0 && (
        <div className="flex justify-end">
          <Button
            variant="primary"
            size="sm"
            onClick={handleSubmitScores}
            loading={scoreQuiz.isPending}
            disabled={!allScored}
          >
            <Icon icon={Check} size="sm" aria-hidden />
            <span className="ml-1">
              <FormattedMessage id="quiz.scoring.submit" />
            </span>
          </Button>
        </div>
      )}
    </div>
  );
}
