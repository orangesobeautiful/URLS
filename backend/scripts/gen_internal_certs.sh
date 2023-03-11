#!/bin/bash

RootDir=$( cd "$(dirname $0)/.." && pwd)
CertDir=$RootDir/server-data/certs

# CA certs
openssl ecparam -out $CertDir/ca.key -name prime256v1 -genkey
openssl req -new -sha256 -key $CertDir/ca.key -out $CertDir/ca.csr \
    -subj "/C=US/ST=Some-State/L=city/O=urls ca, Inc./OU=IT/CN=ca"
openssl x509 -req -sha256 -days 365000 -in $CertDir/ca.csr -signkey $CertDir/ca.key -out $CertDir/ca.crt

# internal service certs
openssl ecparam -out $CertDir/srvc.key -name prime256v1 -genkey
openssl req -new -sha256 -key $CertDir/srvc.key -out $CertDir/srvc.csr \
    -subj "/C=US/ST=Some-State/L=city/O=urls ca, Inc./OU=IT/CN=srvc" 
openssl x509 -req -extfile <(printf "subjectAltName=DNS:*") -in $CertDir/srvc.csr -CA  $CertDir/ca.crt -CAkey $CertDir/ca.key -CAcreateserial -out $CertDir/srvc.crt -days 365000 -sha256 


