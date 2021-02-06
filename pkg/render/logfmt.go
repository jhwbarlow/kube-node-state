package render

import (
	"os"
	"strings"
	"time"

	zaplogfmt "github.com/jsternberg/zap-logfmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
)

type LogFmtRenderer struct {
	out *zap.Logger
}

func NewLogFmtRenderer() *LogFmtRenderer {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.MessageKey = zapcore.OmitKey
	encoderConfig.EncodeTime = func(ts time.Time, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(ts.UTC().Format(time.RFC3339))
	}

	return &LogFmtRenderer{
		out: zap.New(zapcore.NewCore(zaplogfmt.NewEncoder(encoderConfig), os.Stdout, zapcore.InfoLevel)),
	}
}

func (lfr *LogFmtRenderer) Render(nodes []*corev1.Node) error {
	for _, node := range nodes {
		fields := make([]zapcore.Field, 0, 6)

		// Depending on the cluster type (cloud, bare-metal),
		// different combinations of these values will be available
		for _, addr := range node.Status.Addresses {
			switch addr.Type {
			case corev1.NodeInternalIP:
				fields = append(fields, zap.String("node_internal_ip", addr.Address))
			case corev1.NodeExternalIP:
				fields = append(fields, zap.String("node_external_ip", addr.Address))
			case corev1.NodeInternalDNS:
				fields = append(fields, zap.String("node_internal_dns", addr.Address))
			case corev1.NodeExternalDNS:
				fields = append(fields, zap.String("node_external_dns", addr.Address))
			}
		}

		// Create CSV list of node conditions
		trueConditions := getTrueConditions(node.Status.Conditions)
		conditions := make([]string, 0, len(trueConditions))
		for _, condition := range trueConditions {
			if condition.Status == corev1.ConditionTrue {
				conditions = append(conditions, string(condition.Type))
			}
		}
		fields = append(fields, zap.String("conditions", strings.Join(conditions, ",")))

		// Create CSV list of node taints
		if len(node.Spec.Taints) != 0 {
			taints := make([]string, 0, len(node.Spec.Taints))
			for _, taint := range node.Spec.Taints {
				taints = append(taints, taint.Key+": "+taint.Value)
			}
			fields = append(fields, zap.String("taints", strings.Join(taints, ",")))
		}

		lfr.out.Info("", fields...)
	}

	return nil
}
