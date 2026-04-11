# Recreer le container k3s.

cd ~/Projets/SCDB/scdb-k3s/docker/compose
docker-compose down
docker-compose up -d

# Download kubeconfig.yaml

docker cp scdb-k3s-test:/output/kubeconfig.yaml scdb-k3s.conf
docker cp scdb-k3s-test:/output/kubeconfig.yaml scdb-k3s-test.conf

# Setup de node worker 
kubectl label node scdb-k3s node-role.kubernetes.io/worker=worker

# set container name
container=k3s-scdb-1.34
cd docker/compose

# setup de trefick
docker cp ./traefik-config.yaml ${container}:/var/lib/rancher/k3s/server/manifests/traefik-config.yaml

# Setup harbor unsecure.
docker cp ./registries.yaml ${container}:/etc/rancher/k3s/registries.yaml

# Fix coredns
cd ~/Projets/SCDB/scdb-k3s
kubectl replace -f coredns.yaml
kubectl delete pod -n kube-system -l k8s-app=kube-dns
debugger
nslookup dns01.home.dblanchet.net

# Configure GatewayAPI
cd ~/Projets/SCDB/scdb-k3s/GateWayAPI
kubeconfig create -f GatewayClass.yaml

# Configure test.
cd ~/Projets/SCDB/scdb-k3s/Test
kubectl create ns test
kubectl config set-context scdb-k3s-test --cluster scdb-k3s --user scdb-k3s --namespace test
kubectl config use-context scdb-k3s-test
kubectl create -f default-conf.yaml
kubectl create -f nginx-conf.yaml
kubectl create -f pvc-test.yml
kubectl create -f test-web-svc.yml
kubectl create -f test-web.yml

kubectl get pod -l app=test-web -o wide
kubectl exec -it $(kubectl get pod -l app=test-web -o name) -- sh
cp /usr/share/nginx/html/index.html /var/nginx/html
debugger
curl -kv http://test-web-svc

kubectl create -f Gateway.yaml
kubectl create -f HTTPRoute.yaml

# Add argocd
helm repo add argo https://argoproj.github.io/argo-helm
helm repo update

helm install argocd argo/argo-cd \
  -n argocd \
  --create-namespace

# Add TLS CRD
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.5.1/experimental-install.yaml
kubectl get crd | grep tlsroute

# Install MetalLB

kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.14.5/config/manifests/metallb-native.yaml

kubectl apply -f infrastructure/base/MetalLB/metallb-ipaddresspool.yaml
kubectl apply -f infrastructure/base/MetalLB/metallb-l2advertisement.yaml
