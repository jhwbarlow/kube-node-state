package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/table"
	corev1 "k8s.io/api/core/v1"
)

type TableRenderer struct{}

func NewTableRenderer() *TableRenderer {
	return new(TableRenderer)
}

func (tr *TableRenderer) Render(nodes []*corev1.Node) error {
	// Depending on the cluster type (cloud, bare-metal),
	// different combinations of these values will be available
	hasNodeInternalIP := false
	hasNodeExternalIP := false
	hasNodeInternalDNSName := false
	hasNodeExternalDNSName := false

	// Search all nodes to see what columns will be needed
	for _, node := range nodes {
		if hasNodeInternalIP &&
			hasNodeExternalIP &&
			hasNodeInternalDNSName &&
			hasNodeExternalDNSName {
			// If we have determined that all columns are needed,
			// no need to evaluate any more nodes
			break
		}
		for _, addr := range node.Status.Addresses {
			switch addr.Type {
			case corev1.NodeInternalIP:
				hasNodeInternalIP = true
			case corev1.NodeExternalIP:
				hasNodeExternalIP = true
			case corev1.NodeInternalDNS:
				hasNodeInternalDNSName = true
			case corev1.NodeExternalDNS:
				hasNodeExternalDNSName = true
			}
		}
	}

	// For each node, create a table row
	tableRows := make([]table.Row, 0, len(nodes))
	for _, node := range nodes {
		nodeInternalIP := "-"
		nodeExternalIP := "-"
		nodeInternalDNSName := "-"
		nodeExternalDNSName := "-"

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

		// Create newline-separated list of node conditions
		trueConditions := getTrueConditions(node.Status.Conditions)
		conditions := make([]string, 0, len(trueConditions))
		for _, condition := range trueConditions {
			if condition.Status == corev1.ConditionTrue {
				conditions = append(conditions, string(condition.Type))
			}
		}
		conditionsStr := strings.Join(conditions, ",")

		// Create newline-separated list of node taints
		taints := make([]string, 0, len(node.Spec.Taints))
		for _, taint := range node.Spec.Taints {
			taints = append(taints, taint.Key+": "+taint.Value)
		}
		taintsStr := strings.Join(taints, "\n")

		tableRow := make([]interface{}, 1, 7)
		tableRow[0] = node.GetName()
		if hasNodeInternalIP {
			tableRow = append(tableRow, nodeInternalIP)
		}
		if hasNodeExternalIP {
			tableRow = append(tableRow, nodeExternalIP)
		}
		if hasNodeInternalDNSName {
			tableRow = append(tableRow, nodeInternalDNSName)
		}
		if hasNodeExternalDNSName {
			tableRow = append(tableRow, nodeExternalDNSName)
		}

		tableRow = append(tableRow, conditionsStr, taintsStr)

		tableRows = append(tableRows, tableRow)
	}

	// Render table of nodes
	nodesTable := table.NewWriter()
	nodesTable.SetStyle(table.StyleLight)

	header := make([]interface{}, 1, 7)
	header[0] = "Hostname"
	if hasNodeInternalIP {
		header = append(header, "Internal IP")
	}
	if hasNodeExternalIP {
		header = append(header, "External IP")
	}
	if hasNodeInternalDNSName {
		header = append(header, "Internal DNS Name")
	}
	if hasNodeExternalDNSName {
		header = append(header, "Internal DNS Name")
	}

	header = append(header, "Conditions", "Taints")

	nodesTable.AppendHeader(table.Row(header))
	nodesTable.AppendRows(tableRows)
	nodesTable.AppendFooter(table.Row{"Total", len(tableRows)})

	fmt.Printf("Nodes at %s:\n", time.Now().Format(time.RFC3339))
	fmt.Println(nodesTable.Render())

	return nil
}
