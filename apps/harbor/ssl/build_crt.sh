#!/usr/bin/env bash

# give HARBOR_HOSTNAME a value or give ip_address a value depending on whether your Harbor Instance
# is exposed via FQDN (Hostname) or an IP address
# HARBOR_HOSTNAME="harbor.home.dblanchet.net"
ip_address="1.2.3.4"

duration=750

if [ -n "$HARBOR_HOSTNAME" ]; then
  harbor_hostname_value=$HARBOR_HOSTNAME
else
  harbor_hostname_value=$ip_address
fi

cd /tmp/

### Create CA certificate
openssl req \
    -newkey rsa:4096 -nodes -sha256 -keyout harbor_ca.key \
    -x509 -days $duration -out harbor_ca.crt -subj '/C=CN/ST=PEK/L=BeiJing/O=VMware/CN=HarborCA'

### Generate a Certificate Signing Request
openssl req \
    -newkey rsa:4096 -nodes -sha256 -keyout $harbor_hostname_value.key \
    -out $harbor_hostname_value.csr -subj "/C=CN/ST=PEK/L=BeiJing/O=VMware/CN=$harbor_hostname_value"

### Generate the certificate of local registry host
if [ -n "$HARBOR_HOSTNAME" ]; then
  echo subjectAltName = DNS.1:$HARBOR_HOSTNAME > extfile.cnf
else
  echo subjectAltName = IP:$ip_address > extfile.cnf
fi
openssl x509 -req -sha256 -days $duration \
    -extfile extfile.cnf \
    -CA harbor_ca.crt -CAkey harbor_ca.key -CAcreateserial \
    -in $harbor_hostname_value.csr \
    -out $harbor_hostname_value.crt
