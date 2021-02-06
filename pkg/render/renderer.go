package render

import corev1 "k8s.io/api/core/v1"

const (
	JSON   = "json"
	LogFmt = "logfmt"
	Table  = "table"
)

type Renderer interface {
	Render(node []*corev1.Node) error
}

func getTrueConditions(conditions []corev1.NodeCondition) []corev1.NodeCondition {
	// Get only true node conditions
	trueConditions := make([]corev1.NodeCondition, 0, len(conditions))
	for _, condition := range conditions {
		if condition.Status == corev1.ConditionTrue {
			trueConditions = append(trueConditions, condition)
		}
	}

	return trueConditions
}
