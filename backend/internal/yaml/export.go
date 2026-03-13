package yaml

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	sigyaml "sigs.k8s.io/yaml"
)

// noisyAnnotations are annotations added by controllers that should be stripped
// from exported YAML to produce clean, reapply-ready manifests.
var noisyAnnotations = []string{
	"kubectl.kubernetes.io/last-applied-configuration",
	"deployment.kubernetes.io/revision",
}

// CleanForExport strips server-managed fields from an unstructured object,
// producing YAML that can be cleanly re-applied via server-side apply.
// The input object is not modified; a deep copy is returned.
func CleanForExport(obj *unstructured.Unstructured) *unstructured.Unstructured {
	clean := obj.DeepCopy()

	// Remove server-managed metadata fields
	unstructured.RemoveNestedField(clean.Object, "metadata", "uid")
	unstructured.RemoveNestedField(clean.Object, "metadata", "resourceVersion")
	unstructured.RemoveNestedField(clean.Object, "metadata", "generation")
	unstructured.RemoveNestedField(clean.Object, "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(clean.Object, "metadata", "deletionTimestamp")
	unstructured.RemoveNestedField(clean.Object, "metadata", "deletionGracePeriodSeconds")
	unstructured.RemoveNestedField(clean.Object, "metadata", "selfLink")
	unstructured.RemoveNestedField(clean.Object, "metadata", "managedFields")
	unstructured.RemoveNestedField(clean.Object, "metadata", "ownerReferences")

	// Remove status block (server-computed)
	unstructured.RemoveNestedField(clean.Object, "status")

	// Remove noisy annotations
	annotations := clean.GetAnnotations()
	if annotations != nil {
		for _, key := range noisyAnnotations {
			delete(annotations, key)
		}
		if len(annotations) == 0 {
			unstructured.RemoveNestedField(clean.Object, "metadata", "annotations")
		} else {
			clean.SetAnnotations(annotations)
		}
	}

	return clean
}

// ExportToYAML converts an unstructured object to clean, reapply-ready YAML bytes.
// Server-managed fields are stripped.
func ExportToYAML(obj *unstructured.Unstructured) ([]byte, error) {
	clean := CleanForExport(obj)
	return sigyaml.Marshal(clean.Object)
}
