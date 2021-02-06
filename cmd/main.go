// (C) 2021 James Barlow - github.com/jhwbarlow
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jhwbarlow/kube-node-state/pkg/render"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	rendererEnvVar = "KUBE_NODE_STATE_RENDERER"
)

const initialListNodesTimeout = 30 // seconds
const printInterval = 1 * time.Minute

type lockableNodesMap struct {
	*sync.Mutex
	Map map[string]*corev1.Node
}

func newLockableNodesMap() *lockableNodesMap {
	return &lockableNodesMap{
		Mutex: new(sync.Mutex),
		Map:   make(map[string]*corev1.Node),
	}
}

var renderer render.Renderer
var clientset *kubernetes.Clientset
var currentNodes = newLockableNodesMap()

func init() {
	log.SetOutput(os.Stderr)
}

func main() {
	if err := setup(); err != nil {
		log.Fatalf("Setup error: %v", err)
	}

	if err := initialise(); err != nil {
		log.Fatalf("Initialisation error: %v", err)
	}

	done := make(chan struct{})

	if err := run(done); err != nil {
		log.Fatalf("Run error: %v", err)
	}

	close(done)
}

func setup() error {
	var err error

	renderer, err = rendererFromConfig()
	if err != nil {
		return fmt.Errorf("creating renderer from config: %w", err)
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("obtaining in-cluster config: %w", err)
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("obtaining kubernetes clientset: %w", err)
	}

	return nil
}

func rendererFromConfig() (render.Renderer, error) {
	envVar := strings.ToLower(os.Getenv(rendererEnvVar))
	if envVar == "" {
		return nil, errors.New("environment variable " + rendererEnvVar + " is not set")
	}

	switch envVar {
	case render.JSON:
		return render.NewJSONRenderer(), nil
	case render.LogFmt:
		return render.NewLogFmtRenderer(), nil
	case render.Table:
		return render.NewTableRenderer(), nil
	}

	return nil, errors.New("environment variable " + rendererEnvVar + " is invalid")
}

func initialise() error {
	timeout := int64(initialListNodesTimeout)
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), v1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return fmt.Errorf("discovering initial nodes: %w", err)
	}

	currentNodes.Lock()
	for _, node := range nodes.Items {
		log.Printf("discovered initial node: %s", node.GetName())
		currentNodes.Map[node.GetName()] = &node
	}
	currentNodes.Unlock()

	return nil
}

func run(done <-chan struct{}) error {
	factory := informers.NewSharedInformerFactory(clientset, 0)
	informer := factory.Core().V1().Nodes().Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// Note the informer uses pointers to Node structs as the func argument
		AddFunc:    onNodeAdd,
		UpdateFunc: onNodeUpdate,
		DeleteFunc: onNodeDelete,
	})

	go informer.Run(done)

	if err := printPeriodically(done); err != nil {
		return fmt.Errorf("printing periodically: %w", err)
	}

	return nil
}

func onNodeAdd(newObject interface{}) {
	node := newObject.(*corev1.Node)

	currentNodes.Lock()
	currentNodes.Map[node.GetName()] = node
	currentNodes.Unlock()

	log.Printf("new node added to cluster: %s", node.GetName())
}

func onNodeUpdate(_, newObject interface{}) {
	node := newObject.(*corev1.Node)

	currentNodes.Lock()
	currentNodes.Map[node.GetName()] = node
	currentNodes.Unlock()

	log.Printf("existing node updated: %s", node.GetName())
}

func onNodeDelete(oldObject interface{}) {
	node := oldObject.(*corev1.Node)

	currentNodes.Lock()
	delete(currentNodes.Map, node.GetName())
	currentNodes.Unlock()

	log.Printf("node deleted from cluster: %s", node.GetName())
}

func printPeriodically(done <-chan struct{}) error {
	ticker := time.NewTicker(printInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := printNodes(); err != nil {
				return fmt.Errorf("printing nodes: %v", err)
			}
		case <-done:
			return nil
		}
	}
}

func printNodes() error {
	// Lock the current nodes map and make a snapshot copy
	currentNodes.Lock()
	snapshot := make([]*corev1.Node, 0, len(currentNodes.Map))
	for _, node := range currentNodes.Map {
		snapshot = append(snapshot, node)
	}
	currentNodes.Unlock()

	if err := renderer.Render(snapshot); err != nil {
		return fmt.Errorf("rendering nodes: %w", err)
	}

	return nil
}
