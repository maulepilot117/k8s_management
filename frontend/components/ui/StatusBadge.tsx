type StatusVariant = "success" | "warning" | "danger" | "info" | "neutral";

interface StatusBadgeProps {
  status: string;
  variant?: StatusVariant;
}

const variantClasses: Record<StatusVariant, string> = {
  success:
    "bg-green-50 text-green-700 ring-green-600/20 dark:bg-green-500/10 dark:text-green-400 dark:ring-green-500/20",
  warning:
    "bg-amber-50 text-amber-700 ring-amber-600/20 dark:bg-amber-500/10 dark:text-amber-400 dark:ring-amber-500/20",
  danger:
    "bg-red-50 text-red-700 ring-red-600/20 dark:bg-red-500/10 dark:text-red-400 dark:ring-red-500/20",
  info:
    "bg-blue-50 text-blue-700 ring-blue-600/20 dark:bg-blue-500/10 dark:text-blue-400 dark:ring-blue-500/20",
  neutral:
    "bg-slate-50 text-slate-600 ring-slate-500/20 dark:bg-slate-500/10 dark:text-slate-400 dark:ring-slate-500/20",
};

/** Maps common k8s status strings to semantic variants. */
function autoVariant(status: string): StatusVariant {
  const s = status.toLowerCase();
  if (
    [
      "running",
      "active",
      "bound",
      "ready",
      "healthy",
      "available",
      "complete",
      "succeeded",
    ].includes(s)
  ) return "success";
  if (
    ["pending", "waiting", "creating", "terminating", "warning"].includes(s)
  ) return "warning";
  if (
    [
      "failed",
      "error",
      "crashloopbackoff",
      "imagepullbackoff",
      "evicted",
      "oomkilled",
      "lost",
    ].includes(s)
  ) return "danger";
  if (["unknown", "not ready"].includes(s)) return "neutral";
  return "info";
}

export function StatusBadge({ status, variant }: StatusBadgeProps) {
  const v = variant ?? autoVariant(status);
  return (
    <span
      class={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${
        variantClasses[v]
      }`}
    >
      {status}
    </span>
  );
}
