package serviceaccounts

import (
	collectdv1alpha1 "github.com/aneeshkp/collectd-operator/pkg/apis/collectdmon/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//NewServiceAccountForCR method to create serviceaccount
func NewServiceAccountForCR(m *collectdv1alpha1.Collectd) *corev1.ServiceAccount {
	serviceaccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
	}

	return serviceaccount
}
