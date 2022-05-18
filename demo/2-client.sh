TAG=$(git rev-parse --short head)

GOOS=linux GOARCH=arm64 go build -o ../example/bin/client ../example/client
docker build --platform=linux/arm64 -t demo-client -f 2-client.Dockerfile ..
IMAGE=demo-client

## Without Apple M1 Silicon:
# kind --name=demo load docker-image demo-client
# docker exec -it $(docker ps) crictl images
# kubectl create deployment demo-client --image demo-client:$TAG --replicas=1

## With Apple M1 Silicon:
docker tag demo-client localhost:5001/demo-client:$TAG
IMAGE=localhost:5001/demo-client:$TAG
docker push $IMAGE

# Both:
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  labels: { app: demo-client-headless }
  name: demo-client-headless
spec:
  replicas: 2
  selector:
    matchLabels: { app: demo-client-headless }
  template:
    metadata:
      labels: { app: demo-client-headless }
    spec:
      containers:
      - image: $IMAGE
        name: demo-client-headless
        env:
        # - name: JAEGER_TRACE_URL
        #   value: http://jaeger:14268/api/traces
        - name: UPSTREAM_HOST
          value: demo-server-headless:9090
        - name: DURATION
          value: 500ms
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 1
          periodSeconds: 1
          failureThreshold: 1
EOF

cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  labels: { app: demo-client-clusterip }
  name: demo-client-clusterip
spec:
  replicas: 2
  selector:
    matchLabels: { app: demo-client-clusterip }
  template:
    metadata:
      labels: { app: demo-client-clusterip }
    spec:
      containers:
      - image: $IMAGE
        name: demo-client-clusterip
        env:
        # - name: JAEGER_TRACE_URL
        #   value: http://jaeger:14268/api/traces
        - name: UPSTREAM_HOST
          value: demo-server-clusterip:9090
        - name: DURATION
          value: 500ms
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 1
          periodSeconds: 1
          failureThreshold: 1
EOF

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: demo-server-clusterip
spec:
  type: ClusterIP
  selector: { app: demo-server }
  ports:
  - port: 9090
    protocol: TCP
    targetPort: 9090
EOF

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: demo-server-headless
spec:
  type: ClusterIP
  clusterIP: None
  selector: { app: demo-server }
  ports:
  - port: 9090
    protocol: TCP
    targetPort: 9090
EOF
