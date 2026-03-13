import type { K8sResource, Service } from "@/lib/k8s-types.ts";
import { Field, SectionHeader } from "@/components/ui/Field.tsx";
import { KeyValueTable } from "./KeyValueTable.tsx";

export function ServiceOverview({ resource }: { resource: K8sResource }) {
  const s = resource as Service;
  const spec = s.spec;

  return (
    <div class="space-y-4">
      {/* Summary */}
      <div>
        <SectionHeader>Summary</SectionHeader>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Field label="Type" value={spec.type} />
          <Field label="Cluster IP" value={spec.clusterIP ?? "None"} mono />
        </div>
      </div>

      {/* Ports */}
      {spec.ports && spec.ports.length > 0 && (
        <div>
          <SectionHeader>Ports</SectionHeader>
          <div class="overflow-x-auto rounded-md border border-slate-200 dark:border-slate-700">
            <table class="w-full text-sm">
              <thead>
                <tr class="border-b border-slate-200 dark:border-slate-700">
                  <th class="px-3 py-1.5 text-left text-xs font-medium text-slate-500">
                    Name
                  </th>
                  <th class="px-3 py-1.5 text-left text-xs font-medium text-slate-500">
                    Protocol
                  </th>
                  <th class="px-3 py-1.5 text-left text-xs font-medium text-slate-500">
                    Port
                  </th>
                  <th class="px-3 py-1.5 text-left text-xs font-medium text-slate-500">
                    Target Port
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-100 dark:divide-slate-700/50">
                {spec.ports.map((p, i) => (
                  <tr key={p.name ?? i}>
                    <td class="px-3 py-1.5 text-slate-700 dark:text-slate-300">
                      {p.name ?? "-"}
                    </td>
                    <td class="px-3 py-1.5 text-slate-700 dark:text-slate-300">
                      {p.protocol ?? "TCP"}
                    </td>
                    <td class="px-3 py-1.5 font-mono text-slate-700 dark:text-slate-300">
                      {p.port}
                    </td>
                    <td class="px-3 py-1.5 font-mono text-slate-700 dark:text-slate-300">
                      {p.targetPort != null ? String(p.targetPort) : "-"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Selector */}
      {spec.selector && Object.keys(spec.selector).length > 0 && (
        <KeyValueTable title="Selector" data={spec.selector} />
      )}
    </div>
  );
}
