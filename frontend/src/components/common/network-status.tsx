import { useState, useEffect } from "react";

export function NetworkStatus() {
  const [online, setOnline] = useState(navigator.onLine);

  useEffect(() => {
    const handleOnline = () => setOnline(true);
    const handleOffline = () => setOnline(false);

    window.addEventListener("online", handleOnline);
    window.addEventListener("offline", handleOffline);

    return () => {
      window.removeEventListener("online", handleOnline);
      window.removeEventListener("offline", handleOffline);
    };
  }, []);

  if (online) return null;

  return (
    <div
      className="fixed bottom-0 left-0 right-0 z-notification bg-warning-container px-4 py-2 text-center type-body-sm text-on-warning-container"
      role="alert"
    >
      You are offline. Some features may be unavailable.
    </div>
  );
}
