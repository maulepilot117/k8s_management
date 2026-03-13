import type { DaemonSet, K8sResource } from "@/lib/k8s-types.ts";
import { Field, SectionHeader } from "@/components/ui/Field.tsx";
import { KeyValueTable } from "./KeyValueTable.tsx";

export function DaemonSetOverview({ resource }: { resource: K8sResource }) {
  const d = resource as DaemonSet;
  const spec = d.spec;
  const status = d.status;

  return (
    <div class="space-y-4">
      {/* Counts */}
      <div>
        <SectionHeader>Status</SectionHeader>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <Field
            label="Desired"
            value={String(status?.desiredNumberScheduled ?? 0)}
          />
          <Field
            label="Current"
            value={String(status?.currentNumberScheduled ?? 0)}
          />
          <Field label="Ready" value={String(status?.numberReady ?? 0)} />
          <Field
            label="Available"
            value={String(status?.numberAvailable ?? 0)}
          />
        </div>
      </div>

      {/* Selector */}
      {spec.selector?.matchLabels && (
        <KeyValueTable title="Selector" data={spec.selector.matchLabels} />
      )}
    </div>
  );
}
