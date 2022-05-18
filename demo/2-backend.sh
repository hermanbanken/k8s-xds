GOOS=linux GOARCH=arm64 go build -o ../example/bin/server ../example/server
docker build --platform=linux/arm64 -t demo-server -f 2-backend.Dockerfile ..
IMAGE=demo-server

## Without Apple M1 Silicon:
# kind --name=demo load docker-image demo-server
# docker exec -it $(docker ps) crictl images
# kubectl create deployment demo-server --image demo-server --replicas=1

## With Apple M1 Silicon:
docker tag demo-server localhost:5001/demo-server
docker push localhost:5001/demo-server
IMAGE=localhost:5001/demo-server

# Both:
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  labels: { app: demo-server }
  name: demo-server
spec:
  replicas: 1
  selector:
    matchLabels: { app: demo-server }
  template:
    metadata:
      labels: { app: demo-server }
    spec:
      containers:
      - image: $IMAGE
        name: demo-xds
        envFrom: [configMapRef: { name: jaeger }]
---
apiVersion: v1
kind: Service
metadata:
  labels: { app: demo-server }
  name: demo-server
spec:
  clusterIP: None
  ports:
  - port: 9090
    protocol: TCP
    targetPort: 9090
  selector: { app: demo-server }
EOF
