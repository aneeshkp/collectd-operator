package selectors

import (
	"k8s.io/apimachinery/pkg/labels"
)

//Label constant
const (
	LabelAppKey = "application"

	LabelResourceKey = "collectd_cr"
)

//LabelsForCollectd ... Set labels in a map
func LabelsForCollectd(name string) map[string]string {
	return map[string]string{
		LabelAppKey:      name,
		LabelResourceKey: name,
	}
}

//ResourcesByCollectdName ... return a selector that matches resources for a Collectd resource
func ResourcesByCollectdName(name string) labels.Selector {
	set := map[string]string{
		LabelAppKey:      name,
		LabelResourceKey: name,
	}
	return labels.SelectorFromSet(set)
}
