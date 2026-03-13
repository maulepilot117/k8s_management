import { useSignal } from "@preact/signals";
import { useEffect } from "preact/hooks";
import { IS_BROWSER } from "fresh/runtime";
import { apiGet } from "@/lib/api.ts";

interface PerformancePanelProps {
  kind: string;
  name: string;
  namespace?: string;
}

/** Maps resource kinds to their Grafana dashboard UIDs and variable names. */
const DASHBOARD_MAP: Record<string, { uid: string; varName: string }> = {
  pods: { uid: "kubecenter-pod-detail", varName: "pod" },
  deployments: { uid: "kubecenter-deployment-detail", varName: "deployment" },
  statefulsets: {
    uid: "kubecenter-statefulset-detail",
    varName: "statefulset",
  },
  daemonsets: { uid: "kubecenter-daemonset-detail", varName: "daemonset" },
  nodes: { uid: "kubecenter-node-detail", varName: "node" },
  pvcs: { uid: "kubecenter-pvc-detail", varName: "pvc" },
};

interface MonitoringStatusResponse {
  prometheus: { available: boolean };
  grafana: { available: boolean };
}

export default function PerformancePanel(
  { kind, name, namespace }: PerformancePanelProps,
) {
  const status = useSignal<MonitoringStatusResponse | null>(null);
  const loading = useSignal(true);
  const iframeLoaded = useSignal(false);
  const error = useSignal<string | null>(null);

  useEffect(() => {
    if (!IS_BROWSER) return;

    apiGet<MonitoringStatusResponse>("/v1/monitoring/status")
      .then((res) => {
        status.value = res.data;
      })
      .catch((err) => {
        error.value = err.message ?? "Failed to check monitoring status";
      })
      .finally(() => {
        loading.value = false;
      });
  }, []);

  if (!IS_BROWSER) return null;

  if (loading.value) {
    return (
      <div class="flex items-center justify-center p-12">
        <div class="h-6 w-6 animate-spin rounded-full border-2 border-slate-300 border-t-brand" />
      </div>
    );
  }

  if (error.value) {
    return (
      <div class="p-12 text-center text-sm text-slate-400 dark:text-slate-500">
        <p class="text-lg font-medium text-slate-500 dark:text-slate-400">
          Monitoring Error
        </p>
        <p class="mt-1">{error.value}</p>
      </div>
    );
  }

  // Monitoring not configured
  if (!status.value?.prometheus.available && !status.value?.grafana.available) {
    return (
      <div class="p-12 text-center text-sm text-slate-400 dark:text-slate-500">
        <p class="text-lg font-medium text-slate-500 dark:text-slate-400">
          Monitoring Not Configured
        </p>
        <p class="mt-2">
          Prometheus and Grafana were not detected in this cluster.
        </p>
        <a
          href="/monitoring"
          class="mt-3 inline-block text-brand hover:underline"
        >
          View monitoring status
        </a>
      </div>
    );
  }

  // No dashboard for this resource type
  const dashboard = DASHBOARD_MAP[kind];
  if (!dashboard) {
    return (
      <div class="p-12 text-center text-sm text-slate-400 dark:text-slate-500">
        <p class="text-lg font-medium text-slate-500 dark:text-slate-400">
          No Metrics Dashboard
        </p>
        <p class="mt-1">
          Metrics dashboards are not available for this resource type.
        </p>
        {status.value?.prometheus.available && (
          <a
            href="/monitoring/prometheus"
            class="mt-3 inline-block text-brand hover:underline"
          >
            Run a custom PromQL query
          </a>
        )}
      </div>
    );
  }

  // Grafana not available but Prometheus is
  if (!status.value?.grafana.available) {
    return (
      <div class="p-12 text-center text-sm text-slate-400 dark:text-slate-500">
        <p class="text-lg font-medium text-slate-500 dark:text-slate-400">
          Grafana Not Available
        </p>
        <p class="mt-2">
          Prometheus is available but Grafana was not detected. Dashboard embeds
          require Grafana.
        </p>
        <a
          href="/monitoring/prometheus"
          class="mt-3 inline-block text-brand hover:underline"
        >
          Use Prometheus query interface instead
        </a>
      </div>
    );
  }

  // Build iframe URL
  const params = new URLSearchParams({
    orgId: "1",
    kiosk: "1",
    refresh: "30s",
  });
  if (namespace) params.set(`var-namespace`, namespace);
  params.set(`var-${dashboard.varName}`, name);

  const iframeSrc =
    `/api/v1/monitoring/grafana/proxy/d-solo/${dashboard.uid}/overview?${params}`;

  return (
    <div class="relative min-h-[400px]">
      {!iframeLoaded.value && (
        <div class="absolute inset-0 flex items-center justify-center bg-white dark:bg-slate-900">
          <div class="flex flex-col items-center gap-2">
            <div class="h-6 w-6 animate-spin rounded-full border-2 border-slate-300 border-t-brand" />
            <span class="text-sm text-slate-400">Loading dashboard...</span>
          </div>
        </div>
      )}
      <iframe
        src={iframeSrc}
        class="h-[600px] w-full border-0"
        onLoad={() => {
          iframeLoaded.value = true;
        }}
        title={`${kind} metrics dashboard`}
        sandbox="allow-scripts allow-same-origin"
      />
    </div>
  );
}
