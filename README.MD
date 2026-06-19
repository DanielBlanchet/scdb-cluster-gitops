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

# install traefik
# URL: https://github.com/kubernetes-sigs/gateway-api/releases/
helm repo update
helm upgrade --install traefik traefik/traefik --namespace kube-system --values ~/Projets/SCDB/Build_kube_cluster/vars/traefik.yml
helm upgrade traefik traefik/traefik 

# Check traefik crd
kubectl get crd | grep gateway

# update traefik crd
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.5.1/experimental-install.yaml
kubectl apply --server-side=true -f https://github.com/kubernetes-sigs/gateway-api/releases/download/monthly-2026.04/monthly-2026.04-install.yaml
          
# Check traefic TLSroute api version
kubectl get crd tlsroutes.gateway.networking.k8s.io \
  -o jsonpath='{range .spec.versions[*]}{.name}{" served="}{.served}{" storage="}{.storage}{"\n"}{end}'

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

# Configure GatewayAPIfargocd
cd ~/Projets/SCDB/scdb-k3s/GateWayAPI
kubeconfig create -f GatewayClass.yaml

# Configure Gateway
# dev
kubectl create -f ~/Projets/SCDB/scdb-k3s/infrastructure/overlays/dev/Gateway.yaml

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
  
helm upgrade argocd argo/argo-cd

# Add TLS CRD
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.5.1/experimental-install.yaml
kubectl get crd | grep tlsroute

# Install MetalLB

kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.14.5/config/manifests/metallb-native.yaml

kubectl apply -f infrastructure/base/MetalLB/metallb-ipaddresspool.yaml
kubectl apply -f infrastructure/base/MetalLB/metallb-l2advertisement.yaml

# Restat Core DNS
kubectl -n kube-system delete pod -l k8s-app=kube-dns

# List tous les version des CRDs.
cluster=scdb-k3s
cluster=kubernetes
cluster=docker-desktop
~/Project/SCDB/scdb-k3s/infrastructure/tools/deprecated_crd.py --deprecated-only

# Upgrade argocd

kubectl -n argocd get cm argocd-ssh-known-hosts-cm -o yaml > /tmp/argocd-ssh-known-hosts-cm.backup.yaml
kubectl -n argocd get cm argocd-ssh-known-hosts-cm \
  -o jsonpath='{.data.ssh_known_hosts}' > /tmp/ssh_known_hosts.current
  
vi values.yaml
configs:
  params:
    # Requis quand le TLS est terminé par Gateway API / Envoy / Traefik
    # Sinon argocd-server force HTTPS et le service port 80 ne répond plus correctement
    server.insecure: true
  ssh:
    knownHosts: |
      {contents of /tmp/ssh_known_hosts.current}
      
helm -n argocd upgrade argocd argo/argo-cd -f values.yaml
