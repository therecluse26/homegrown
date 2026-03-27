import { Component, type ReactNode, type ErrorInfo } from "react";
import { Button } from "../ui/button";

type ErrorBoundaryProps = {
  children: ReactNode;
  /** Optional fallback UI. Receives the error and a reset function. */
  fallback?: (props: { error: Error; reset: () => void }) => ReactNode;
};

type ErrorBoundaryState = {
  error: Error | null;
};

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    // Log internally — never expose to user
    console.error("ErrorBoundary caught:", error, errorInfo);
  }

  handleReset = () => {
    this.setState({ error: null });
  };

  render() {
    const { error } = this.state;
    const { children, fallback } = this.props;

    if (error) {
      if (fallback) {
        return fallback({ error, reset: this.handleReset });
      }

      return (
        <div className="flex flex-col items-center justify-center gap-4 py-16 text-center">
          <h2 className="type-headline-sm text-on-surface">
            Something went wrong
          </h2>
          <p className="type-body-md text-on-surface-variant max-w-md">
            We encountered an unexpected error. Please try again or return to the
            home page.
          </p>
          <div className="flex gap-3">
            <Button variant="tertiary" onClick={this.handleReset}>
              Try again
            </Button>
            <Button
              variant="primary"
              onClick={() => {
                window.location.href = "/";
              }}
            >
              Go home
            </Button>
          </div>
        </div>
      );
    }

    return children;
  }
}
