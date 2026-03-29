import { FormattedMessage, useIntl } from "react-intl";
import { ShieldCheck, Copy, Download, ShieldOff, Loader2 } from "lucide-react";
import {
  Badge,
  Button,
  Card,
  ConfirmationDialog,
  Icon,
  Input,
  Skeleton,
  Spinner,
} from "@/components/ui";
import {
  useMfaStatus,
  useInitTotpSetup,
  useVerifyTotp,
  useDisableMfa,
} from "@/hooks/use-mfa";
import { useState, useEffect, useRef, useCallback } from "react";

// ─── QR Code renderer (SVG-based, no external dependency) ──────────────────

function QrCodeDisplay({ uri }: { uri: string }) {
  const intl = useIntl();
  return (
    <div className="flex flex-col items-center gap-3">
      {/* Use a simple image tag with QR API for the otpauth URI.
          In production this would be a local QR lib — using a data-uri
          placeholder pattern to avoid external dependency. */}
      <div
        className="w-48 h-48 bg-surface-container-lowest rounded-radius-lg flex items-center justify-center border-2 border-outline-variant"
        role="img"
        aria-label={intl.formatMessage({ id: "mfa.setup.qrCode.alt" })}
      >
        <div className="text-center p-4">
          <Icon icon={ShieldCheck} size="xl" className="text-primary mx-auto mb-2" aria-hidden />
          <p className="type-label-sm text-on-surface-variant">
            <FormattedMessage id="mfa.setup.qrCode.scanPrompt" />
          </p>
        </div>
      </div>
      <div className="w-full">
        <p className="type-label-sm text-on-surface-variant mb-1">
          <FormattedMessage id="mfa.setup.manualEntry" />
        </p>
        <code className="type-body-sm text-on-surface bg-surface-container-high px-3 py-2 rounded-radius-sm block break-all select-all">
          {uri}
        </code>
      </div>
    </div>
  );
}

// ─── Recovery codes display ────────────────────────────────────────────────

