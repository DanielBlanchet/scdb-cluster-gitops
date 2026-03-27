# Create webts ssl certificate / secret

env=prod
env=dev

site=webts.${env}.home.dblanchet.net
name=webts-web-${env}-tls
email='daniel@dblanchet.net'
namespace=webts

cd /tmp
openssl req -newkey rsa:4096 -x509 -sha256 -days `expr 365 \* 5` -nodes -out $site.crt -keyout $site.key -subj "/C=CA/ST=Quebec/L=La Prairie/O=SCDB inc./OU=Technologie/CN=$site/emailAddress=$email"

kubectl create secret tls ${name} \
  --cert=./${site}.crt \
  --key=./${site}.key \
  -n $namespace

# Secrete required

kubectl create -f apps/WebTs/base/Secrets/db-connection-info.yml
kubectl create -f apps/WebTs/base/Secrets/my-cnf-backup.yml
kubectl create -f apps/WebTs/base/Secrets/webts-config-cfg.yml

# DB create/restore

kubectl exec -it mysql-db-0 -- mysql -u root -p
create user webts identified by 'my_pass';
GRANT PROCESS ON *.* TO `webts`@`%`;
GRANT ALL PRIVILEGES ON `TimeSheets`.* TO `webts`@`%`;
create schema TimeSheets;

# Recover db from latest backup.
