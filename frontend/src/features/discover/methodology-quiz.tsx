import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { ArrowLeft, ArrowRight, Send } from "lucide-react";
import { Button, Card, EmptyState, ProgressBar, Skeleton } from "@/components/ui";
import { useDiscoverQuiz, useSubmitQuiz } from "@/hooks/use-discover";

export function MethodologyQuiz() {
  const headingRef = useRef<HTMLHeadingElement>(null);
  const navigate = useNavigate();

  const { data: quiz, isPending, error } = useDiscoverQuiz();
  const submitMutation = useSubmitQuiz();

  const [currentIndex, setCurrentIndex] = useState(0);
  const [answers, setAnswers] = useState<Record<string, string>>({});

  useEffect(() => {
    document.title = "Methodology Quiz - Homegrown Academy";
    headingRef.current?.focus();
  }, []);

  // Focus heading when question changes
  useEffect(() => {
    headingRef.current?.focus();
  }, [currentIndex]);

  if (isPending) {
    return (
      <div className="space-y-6">
        <Skeleton width="w-48" height="h-8" />
        <Skeleton width="w-full" height="h-2" />
        <Skeleton width="w-full" height="h-32" />
        <div className="space-y-3">
          <Skeleton width="w-full" height="h-14" />
          <Skeleton width="w-full" height="h-14" />
          <Skeleton width="w-full" height="h-14" />
        </div>
      </div>
    );
  }

  if (error || !quiz?.questions?.length) {
    return (
      <EmptyState
        message="Unable to load the quiz"
        description="Please try again later."
        action={
          <Button onClick={() => window.location.reload()}>Retry</Button>
        }
      />
    );
  }

  const questions = quiz.questions;
  const total = questions.length;
  const question = questions[currentIndex];
  const questionId = question?.id ?? "";
  const selectedAnswer = answers[questionId];
  const progress = ((currentIndex + 1) / total) * 100;
  const isLast = currentIndex === total - 1;
  const answeredCount = Object.keys(answers).length;

  function selectAnswer(answerId: string) {
    setAnswers((prev) => ({ ...prev, [questionId]: answerId }));
  }

  function goBack() {
    if (currentIndex > 0) setCurrentIndex((i) => i - 1);
  }

  function goNext() {
    if (!isLast && selectedAnswer) setCurrentIndex((i) => i + 1);
  }

  function handleSubmit() {
    submitMutation.mutate(
      { answers },
      {
        onSuccess: (result) => {
          if (result.share_id) {
            void navigate(`/discover/results/${result.share_id}`);
          }
        },
      },
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="space-y-2">
        <p className="type-label-md text-on-surface-variant">
          Question {currentIndex + 1} of {total}
        </p>
        <ProgressBar value={progress} label="Quiz progress" />
      </div>

      {/* Question */}
      <div className="space-y-2">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-title-lg text-on-surface font-semibold outline-none"
        >
          {question?.text}
        </h1>
        {question?.help_text && (
          <p className="type-body-md text-on-surface-variant">
            {question.help_text}
          </p>
        )}
      </div>

      {/* Answer options */}
      <div className="space-y-3" role="radiogroup" aria-label="Answer choices">
        {question?.answers?.map((answer) => {
          const isSelected = selectedAnswer === answer.id;
          return (
            <Card
              key={answer.id}
              interactive
              role="radio"
              aria-checked={isSelected}
              tabIndex={0}
              onClick={() => selectAnswer(answer.id ?? "")}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  selectAnswer(answer.id ?? "");
                }
              }}
              className={`cursor-pointer transition-all ${
                isSelected
                  ? "ring-2 ring-primary bg-primary-container/30"
                  : "hover:bg-surface-container-low"
              }`}
            >
              <p className="type-body-md text-on-surface">{answer.text}</p>
            </Card>
          );
        })}
      </div>

      {/* Navigation */}
      <div className="flex items-center justify-between pt-2">
        <Button
          variant="tertiary"
          onClick={goBack}
          disabled={currentIndex === 0}
          leadingIcon={<ArrowLeft className="h-4 w-4" />}
        >
          Back
        </Button>

        {isLast ? (
          <Button
            variant="primary"
            onClick={handleSubmit}
            disabled={answeredCount === 0}
            loading={submitMutation.isPending}
            trailingIcon={<Send className="h-4 w-4" />}
          >
            See Results
          </Button>
        ) : (
          <Button
            variant="primary"
            onClick={goNext}
            disabled={!selectedAnswer}
            trailingIcon={<ArrowRight className="h-4 w-4" />}
          >
            Next
          </Button>
        )}
      </div>

      {/* Submission error */}
      {submitMutation.error && (
        <Card className="bg-error-container">
          <p className="type-body-sm text-on-error-container">
            Something went wrong submitting your quiz. Please try again.
          </p>
        </Card>
      )}
    </div>
  );
}
