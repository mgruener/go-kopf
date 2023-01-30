package main

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/mgruener/go-kopf/pkg/kopf"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func main() {
	kopf.On.Create("", "v1", "ConfigMap", createHandler)
	kopf.On.Update("", "v1", "ConfigMap", updateHandler)
	kopf.ExecuteOrDie(nil)
}

func updateHandler(res *unstructured.Unstructured, patch *unstructured.Unstructured, log logr.Logger) error {
	name := res.GetName()
	namespace := res.GetNamespace()

	log.Info(fmt.Sprintf("Executed updateHandler for: %s/%s", namespace, name))

	if (name == "hello-world") && (namespace == "controller-test") {
		log.Info(fmt.Sprintf("Setting update labels on: %s/%s", namespace, name))
		patch.SetLabels(map[string]string{"update": "done"})
	}

	return nil
}

func createHandler(res *unstructured.Unstructured, patch *unstructured.Unstructured, log logr.Logger) error {
	name := res.GetName()
	namespace := res.GetNamespace()

	log.Info(fmt.Sprintf("Executed createHandler for: %s/%s", namespace, name))

	if (name == "hello-world") && (namespace == "controller-test") {
		log.Info(fmt.Sprintf("Setting create annotation on: %s/%s", namespace, name))
		patch.SetAnnotations(map[string]string{"create": "done"})
	}

	return nil
}
