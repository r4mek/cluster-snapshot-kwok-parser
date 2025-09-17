package kwok

// KwokProviderConfig is the struct to hold kwok provider config
type KwokProviderConfig struct {
	APIVersion    string            `json:"apiVersion" yaml:"apiVersion"`
	ReadNodesFrom string            `json:"readNodesFrom" yaml:"readNodesFrom"`
	Nodegroups    *NodegroupsConfig `json:"nodegroups" yaml:"nodegroups"`
	Nodes         *NodeConfig       `json:"nodes" yaml:"nodes"`
	ConfigMap     *ConfigMapConfig  `json:"configmap" yaml:"configmap"`
	Kwok          *KwokConfig       `json:"kwok" yaml:"kwok"`
	status        *GroupingConfig
}

// NodegroupsConfig defines options for creating nodegroups
type NodegroupsConfig struct {
	FromNodeLabelKey        string `json:"fromNodeLabelKey" yaml:"fromNodeLabelKey"`
	FromNodeLabelAnnotation string `json:"fromNodeLabelAnnotation" yaml:"fromNodeLabelAnnotation"`
}

// NodeConfig defines config options for the nodes
type NodeConfig struct {
	GPUConfig *GPUConfig `json:"gpuConfig" yaml:"gpuConfig"`
	SkipTaint bool       `json:"skipTaint" yaml:"skipTaint"`
}

// ConfigMapConfig allows setting the kwok provider configmap name
type ConfigMapConfig struct {
	Name string `json:"name" yaml:"name"`
	Key  string `json:"key" yaml:"key"`
}

// KwokConfig is the struct to define kwok specific config
// (needs to be implemented; currently empty)
type KwokConfig struct {
}

// GroupingConfig defines different
type GroupingConfig struct {
	groupNodesBy      string              // [annotation, label]
	key               string              // annotation or label key
	gpuLabel          string              // gpu label key
	availableGPUTypes map[string]struct{} // available gpu types
}

// GPUConfig defines GPU related config for the node
type GPUConfig struct {
	GPULabelKey       string              `json:"gpuLabelKey" yaml:"gpuLabelKey"`
	AvailableGPUTypes map[string]struct{} `json:"availableGPUTypes" yaml:"availableGPUTypes"`
}
