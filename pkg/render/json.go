package render

import (
	"os"

	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
)

type JSONRenderer struct {
	out zerolog.Logger
}

func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{
		out: zerolog.New(os.Stdout).With().Timestamp().Logger(),
	}
}

func (jr *JSONRenderer) Render(nodes []*corev1.Node) error {
	for _, node := range nodes {
		nodeInternalIP := ""
		nodeExternalIP := ""
		nodeInternalDNSName := ""
		nodeExternalDNSName := ""

		// Depending on the cluster type (cloud, bare-metal),
		// different combinations of these values will be available
		for _, addr := range node.Status.Addresses {
			switch addr.Type {
			case corev1.NodeInternalIP:
				nodeInternalIP = addr.Address
			case corev1.NodeExternalIP:
				nodeExternalIP = addr.Address
			case corev1.NodeInternalDNS:
				nodeInternalDNSName = addr.Address
			case corev1.NodeExternalDNS:
				nodeExternalDNSName = addr.Address
			}
		}

		// Create array of node conditions
		conditionsArray := zerolog.Arr()
		for _, condition := range getTrueConditions(node.Status.Conditions) {
			conditionsArray.Str(string(condition.Type))
		}

		// Create array of node taints
		taintsArray := zerolog.Arr()
		for _, taint := range node.Spec.Taints {
			taintsArray.Str(taint.Key + ": " + taint.Value)
		}

		jr.out.Info().
			Str("nodeInternalIP", nodeInternalIP).
			Str("nodeExternalIP", nodeExternalIP).
			Str("nodeInternalDNSName", nodeInternalDNSName).
			Str("nodeExternalDNSName", nodeExternalDNSName).
			Array("conditions", conditionsArray).
			Array("taints", taintsArray).
			Send()
	}

	return nil
}
