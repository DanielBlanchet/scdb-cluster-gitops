# Install argocd
helm repo add argo https://argoproj.github.io/argo-helm
helm repo update

helm install argocd argo/argo-cd \
  -n argocd \
  --create-namespace
  
# Create webts ssl certificate / secret

env=prod
site=argocd.${env}.home.dblanchet.net
name=argocd-${env}-ssl
email='daniel@dblanchet.net'
namespace=argocd

cd /tmp
openssl req -newkey rsa:4096 -x509 -sha256 -days `expr 365 \* 5` -nodes -out $site.crt -keyout $site.key -subj "/C=CA/ST=Quebec/L=La Prairie/O=SCDB inc./OU=Technologie/CN=$site/emailAddress=$email"

kubectl create secret tls ${name} \
  --cert=./${site}.crt \
  --key=./${site}.key \
  -n $namespace
