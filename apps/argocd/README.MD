# Install argocd
cd ~/Projets/SCDB/scdb-k3s
helm repo add argo https://argoproj.github.io/argo-helm
helm repo update

helm install argocd argo/argo-cd \
  -n argocd \
  --create-namespace \
  --values apps/argocd/base/helm/values.yaml

# Mot de passe original

kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
  
# Create webts ssl certificate / secret

env=prod
site=argocd.${env}.home.dblanchet.net
name=argocd-${env}-ssl
email='daniel@dblanchet.net'
namespace=argocd

cd /tmp
site=argocd.prod.home.dblanchet.net
email='daniel@dblanchet.net'
~/Projets/scripts/build_ssl.py -d $site --country CA --state Quebec --city "La Prairie" --organization "SCDB inc." --ou Infra --email $email

# Create secret:

kubectl create secret tls argocd-prod-tls --cert=ssl/$site.cert.pem --key=ssl/$site.key.pem -n argocd

# install route

cd ~/Projets/SCDB/scdb-k3s
kubectl create -f apps/argocd/base/ReferenceGrants/allow-gateway-to-certs.yaml
kubectl create -f apps/argocd/base/HTTPRoutes/argocd-secure-prod.yaml
