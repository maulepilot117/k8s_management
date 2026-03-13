import { Input } from "@/components/ui/Input.tsx";
import { Select } from "@/components/ui/Select.tsx";

interface LabelEntry {
  key: string;
  value: string;
}

interface DeploymentBasicsProps {
  name: string;
  namespace: string;
  image: string;
  replicas: number;
  labels: LabelEntry[];
  namespaces: string[];
  errors: Record<string, string>;
  onChange: (field: string, value: unknown) => void;
}

export function DeploymentBasicsStep({
  name,
  namespace,
  image,
  replicas,
  labels,
  namespaces,
  errors,
  onChange,
}: DeploymentBasicsProps) {
  const nsOptions = namespaces.map((ns) => ({ value: ns, label: ns }));

  const updateLabel = (index: number, field: "key" | "value", val: string) => {
    const updated = [...labels];
    updated[index] = { ...updated[index], [field]: val };
    onChange("labels", updated);
  };

  const addLabel = () => {
    onChange("labels", [...labels, { key: "", value: "" }]);
  };

  const removeLabel = (index: number) => {
    onChange("labels", labels.filter((_, i) => i !== index));
  };

  return (
    <div class="space-y-6 max-w-lg">
      <Input
        label="Name"
        value={name}
        onInput={(e) => onChange("name", (e.target as HTMLInputElement).value)}
        placeholder="my-deployment"
        error={errors.name}
        required
      />

      <Select
        label="Namespace"
        value={namespace}
        onChange={(e) =>
          onChange("namespace", (e.target as HTMLSelectElement).value)}
        options={nsOptions}
        error={errors.namespace}
      />

      <Input
        label="Container Image"
        value={image}
        onInput={(e) => onChange("image", (e.target as HTMLInputElement).value)}
        placeholder="nginx:1.25"
        error={errors.image}
        required
      />

      <Input
        label="Replicas"
        type="number"
        value={String(replicas)}
        onInput={(e) =>
          onChange(
            "replicas",
            parseInt((e.target as HTMLInputElement).value) ||
              0,
          )}
        min={0}
        max={1000}
        error={errors.replicas}
      />

      <div class="space-y-2">
        <label class="block text-sm font-medium text-slate-700 dark:text-slate-300">
          Labels
        </label>
        {labels.map((label, i) => (
          <div key={i} class="flex items-center gap-2">
            <Input
              value={label.key}
              onInput={(e) =>
                updateLabel(i, "key", (e.target as HTMLInputElement).value)}
              placeholder="key"
              class="flex-1"
            />
            <span class="text-slate-400">=</span>
            <Input
              value={label.value}
              onInput={(e) =>
                updateLabel(i, "value", (e.target as HTMLInputElement).value)}
              placeholder="value"
              class="flex-1"
            />
            <button
              type="button"
              onClick={() => removeLabel(i)}
              class="p-1 text-slate-400 hover:text-danger"
              title="Remove label"
            >
              <svg
                class="w-4 h-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>
        ))}
        <button
          type="button"
          onClick={addLabel}
          class="text-sm text-brand hover:text-brand/80"
        >
          + Add Label
        </button>
      </div>
    </div>
  );
}
