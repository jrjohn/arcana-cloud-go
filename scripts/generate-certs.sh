#!/bin/bash
# Generate mTLS certificates for development/testing

set -e

CERT_DIR="${1:-./certs}"
DAYS=365
KEY_SIZE=4096

echo "Generating mTLS certificates in ${CERT_DIR}..."
mkdir -p "${CERT_DIR}"

# Generate CA private key and certificate
echo "Generating CA..."
openssl genrsa -out "${CERT_DIR}/ca.key" ${KEY_SIZE}
openssl req -new -x509 -days ${DAYS} -key "${CERT_DIR}/ca.key" \
    -out "${CERT_DIR}/ca.crt" \
    -subj "/C=US/ST=California/L=San Francisco/O=Arcana Cloud/OU=Platform/CN=Arcana CA"

# Generate server private key and CSR
echo "Generating server certificate..."
openssl genrsa -out "${CERT_DIR}/server.key" ${KEY_SIZE}
openssl req -new -key "${CERT_DIR}/server.key" \
    -out "${CERT_DIR}/server.csr" \
    -subj "/C=US/ST=California/L=San Francisco/O=Arcana Cloud/OU=Server/CN=arcana-server"

# Create server certificate extensions
cat > "${CERT_DIR}/server.ext" << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = repository-layer
DNS.3 = service-layer
DNS.4 = controller-layer
DNS.5 = *.arcana-cloud.svc.cluster.local
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

# Sign server certificate with CA
openssl x509 -req -in "${CERT_DIR}/server.csr" \
    -CA "${CERT_DIR}/ca.crt" -CAkey "${CERT_DIR}/ca.key" -CAcreateserial \
    -out "${CERT_DIR}/server.crt" -days ${DAYS} \
    -extfile "${CERT_DIR}/server.ext"

# Generate client private key and CSR
echo "Generating client certificate..."
openssl genrsa -out "${CERT_DIR}/client.key" ${KEY_SIZE}
openssl req -new -key "${CERT_DIR}/client.key" \
    -out "${CERT_DIR}/client.csr" \
    -subj "/C=US/ST=California/L=San Francisco/O=Arcana Cloud/OU=Client/CN=arcana-client"

# Create client certificate extensions
cat > "${CERT_DIR}/client.ext" << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = clientAuth
EOF

# Sign client certificate with CA
openssl x509 -req -in "${CERT_DIR}/client.csr" \
    -CA "${CERT_DIR}/ca.crt" -CAkey "${CERT_DIR}/ca.key" -CAcreateserial \
    -out "${CERT_DIR}/client.crt" -days ${DAYS} \
    -extfile "${CERT_DIR}/client.ext"

# Cleanup CSR and extension files
rm -f "${CERT_DIR}"/*.csr "${CERT_DIR}"/*.ext "${CERT_DIR}"/*.srl

echo "Certificates generated successfully:"
echo "  CA:     ${CERT_DIR}/ca.crt, ${CERT_DIR}/ca.key"
echo "  Server: ${CERT_DIR}/server.crt, ${CERT_DIR}/server.key"
echo "  Client: ${CERT_DIR}/client.crt, ${CERT_DIR}/client.key"

# Verify certificates
echo ""
echo "Verifying certificates..."
openssl verify -CAfile "${CERT_DIR}/ca.crt" "${CERT_DIR}/server.crt"
openssl verify -CAfile "${CERT_DIR}/ca.crt" "${CERT_DIR}/client.crt"

echo ""
echo "Done! To use mTLS, set the following environment variables:"
echo "  ARCANA_MTLS_ENABLED=true"
echo "  ARCANA_MTLS_CA_FILE=${CERT_DIR}/ca.crt"
echo "  ARCANA_MTLS_CERT_FILE=${CERT_DIR}/server.crt (for server)"
echo "  ARCANA_MTLS_KEY_FILE=${CERT_DIR}/server.key (for server)"
echo "  ARCANA_MTLS_CERT_FILE=${CERT_DIR}/client.crt (for client)"
echo "  ARCANA_MTLS_KEY_FILE=${CERT_DIR}/client.key (for client)"
