package kwok

import (
	corev1 "k8s.io/api/core/v1"
)

type KwokProviderTemplates struct {
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`
	Kind       string `json:"kind" yaml:"kind"`
	Metadata   `json:"metadata" yaml:"metadata"`
	Items      []corev1.Node `json:"items" yaml:"items"`
}

type Metadata struct {
	ResourceVersion string `json:"resourceVersion" yaml:"resourceVersion"`
}
