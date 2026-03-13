import { define } from "@/utils.ts";
import MonitoringStatus from "@/islands/MonitoringStatus.tsx";

export default define.page(function MonitoringPage() {
  return (
    <div class="space-y-6">
      <div>
        <h1 class="text-2xl font-bold text-slate-900 dark:text-white">
          Monitoring
        </h1>
        <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">
          Prometheus and Grafana discovery status
        </p>
      </div>
      <MonitoringStatus />
    </div>
  );
});
