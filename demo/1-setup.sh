#!/usr/bin/env bash
brew install kind

## For M1 Apple Silicon, continue below by creating a local registry (https://github.com/admiraltyio/admiralty/issues/148)
# https://kind.sigs.k8s.io/docs/user/local-registry/
# create registry container unless it already exists
reg_name='kind-registry'
reg_port='5001'
if [ "$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)" != 'true' ]; then
  docker run \
    -d --restart=always -p "127.0.0.1:${reg_port}:5000" --name "${reg_name}" \
    registry:2
fi

# cluster
cat <<EOF | kind create cluster --name=demo --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 38000
    hostPort: 8000
  - containerPort: 32686
    hostPort: 16686
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${reg_port}"]
    endpoint = ["http://${reg_name}:5000"]
EOF

# connect the registry to the cluster network if not already connected
if [ "$(docker inspect -f='{{json .NetworkSettings.Networks.kind}}' "${reg_name}")" = 'null' ]; then
  docker network connect "kind" "${reg_name}"
fi

# Document the local registry
# https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${reg_port}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF

kubectl create configmap jaeger -o yaml --dry-run=client | kubectl apply -f -
if false
then
  # Jaeger (https://www.jaegertracing.io/docs/1.33/deployment/#all-in-one)
  kubectl create deployment jaeger --image="jaegertracing/all-in-one:1.33" --replicas=1 --port=14268
  kubectl set env deployment/jaeger COLLECTOR_ZIPKIN_HOST_PORT=:9411
  kubectl expose deployment jaeger --port=14268 --target-port=14268 --selector='app=jaeger' # tracing ingress
  kubectl create configmap jaeger --from-literal=JAEGER_TRACE_URL=http://jaeger:14268/api/traces -o yaml --dry-run=client | kubectl apply -f -
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: jaeger-ui
spec:
  type: NodePort
  ports:
  - name: http
    nodePort: 32686
    port: 16686
  selector:
    app: jaeger
EOF

fi

# Locally:
# docker run -d --name jaeger \
#   -e COLLECTOR_ZIPKIN_HOST_PORT=:9411 \
#   -p 5775:5775/udp \
#   -p 6831:6831/udp \
#   -p 6832:6832/udp \
#   -p 5778:5778 \
#   -p 16686:16686 \
#   -p 14250:14250 \
#   -p 14268:14268 \
#   -p 14269:14269 \
#   -p 9411:9411 \
#   jaegertracing/all-in-one:1.33