import { useSignal } from "@preact/signals";
import { IS_BROWSER } from "fresh/runtime";
import { apiPost } from "@/lib/api.ts";
import { Button } from "@/components/ui/Button.tsx";
import { Input } from "@/components/ui/Input.tsx";

export default function NamespaceCreator() {
  const name = useSignal("");
  const labels = useSignal<Array<{ key: string; value: string }>>([
    { key: "", value: "" },
  ]);
  const submitting = useSignal(false);
  const error = useSignal<string | null>(null);
  const success = useSignal(false);

  if (!IS_BROWSER) {
    return (
      <div class="p-6">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          Create Namespace
        </h1>
      </div>
    );
  }

  const validate = (): string | null => {
    if (!name.value.trim()) return "Name is required";
    if (!/^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$/.test(name.value)) {
      return "Must be lowercase alphanumeric with dashes, max 63 chars";
    }
    return null;
  };

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    error.value = null;
    const err = validate();
    if (err) {
      error.value = err;
      return;
    }

    submitting.value = true;
    try {
      const nsLabels: Record<string, string> = {};
      for (const l of labels.value) {
        if (l.key.trim()) nsLabels[l.key.trim()] = l.value.trim();
      }

      await apiPost("/v1/resources/namespaces", {
        metadata: {
          name: name.value,
          labels: Object.keys(nsLabels).length > 0 ? nsLabels : undefined,
        },
      });
      success.value = true;
    } catch (err: unknown) {
      const msg = err instanceof Error
        ? err.message
        : "Failed to create namespace";
      error.value = msg;
    } finally {
      submitting.value = false;
    }
  };

  const addLabel = () => {
    labels.value = [...labels.value, { key: "", value: "" }];
  };

  const removeLabel = (idx: number) => {
    labels.value = labels.value.filter((_, i) => i !== idx);
  };

  const updateLabel = (idx: number, field: "key" | "value", val: string) => {
    labels.value = labels.value.map((l, i) =>
      i === idx ? { ...l, [field]: val } : l
    );
  };

  return (
    <div class="p-6 max-w-lg">
      <div class="flex items-center justify-between mb-6">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          Create Namespace
        </h1>
        <a
          href="/cluster/namespaces"
          class="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400"
        >
          Back to list
        </a>
      </div>

      {success.value && (
        <div class="mb-4 rounded-md bg-green-50 dark:bg-green-900/20 p-4 border border-green-200 dark:border-green-800">
          <p class="text-sm text-green-800 dark:text-green-200">
            Namespace "{name.value}" created.{" "}
            <a href="/cluster/namespaces" class="underline font-medium">
              View all namespaces
            </a>
          </p>
        </div>
      )}

      {error.value && (
        <div class="mb-4 rounded-md bg-red-50 dark:bg-red-900/20 p-4 border border-red-200 dark:border-red-800">
          <p class="text-sm text-red-800 dark:text-red-200">{error.value}</p>
        </div>
      )}

      <form onSubmit={handleSubmit} class="space-y-6">
        <Input
          label="Namespace Name"
          value={name.value}
          onInput={(e) => name.value = (e.target as HTMLInputElement).value}
          placeholder="my-namespace"
          required
        />

        <div>
          <div class="flex items-center justify-between mb-2">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Labels (optional)
            </label>
            <button
              type="button"
              onClick={addLabel}
              class="text-sm text-blue-500 hover:text-blue-700"
            >
              + Add Label
            </button>
          </div>
          {labels.value.map((label, idx) => (
            <div key={idx} class="flex items-center gap-2 mb-2">
              <input
                type="text"
                value={label.key}
                onInput={(e) =>
                  updateLabel(idx, "key", (e.target as HTMLInputElement).value)}
                placeholder="key"
                class="flex-1 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-1.5 text-sm text-gray-900 dark:text-white"
              />
              <span class="text-gray-400">=</span>
              <input
                type="text"
                value={label.value}
                onInput={(e) =>
                  updateLabel(
                    idx,
                    "value",
                    (e.target as HTMLInputElement).value,
                  )}
                placeholder="value"
                class="flex-1 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-1.5 text-sm text-gray-900 dark:text-white"
              />
              <button
                type="button"
                onClick={() =>
                  removeLabel(idx)}
                class="text-red-500 hover:text-red-700 text-sm px-2"
              >
                Remove
              </button>
            </div>
          ))}
        </div>

        <div class="flex items-center gap-4">
          <Button
            type="submit"
            variant="primary"
            disabled={submitting.value || success.value}
          >
            {submitting.value ? "Creating..." : "Create Namespace"}
          </Button>
          <a
            href="/cluster/namespaces"
            class="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400"
          >
            Cancel
          </a>
        </div>
      </form>
    </div>
  );
}
