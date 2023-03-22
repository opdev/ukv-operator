package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// labelsForUKV returns the labels for selecting the resources
// belonging to the given UKV resource name.
func LabelsForUKV(name string) map[string]string {
	return map[string]string{"app": "ukv", "ownerInstance": name}
}

func SetObjectMeta(name string, namespace string, labels map[string]string) metav1.ObjectMeta {
	objectMeta := metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels:    labels,
	}
	return objectMeta
}
