import { define } from "@/utils.ts";
import ResourceTable from "@/islands/ResourceTable.tsx";

export default define.page(function EventsPage() {
  return <ResourceTable kind="events" title="Events" />;
});
