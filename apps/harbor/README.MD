# Installation de harbor.

helm repo update
helm install harbor harbor/harbor -n harbor --create-namespace

# Create webts ssl certificate / secret

env=prod

site=harbor.${env}.home.dblanchet.net
name=crypto-scdb-${env}-ssl
email='daniel@dblanchet.net'
namespace=harbor

export HARBOR_HOSTNAME="harbor.prod.home.dblanchet.net"
cd ~/Projets/SCDB/scdb-k3s/
./apps/harbor/ssl/build_crt.sh

# Create secret

kubectl create secret generic harbor-ingress \
--dry-run=client \
--from-file=ca.crt=/tmp/harbor_ca.crt \
--from-file=tls.crt=/tmp/${HARBOR_HOSTNAME}.crt \
--from-file=tls.key=/tmp/${HARBOR_HOSTNAME}.key -o yaml | \
kubectl apply -f -
