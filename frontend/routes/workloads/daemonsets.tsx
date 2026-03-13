import { define } from "@/utils.ts";
import ResourceTable from "@/islands/ResourceTable.tsx";

export default define.page(function DaemonSetsPage() {
  return <ResourceTable kind="daemonsets" title="DaemonSets" />;
});
