import { define } from "@/utils.ts";
import ResourceTable from "@/islands/ResourceTable.tsx";

export default define.page(function PodsPage() {
  return <ResourceTable kind="pods" title="Pods" />;
});
