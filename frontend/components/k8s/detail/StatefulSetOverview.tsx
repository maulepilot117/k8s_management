import type { K8sResource, StatefulSet } from "@/lib/k8s-types.ts";
import { Field, SectionHeader } from "@/components/ui/Field.tsx";
import { KeyValueTable } from "./KeyValueTable.tsx";

export function StatefulSetOverview({ resource }: { resource: K8sResource }) {
  const s = resource as StatefulSet;
  const spec = s.spec;
  const status = s.status;
  const updateStrategy = spec.updateStrategy;

  return (
    <div class="space-y-4">
      {/* Replicas */}
      <div>
        <SectionHeader>Replicas</SectionHeader>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <Field label="Desired" value={String(spec.replicas ?? 1)} />
          <Field label="Ready" value={String(status?.readyReplicas ?? 0)} />
          <Field label="Current" value={String(status?.currentReplicas ?? 0)} />
          <Field label="Updated" value={String(status?.updatedReplicas ?? 0)} />
        </div>
      </div>

      {/* Configuration */}
      <div>
        <SectionHeader>Configuration</SectionHeader>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Field label="Service Name" value={spec.serviceName} />
          {updateStrategy && (
            <>
              <Field
                label="Update Strategy"
                value={updateStrategy.type ?? "RollingUpdate"}
              />
              {updateStrategy.rollingUpdate?.partition != null && (
                <Field
                  label="Partition"
                  value={String(updateStrategy.rollingUpdate.partition)}
                />
              )}
            </>
          )}
        </div>
      </div>

      {/* Selector */}
      {spec.selector?.matchLabels && (
        <KeyValueTable title="Selector" data={spec.selector.matchLabels} />
      )}
    </div>
  );
}
