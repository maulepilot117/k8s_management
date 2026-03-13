import { define } from "@/utils.ts";
import ResourceTable from "@/islands/ResourceTable.tsx";

export default define.page(function JobsPage() {
  return <ResourceTable kind="jobs" title="Jobs" />;
});
