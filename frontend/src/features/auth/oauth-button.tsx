import { useIntl } from "react-intl";

type OAuthProvider = "google" | "facebook" | "apple";

interface OAuthButtonProps {
  provider: OAuthProvider;
  kratosActionUrl: string;
  csrfToken: string;
  disabled?: boolean;
}

const PROVIDER_ICONS: Record<OAuthProvider, string> = {
  google: `<svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/><path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/><path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/><path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/></svg>`,
  facebook: `<svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><path fill="#1877F2" d="M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z"/></svg>`,
  apple: `<svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true" fill="currentColor"><path d="M14.94 5.19A4.38 4.38 0 0 0 16 2a4.44 4.44 0 0 0-3 1.52 4.17 4.17 0 0 0-1 3.09 3.69 3.69 0 0 0 2.94-1.42zm2.52 7.44a4.51 4.51 0 0 1 2.16-3.81 4.66 4.66 0 0 0-3.66-2c-1.56-.16-3 .91-3.83.91-.83 0-2-.89-3.3-.87a4.92 4.92 0 0 0-4.14 2.53C2.93 12 4.24 17 6 19.47c.89 1.21 1.94 2.58 3.32 2.53s1.87-.82 3.5-.82 2.1.82 3.5.78 2.34-1.24 3.2-2.47a10.91 10.91 0 0 0 1.44-2.84 4.39 4.39 0 0 1-2.5-3.02z"/></svg>`,
};

/**
 * OAuthButton renders a provider-specific OAuth initiation button.
 * Submits a POST to the Kratos flow action URL with `provider` as method,
 * which triggers an OAuth redirect.
 */
export function OAuthButton({
  provider,
  kratosActionUrl,
  csrfToken,
  disabled = false,
}: OAuthButtonProps) {
  const intl = useIntl();

  const label = intl.formatMessage({ id: `auth.oauth.${provider}` });

  function handleClick() {
    // OAuth initiation: submit a hidden form POST to Kratos
    // Kratos responds with a redirect to the OAuth provider
    const form = document.createElement("form");
    form.method = "POST";
    form.action = kratosActionUrl;

    const methodInput = document.createElement("input");
    methodInput.type = "hidden";
    methodInput.name = "provider";
    methodInput.value = provider;
    form.appendChild(methodInput);

    const csrfInput = document.createElement("input");
    csrfInput.type = "hidden";
    csrfInput.name = "csrf_token";
    csrfInput.value = csrfToken;
    form.appendChild(csrfInput);

    document.body.appendChild(form);
    form.submit();
    document.body.removeChild(form);
  }

  return (
    <button
      type="button"
      onClick={handleClick}
      disabled={disabled}
      aria-label={label}
      className="flex w-full items-center justify-center gap-3 rounded-button border border-outline-variant bg-surface px-4 py-2.5 text-label-md font-medium text-on-surface transition-colors hover:bg-surface-container-low focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-disabled"
    >
      <span
        dangerouslySetInnerHTML={{ __html: PROVIDER_ICONS[provider] }}
        aria-hidden="true"
        className="flex-shrink-0"
      />
      {label}
    </button>
  );
}
