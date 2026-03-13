import type { Ingress, K8sResource } from "@/lib/k8s-types.ts";
import { Field, SectionHeader } from "@/components/ui/Field.tsx";

function formatBackend(backend: unknown): string {
  const b = backend as Record<string, unknown>;
  if (b.service) {
    const svc = b.service as {
      name?: string;
      port?: { number?: number; name?: string };
    };
    const port = svc.port?.number ?? svc.port?.name ?? "";
    return `${svc.name ?? "?"}:${port}`;
  }
  return JSON.stringify(backend);
}

export function IngressOverview({ resource }: { resource: K8sResource }) {
  const ing = resource as Ingress;
  const spec = ing.spec;
  const lbIngress = ing.status?.loadBalancer?.ingress;

  // Flatten rules into rows
  const rows: Array<{ host: string; path: string; backend: string }> = [];
  for (const rule of spec.rules ?? []) {
    const host = rule.host ?? "*";
    for (const p of rule.http?.paths ?? []) {
      rows.push({
        host,
        path: p.path ?? "/",
        backend: formatBackend(p.backend),
      });
    }
  }

  return (
    <div class="space-y-4">
      {/* Load Balancer */}
      {lbIngress && lbIngress.length > 0 && (
        <div>
          <SectionHeader>Load Balancer</SectionHeader>
          <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            {lbIngress.map((lb, i) => (
              <Field
                key={i}
                label={lb.ip ? "IP" : "Hostname"}
                value={lb.ip ?? lb.hostname ?? "-"}
                mono
              />
            ))}
          </div>
        </div>
      )}

      {/* Rules */}
      {rows.length > 0 && (
        <div>
          <SectionHeader>Rules</SectionHeader>
          <div class="overflow-x-auto rounded-md border border-slate-200 dark:border-slate-700">
            <table class="w-full text-sm">
              <thead>
                <tr class="border-b border-slate-200 dark:border-slate-700">
                  <th class="px-3 py-1.5 text-left text-xs font-medium text-slate-500">
                    Host
                  </th>
                  <th class="px-3 py-1.5 text-left text-xs font-medium text-slate-500">
                    Path
                  </th>
                  <th class="px-3 py-1.5 text-left text-xs font-medium text-slate-500">
                    Backend
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-100 dark:divide-slate-700/50">
                {rows.map((r, i) => (
                  <tr key={i}>
                    <td class="px-3 py-1.5 font-mono text-slate-700 dark:text-slate-300">
                      {r.host}
                    </td>
                    <td class="px-3 py-1.5 font-mono text-slate-700 dark:text-slate-300">
                      {r.path}
                    </td>
                    <td class="px-3 py-1.5 font-mono text-slate-700 dark:text-slate-300">
                      {r.backend}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* TLS */}
      {spec.tls && spec.tls.length > 0 && (
        <div>
          <SectionHeader>TLS</SectionHeader>
          <div class="overflow-x-auto rounded-md border border-slate-200 dark:border-slate-700">
            <table class="w-full text-sm">
              <thead>
                <tr class="border-b border-slate-200 dark:border-slate-700">
                  <th class="px-3 py-1.5 text-left text-xs font-medium text-slate-500">
                    Hosts
                  </th>
                  <th class="px-3 py-1.5 text-left text-xs font-medium text-slate-500">
                    Secret Name
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-100 dark:divide-slate-700/50">
                {spec.tls.map((t, i) => (
                  <tr key={i}>
                    <td class="px-3 py-1.5 font-mono text-slate-700 dark:text-slate-300">
                      {t.hosts?.join(", ") ?? "*"}
                    </td>
                    <td class="px-3 py-1.5 text-slate-700 dark:text-slate-300">
                      {t.secretName ?? "-"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
