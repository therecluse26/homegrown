import messages from "@/locales/en.json";

export const defaultLocale = "en";

export const localeMessages: Record<string, Record<string, string>> = {
  en: messages,
};

export function getMessages(locale: string): Record<string, string> {
  return localeMessages[locale] ?? localeMessages[defaultLocale] ?? {};
}
