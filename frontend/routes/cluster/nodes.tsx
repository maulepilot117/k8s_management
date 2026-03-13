import { define } from "@/utils.ts";
import ResourceTable from "@/islands/ResourceTable.tsx";

export default define.page(function NodesPage() {
  return <ResourceTable kind="nodes" title="Nodes" clusterScoped />;
});
