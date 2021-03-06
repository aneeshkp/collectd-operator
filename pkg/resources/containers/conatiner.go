package containers

import (
	"os"
	"reflect"

	collectdv1alpha1 "github.com/aneeshkp/collectd-operator/pkg/apis/collectdmon/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	log = logf.Log.WithName("Collectd_Containers")
)

//CheckCollectdContainer ...
func CheckCollectdContainer(desired *corev1.Container, actual *corev1.Container) bool {
	if desired.Image != actual.Image {
		return false
	}
	if !reflect.DeepEqual(desired.Env, actual.Env) {
		return false
	}
	if !reflect.DeepEqual(desired.Ports, actual.Ports) {
		return false
	}
	if !reflect.DeepEqual(desired.VolumeMounts, actual.VolumeMounts) {
		return false
	}
	return true
}

//ContainerForCollectd ..
func ContainerForCollectd(m *collectdv1alpha1.Collectd) corev1.Container {
	var image string
	if m.Spec.DeploymentPlan.Image != "" {
		image = m.Spec.DeploymentPlan.Image
	} else {
		image = os.Getenv("COLLECTD_IMAGE")
	}

	container := corev1.Container{
		Image: image,
		Name:  m.Name,
	}

	volumeMounts := []corev1.VolumeMount{}
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      m.Name,
		MountPath: "/opt/collectd/etc/",
	})

	container.VolumeMounts = volumeMounts
	return container

}
