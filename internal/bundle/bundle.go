package bundle

type Component struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Namespace string `json:"namespace"`
	Manifest  string `json:"manifest"`
}

var Components = []Component{
	{Name: "pipelines", Version: "v1.14.0", Namespace: "tekton-pipelines", Manifest: "https://github.com/tektoncd/pipeline/releases/download/v1.14.0/release.yaml"},
	{Name: "triggers", Version: "v0.36.0", Namespace: "tekton-pipelines", Manifest: "https://github.com/tektoncd/triggers/releases/download/v0.36.0/release.yaml"},
	{Name: "chains", Version: "v0.28.0", Namespace: "tekton-chains", Manifest: "https://github.com/tektoncd/chains/releases/download/v0.28.0/release.yaml"},
	{Name: "results", Version: "v0.19.0", Namespace: "tekton-pipelines", Manifest: "https://github.com/tektoncd/results/releases/download/v0.19.0/release.yaml"},
	{Name: "pipelines-as-code", Version: "v0.49.0", Namespace: "pipelines-as-code", Manifest: "https://github.com/openshift-pipelines/pipelines-as-code/releases/download/v0.49.0/release.k8s.yaml"},
}

const TKNVersion = "v0.45.0"

func Reverse() []Component {
	result := make([]Component, len(Components))
	for i := range Components {
		result[i] = Components[len(Components)-1-i]
	}
	return result
}
