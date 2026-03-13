import { define } from "@/utils.ts";
import ResourceTable from "@/islands/ResourceTable.tsx";

export default define.page(function StatefulSetsPage() {
  return <ResourceTable kind="statefulsets" title="StatefulSets" />;
});