function RecoveryCodes({ codes }: { codes: string[] }) {
  const intl = useIntl();
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(() => {
    void navigator.clipboard.writeText(codes.join("\n")).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }, [codes]);

  const handleDownload = useCallback(() => {
    const content = [
      intl.formatMessage({ id: "mfa.recovery.fileHeader" }),
      "",
      ...codes,
      "",
      intl.formatMessage({ id: "mfa.recovery.fileFooter" }),
    ].join("\n");
    const blob = new Blob([content], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "homegrown-academy-recovery-codes.txt";
    a.click();
    URL.revokeObjectURL(url);
  }, [codes, intl]);

  return (
    <div>
      <div className="bg-surface-container-high rounded-radius-md p-4 mb-3">
        <div className="grid grid-cols-2 gap-2" role="list" aria-label={intl.formatMessage({ id: "mfa.recovery.listLabel" })}>
          {codes.map((code, i) => (
            <code
              key={i}
              role="listitem"
              className="type-body-md text-on-surface font-mono bg-surface-container-lowest px-3 py-1.5 rounded-radius-sm text-center select-all"
            >
              {code}
            </code>
          ))}
        </div>
      </div>
      <div className="flex gap-2">
        <Button variant="secondary" size="sm" onClick={handleCopy}>
          <Icon icon={Copy} size="xs" aria-hidden className="mr-1.5" />
          {copied ? (
            <FormattedMessage id="mfa.recovery.copied" />
          ) : (
            <FormattedMessage id="mfa.recovery.copy" />
          )}
        </Button>
        <Button variant="secondary" size="sm" onClick={handleDownload}>
          <Icon icon={Download} size="xs" aria-hidden className="mr-1.5" />
          <FormattedMessage id="mfa.recovery.download" />
        </Button>
      </div>
    </div>
  );
}

// ─── Setup flow steps ──────────────────────────────────────────────────────

type SetupStep = "idle" | "scanning" | "verifying" | "complete";

// ─── Component ─────────────────────────────────────────────────────────────

export function MfaSetup() {
  const intl = useIntl();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const mfaStatus = useMfaStatus();
  const initSetup = useInitTotpSetup();
  const verifyTotp = useVerifyTotp();
  const disableMfa = useDisableMfa();

  const [step, setStep] = useState<SetupStep>("idle");
  const [verifyCode, setVerifyCode] = useState("");
  const [disableCode, setDisableCode] = useState("");
  const [showDisableDialog, setShowDisableDialog] = useState(false);
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([]);
  const [verifyError, setVerifyError] = useState<string | null>(null);

  useEffect(() => {
    document.title = `${intl.formatMessage({ id: "mfa.title" })} — ${intl.formatMessage({ id: "app.name" })}`;
    headingRef.current?.focus();
  }, [intl]);

  const handleStartSetup = useCallback(() => {
    initSetup.mutate(undefined, {
      onSuccess: () => {
        setStep("scanning");
      },
    });
  }, [initSetup]);

  const handleVerify = useCallback(() => {
    if (verifyCode.length !== 6) return;
    setVerifyError(null);
    verifyTotp.mutate(
      { code: verifyCode },
      {
        onSuccess: (data) => {
          setRecoveryCodes(data.recovery_codes);
          setStep("complete");
        },
        onError: () => {
          setVerifyError(
            intl.formatMessage({ id: "mfa.setup.verify.error" }),
          );
        },
      },
    );
  }, [verifyCode, verifyTotp, intl]);

  const handleDisable = useCallback(() => {
    disableMfa.mutate(
      { code: disableCode },
      {
        onSuccess: () => {
          setShowDisableDialog(false);
          setDisableCode("");
        },
      },
    );
  }, [disableCode, disableMfa]);

  // ─── Loading ────────────────────────────────────────────────────────────

  if (mfaStatus.isPending) {
    return (
      <div className="mx-auto max-w-2xl">
        <Skeleton height="h-8" width="w-48" className="mb-6" />
        <Skeleton height="h-40" />
      </div>
    );
  }

  // ─── Error ──────────────────────────────────────────────────────────────

  if (mfaStatus.error) {
    return (
      <div className="mx-auto max-w-2xl">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-6"
        >
          <FormattedMessage id="mfa.title" />
        </h1>
        <Card className="bg-error-container">
          <p className="type-body-md text-on-error-container">
            <FormattedMessage id="error.generic" />
          </p>
        </Card>
      </div>
    );
  }

  const isEnabled = mfaStatus.data?.enabled ?? false;

  // ─── MFA already enabled ───────────────────────────────────────────────

  if (isEnabled && step === "idle") {
    return (
      <div className="mx-auto max-w-2xl">
        <h1
          ref={headingRef}
          tabIndex={-1}
          className="type-headline-md text-on-surface font-semibold outline-none mb-6"
        >
          <FormattedMessage id="mfa.title" />
        </h1>
        <Card>
          <div className="flex items-start gap-4">
            <div className="w-10 h-10 rounded-radius-full bg-primary/10 flex items-center justify-center shrink-0">
              <Icon icon={ShieldCheck} size="md" className="text-primary" aria-hidden />
            </div>
            <div className="flex-1">
              <div className="flex items-center gap-2 mb-1">
                <p className="type-title-sm text-on-surface font-medium">
                  <FormattedMessage id="mfa.enabled.title" />
                </p>
                <Badge variant="primary">
                  <FormattedMessage id="mfa.enabled.badge" />
                </Badge>
              </div>
              <p className="type-body-sm text-on-surface-variant mb-4">
                <FormattedMessage id="mfa.enabled.description" />
              </p>
              <Button
                variant="tertiary"
                size="sm"
                onClick={() => setShowDisableDialog(true)}
                className="text-error"
              >
                <Icon icon={ShieldOff} size="xs" aria-hidden className="mr-1.5" />
                <FormattedMessage id="mfa.disable.button" />
              </Button>
            </div>
          </div>
        </Card>

        <ConfirmationDialog
          open={showDisableDialog}
          onClose={() => {
            setShowDisableDialog(false);
            setDisableCode("");
          }}
          onConfirm={handleDisable}
          title={intl.formatMessage({ id: "mfa.disable.title" })}
          confirmLabel={intl.formatMessage({ id: "mfa.disable.confirm" })}
          destructive
          loading={disableMfa.isPending}
        >
          <div className="flex flex-col gap-3">
            <p className="type-body-md text-on-surface">
              <FormattedMessage id="mfa.disable.warning" />
            </p>
            <div>
              <label
                htmlFor="disable-code"
                className="type-label-md text-on-surface font-medium mb-1 block"
              >
                <FormattedMessage id="mfa.disable.codeLabel" />
              </label>
              <Input
                id="disable-code"
                value={disableCode}
                onChange={(e) => setDisableCode(e.target.value)}
                placeholder="000000"
                maxLength={6}
                inputMode="numeric"
                pattern="[0-9]*"
                autoComplete="one-time-code"
              />
            </div>
          </div>
        </ConfirmationDialog>
      </div>
    );
  }

  // ─── Setup flow ────────────────────────────────────────────────────────

  return (
    <div className="mx-auto max-w-2xl">
      <h1
        ref={headingRef}
        tabIndex={-1}
        className="type-headline-md text-on-surface font-semibold outline-none mb-2"
      >
        <FormattedMessage id="mfa.title" />
      </h1>
      <p className="type-body-md text-on-surface-variant mb-6">
        <FormattedMessage id="mfa.setup.description" />
      </p>

      {/* Step: idle — start setup */}
      {step === "idle" && (
        <Card>
          <div className="flex items-start gap-4">
            <div className="w-10 h-10 rounded-radius-full bg-surface-container-high flex items-center justify-center shrink-0">
              <Icon icon={ShieldCheck} size="md" className="text-on-surface-variant" aria-hidden />
            </div>
            <div className="flex-1">
              <p className="type-title-sm text-on-surface font-medium mb-1">
                <FormattedMessage id="mfa.setup.notEnabled.title" />
              </p>
              <p className="type-body-sm text-on-surface-variant mb-4">
                <FormattedMessage id="mfa.setup.notEnabled.description" />
              </p>
              <Button
                variant="primary"
                onClick={handleStartSetup}
                disabled={initSetup.isPending}
              >
                {initSetup.isPending ? (
                  <Spinner size="sm" className="mr-2" />
                ) : (
                  <Icon icon={ShieldCheck} size="xs" aria-hidden className="mr-1.5" />
                )}
                <FormattedMessage id="mfa.setup.start" />
              </Button>
            </div>
          </div>
        </Card>
      )}

      {/* Step: scanning — show QR code */}
      {step === "scanning" && initSetup.data && (
        <Card>
          <h2 className="type-title-md text-on-surface font-semibold mb-2">
            <FormattedMessage id="mfa.setup.step1.title" />
          </h2>
          <p className="type-body-sm text-on-surface-variant mb-4">
            <FormattedMessage id="mfa.setup.step1.description" />
          </p>
          <QrCodeDisplay uri={initSetup.data.otpauth_uri} />
          <div className="mt-6 pt-4 border-t border-outline-variant/20">
            <h3 className="type-title-sm text-on-surface font-medium mb-2">
              <FormattedMessage id="mfa.setup.step2.title" />
            </h3>
            <p className="type-body-sm text-on-surface-variant mb-3">
              <FormattedMessage id="mfa.setup.step2.description" />
            </p>
            <div className="flex items-end gap-3">
              <div className="flex-1 max-w-48">
                <label
                  htmlFor="verify-code"
                  className="type-label-md text-on-surface font-medium mb-1 block"
                >
                  <FormattedMessage id="mfa.setup.verify.label" />
                </label>
                <Input
                  id="verify-code"
                  value={verifyCode}
                  onChange={(e) => {
                    setVerifyCode(e.target.value.replace(/\D/g, "").slice(0, 6));
                    setVerifyError(null);
                  }}
                  placeholder="000000"
                  maxLength={6}
                  inputMode="numeric"
                  pattern="[0-9]*"
                  autoComplete="one-time-code"
                  aria-describedby={verifyError ? "verify-error" : undefined}
                  aria-invalid={!!verifyError}
                />
                {verifyError && (
                  <p
                    id="verify-error"
                    className="type-body-sm text-error mt-1"
                    role="alert"
                  >
                    {verifyError}
                  </p>
                )}
              </div>
              <Button
                variant="primary"
                onClick={handleVerify}
                disabled={verifyCode.length !== 6 || verifyTotp.isPending}
              >
                {verifyTotp.isPending && (
                  <Icon icon={Loader2} size="xs" aria-hidden className="mr-1.5 animate-spin" />
                )}
                <FormattedMessage id="mfa.setup.verify.submit" />
              </Button>
            </div>
          </div>
        </Card>
      )}

      {/* Step: complete — show recovery codes */}
      {step === "complete" && (
        <Card>
          <div className="flex items-center gap-2 mb-2">
            <Icon icon={ShieldCheck} size="md" className="text-primary" aria-hidden />
            <h2 className="type-title-md text-on-surface font-semibold">
              <FormattedMessage id="mfa.setup.complete.title" />
            </h2>
          </div>
          <p className="type-body-sm text-on-surface-variant mb-4">
            <FormattedMessage id="mfa.setup.complete.description" />
          </p>
          <div className="bg-warning-container/30 rounded-radius-md p-3 mb-4">
            <p className="type-body-sm text-on-surface font-medium">
              <FormattedMessage id="mfa.recovery.warning" />
            </p>
          </div>
          <RecoveryCodes codes={recoveryCodes} />
          <div className="mt-6 pt-4 border-t border-outline-variant/20">
            <Button
              variant="primary"
              onClick={() => {
                setStep("idle");
                setVerifyCode("");
                setRecoveryCodes([]);
              }}
            >
              <FormattedMessage id="mfa.setup.complete.done" />
            </Button>
          </div>
        </Card>
      )}
    </div>
  );
}
