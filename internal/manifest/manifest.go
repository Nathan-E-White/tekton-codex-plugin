package manifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Nathan-E-White/tekton-codex-plugin/internal/safety"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

type Document struct {
	Bytes     []byte                       `json:"-"`
	SHA256    string                       `json:"sha256"`
	Resources []map[string]string          `json:"resources"`
	Objects   []*unstructured.Unstructured `json:"-"`
}

var supportedTektonAPIs = map[string]bool{
	"tekton.dev/v1": true, "triggers.tekton.dev/v1beta1": true,
	"results.tekton.dev/v1alpha2": true, "chains.tekton.dev/v1alpha1": true,
	"pipelinesascode.tekton.dev/v1alpha1": true,
}

func Load(inline, path string, policy *safety.PathPolicy) (Document, error) {
	if strings.TrimSpace(inline) != "" && strings.TrimSpace(path) != "" {
		return Document{}, errors.New("provide inline YAML or a path, not both")
	}
	data := []byte(inline)
	if len(data) == 0 {
		if policy == nil {
			return Document{}, errors.New("file input is unavailable because no manifest roots are configured")
		}
		resolved, err := policy.Resolve(path)
		if err != nil {
			return Document{}, err
		}
		data, err = osReadFile(resolved)
		if err != nil {
			return Document{}, err
		}
	}
	if len(data) == 0 {
		return Document{}, errors.New("manifest YAML is required")
	}
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
	doc := Document{Bytes: data}
	for {
		object := map[string]any{}
		err := decoder.Decode(&object)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Document{}, fmt.Errorf("parse manifest: %w", err)
		}
		if len(object) == 0 {
			continue
		}
		u := &unstructured.Unstructured{Object: object}
		if err := validateObject(u); err != nil {
			return Document{}, err
		}
		doc.Objects = append(doc.Objects, u)
		doc.Resources = append(doc.Resources, map[string]string{"apiVersion": u.GetAPIVersion(), "kind": u.GetKind(), "namespace": u.GetNamespace(), "name": u.GetName()})
	}
	if len(doc.Objects) == 0 {
		return Document{}, errors.New("manifest contains no Kubernetes resources")
	}
	digest := sha256.Sum256(data)
	doc.SHA256 = hex.EncodeToString(digest[:])
	return doc, nil
}

func validateObject(object *unstructured.Unstructured) error {
	if object.GetAPIVersion() == "" || object.GetKind() == "" || object.GetName() == "" {
		return errors.New("every resource requires apiVersion, kind, and metadata.name")
	}
	if strings.EqualFold(object.GetKind(), "Secret") {
		return errors.New("Secret resources are forbidden; use credential references")
	}
	api := object.GetAPIVersion()
	if strings.Contains(api, "tekton.dev/") && !supportedTektonAPIs[api] {
		return fmt.Errorf("unsupported Tekton API %q", api)
	}
	serialized := strings.ToLower(fmt.Sprint(object.Object))
	for _, forbidden := range []string{"privatekey", "private_key", "clientsecret", "client_secret"} {
		if strings.Contains(serialized, forbidden) {
			return fmt.Errorf("manifest contains forbidden credential material field %q", forbidden)
		}
	}
	return nil
}

var osReadFile = func(path string) ([]byte, error) {
	return os.ReadFile(path)
}
