TAG=$(git rev-parse --short head)

# xDS
GOOS=linux GOARCH=arm64 go build -o ../example/bin/xds ../
docker build --platform=linux/arm64 -t demo-xds -f 3-xds.Dockerfile ..
IMAGE_XDS=localhost:5001/demo-xds:$TAG
docker tag demo-xds $IMAGE_XDS
docker push $IMAGE_XDS

# xDS = regular client
IMAGE_CLIENT_XDS=localhost:5001/demo-client:$TAG

# xDS server:
cat <<EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: default
  name: endpointslices-reader
rules:
- apiGroups: ["discovery.k8s.io"]
  resources: ["endpointslices"]
  verbs: ["get", "watch", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: endpointslices-reader
rules:
- apiGroups: ["discovery.k8s.io"]
  resources: ["endpointslices"]
  verbs: ["get", "watch", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: endpointslices-reader
subjects:
- kind: ServiceAccount
  name: default
  namespace: default
roleRef:
  kind: ClusterRole
  name: endpointslices-reader
  apiGroup: rbac.authorization.k8s.io
---

apiVersion: apps/v1
kind: Deployment
metadata:
  labels: { app: demo-xds }
  name: demo-xds
spec:
  replicas: 1
  selector:
    matchLabels: { app: demo-xds }
  template:
    metadata:
      labels: { app: demo-xds }
    spec:
      containers:
      - image: $IMAGE_XDS
        name: demo-xds
---
apiVersion: v1
kind: Service
metadata:
  labels: { app: demo-xds }
  name: demo-xds
spec:
  ports:
  - port: 9000
    protocol: TCP
    targetPort: 9000
  selector: { app: demo-xds }
EOF

# xDS client:
kubectl create configmap demo-client-xds -o yaml --dry-run=client --from-file=bootstrap.json=./3-bootstrap.json | kubectl apply -f -
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: demo-client-xds
  name: demo-client-xds
spec:
  replicas: 2
  selector:
    matchLabels:
      app: demo-client-xds
  template:
    metadata:
      labels:
        app: demo-client-xds
    spec:
      containers:
      - image: $IMAGE_CLIENT_XDS
        name: demo-client-xds
        env:
        # - name: JAEGER_TRACE_URL
        #  value: http://jaeger:14268/api/traces
        - name: UPSTREAM_HOST
          value: xds:///demo-server-headless
        - name: DURATION
          value: 500ms
        - name: GRPC_XDS_BOOTSTRAP
          value: /etc/config/bootstrap.json
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 1
          periodSeconds: 1
          failureThreshold: 1
        volumeMounts:
        - name: bootstrap
          mountPath: /etc/config
      volumes:
        - name: bootstrap
          configMap: { name: demo-client-xds }
EOF