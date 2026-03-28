# Create webts ssl certificate / secret

env=prod
env=dev

site=test.${env}.home.dblanchet.net
name=test-web-${env}-ssl
email='daniel@dblanchet.net'
namespace=test

cd /tmp
openssl req -newkey rsa:4096 -x509 -sha256 -days `expr 365 \* 5` -nodes -out $site.crt -keyout $site.key -subj "/C=CA/ST=Quebec/L=La Prairie/O=SCDB inc./OU=Technologie/CN=$site/emailAddress=$email"

kubectl create secret tls ${name} \
  --cert=./${site}.crt \
  --key=./${site}.key \
  -n $namespace
