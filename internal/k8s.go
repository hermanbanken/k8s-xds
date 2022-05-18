package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lestrrat-go/backoff/v2"
	"go.uber.org/zap"
	v1 "k8s.io/api/discovery/v1"
	"k8s.io/api/discovery/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var p = backoff.Exponential(
	backoff.WithMinInterval(time.Second),
	backoff.WithMaxInterval(time.Minute),
	backoff.WithJitterFactor(0.05),
)

type watcher struct {
	Fn              func(ctx context.Context, lo metav1.ListOptions) (watch.Interface, error)
	resourceVersion string
}

func (w *watcher) WatchLooped(ctx context.Context, fn func(watch.Event), opt metav1.ListOptions) error {
	opt.AllowWatchBookmarks = true
	b := p.Start(ctx)
	for backoff.Continue(b) {
		err := w.Watch(ctx, func(e watch.Event) {
			// https://stackoverflow.com/questions/66080942/what-k8s-bookmark-solves
			if e.Type == watch.Bookmark {
				if es, ok := e.Object.(*v1beta1.EndpointSlice); ok {
					w.resourceVersion = es.ResourceVersion
				} else if es, ok := e.Object.(*v1.EndpointSlice); ok {
					w.resourceVersion = es.ResourceVersion
				}
				opt.ResourceVersion = w.resourceVersion
			} else {
				fn(e)
			}
		}, opt)
		if err == ctx.Err() {
			return err
		}
	}
	return ctx.Err()
}

func (w *watcher) Watch(ctx context.Context, fn func(watch.Event), opt metav1.ListOptions) error {
	it, err := w.Fn(ctx, opt)
	defer func() {
		if err != nil {
			zap.L().Warn("error in watch", zap.Error(err))
		}
	}()
	if err != nil {
		return err
	}
	defer it.Stop()
	for {
		select {
		case <-ctx.Done():
			it.Stop()
			return nil

		case event, ok := <-it.ResultChan():
			if !ok {
				return errors.New("channel closed")
			}
			fn(event)
		}
	}

}

func KubernetesEndpointWatch(ctx context.Context, fn func(watch.EventType, Slice)) error {
	m := client()
	readyz(m)

	// Somehow 'paths' does not work in 'kind'; sofar only tested to work in GKE
	registered, _ := paths(m)
	continueIfPathsUnknown := len(registered.Paths) == 0

	if continueIfPathsUnknown || registered.Has("/apis/discovery.k8s.io/v1") {
		zap.L().Debug("Using /apis/discovery.k8s.io/v1")
		w := &watcher{Fn: m.DiscoveryV1().EndpointSlices(Namespace()).Watch}
		return w.WatchLooped(ctx, func(e watch.Event) {
			if es, ok := e.Object.(*v1.EndpointSlice); ok {
				slice := Slice{}
				slice.FromV1(es)
				fn(e.Type, slice)
			}
		}, metav1.ListOptions{})
	} else if registered.Has("/apis/discovery.k8s.io/v1beta1") {
		zap.L().Debug("Using /apis/discovery.k8s.io/v1beta1")
		w := &watcher{Fn: m.DiscoveryV1beta1().EndpointSlices(Namespace()).Watch}
		return w.WatchLooped(ctx, func(e watch.Event) {
			if es, ok := e.Object.(*v1beta1.EndpointSlice); ok {
				slice := Slice{}
				slice.FromV1Beta1(es)
				fn(e.Type, slice)
			}
		}, metav1.ListOptions{})
	} else {
		return fmt.Errorf("EndpointSlices Discovery API is not supported by your cluster; supported paths: %v", registered)
	}
}

// Lists available API's in the Kubernetes API
func readyz(m *kubernetes.Clientset) (err error) {
	var payload []byte
	req := m.RESTClient().Get().AbsPath("readyz")
	payload, err = req.DoRaw(context.TODO())
	if err != nil {
		return
	}
	if !bytes.Equal(payload[0:2], []byte("ok")) {
		return fmt.Errorf("unexpected healthz response %q", string(payload))
	}
	zap.L().Debug("kubernetes ready")
	return
}

type Paths struct{ Paths []string }

func (p Paths) Has(api string) bool {
	for _, path := range p.Paths {
		if path == api {
			return true
		}
	}
	return false
}

// Lists available API's in the Kubernetes API
func paths(m *kubernetes.Clientset) (paths Paths, err error) {
	var payload []byte
	payload, err = m.RESTClient().Get().DoRaw(context.TODO())
	if err != nil {
		zap.L().Error(err.Error(), zap.Error(err))
		return
	}

	zap.L().Info("raw REST get", zap.ByteString("result", payload))
	err = json.NewDecoder(bytes.NewReader(payload)).Decode(&paths)
	return
}

var kubeFlagSet = flag.NewFlagSet("kube", flag.ExitOnError)

func client() *kubernetes.Clientset {
	var kubeconfig *string
	var defaultLocation string
	if home := homedir.HomeDir(); home != "" {
		defaultLocation = filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(defaultLocation); os.IsNotExist(err) {
			defaultLocation = ""
		}
	}
	kubeconfig = kubeFlagSet.String("kubeconfig", defaultLocation, "absolute path to the kubeconfig file")
	kubeFlagSet.Parse(os.Args[1:])

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Default to InCluster-config
	if clientset == nil {
		_, isKubernetesRuntime := os.LookupEnv("KUBERNETES_SERVICE_HOST")
		if !isKubernetesRuntime {
			zap.L().Warn("Not starting Kubernetes discovery of peers because we are not running in Kubernetes")
			return nil
		}

		// Create a kubernetes client
		// Installed using: go get k8s.io/client-go@kubernetes-1.18.0
		// https://github.com/kubernetes/client-go#how-to-use-it
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err)
		}
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			panic(err)
		}
	}

	return clientset
}

// Namespace provides some Kubernetes Magic: inside kubernetes the /var/run directory is populated with useful information
// Source: https://github.com/kubernetes/kubernetes/pull/63707#issuecomment-539648137
func Namespace() string {
	// This way assumes you've set the POD_NAMESPACE environment variable using the downward API.
	// This check has to be done first for backwards compatibility with the way InClusterConfig was originally set up
	if ns, ok := os.LookupEnv("POD_NAMESPACE"); ok {
		return ns
	}

	// Fall back to the namespace associated with the service account token, if available
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}

	return "default"
}
