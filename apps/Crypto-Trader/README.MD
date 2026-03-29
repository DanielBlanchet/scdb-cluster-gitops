# Create webts ssl certificate / secret

env=prod
env=dev

site=crypto-scdb.${env}.home.dblanchet.net
name=crypto-scdb-${env}-ssl
email='daniel@dblanchet.net'
namespace=crypto-trader

cd ~/Projets/scdb-k3s/apps/harbor/ssl
./build_crt.sh

cd /tmp
openssl req -newkey rsa:4096 -x509 -sha256 -days `expr 365 \* 5` -nodes -out $site.crt -keyout $site.key -subj "/C=CA/ST=Quebec/L=La Prairie/O=SCDB inc./OU=Technologie/CN=$site/emailAddress=$email"

kubectl create secret tls ${name} \
  --cert=./${site}.crt \
  --key=./${site}.key \
  -n $namespace
cd -

# Create secret

kubectl create -f apps/Crypto-Trader/base/Secrets/my-cnf-backup.yaml
kubectl create -f apps/Crypto-Trader/base/Secrets/mysql-db-connect.yml
kubectl create -f apps/Crypto-Trader/base/Secrets/web-config-cfg.yml
kubectl create secret generic kube-config --from-file=config=${HOME}/.kube/config.d/scdb-k3s.conf
