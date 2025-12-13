#!/bin/bash
# Certificate generation script for HelixFlow mTLS

set -e

# Configuration
CERT_DIR="./certs"
CA_KEY="${CERT_DIR}/ca.key"
CA_CERT="${CERT_DIR}/ca.crt"
VALIDITY_DAYS=365

# Service list
SERVICES=("api-gateway" "inference-pool" "auth-service" "monitoring")

# Create certificate directory
mkdir -p "${CERT_DIR}"

echo "Generating CA certificate..."
# Generate CA private key
openssl genrsa -out "${CA_KEY}" 4096

# Generate CA certificate
openssl req -x509 -new -nodes -key "${CA_KEY}" -sha256 -days "${VALIDITY_DAYS}" \
  -out "${CA_CERT}" \
  -subj "/C=US/ST=CA/L=San Francisco/O=HelixFlow/CN=HelixFlow Root CA"

echo "CA certificate generated: ${CA_CERT}"

# Generate certificates for each service
for service in "${SERVICES[@]}"; do
  echo "Generating certificate for ${service}..."

  SERVICE_KEY="${CERT_DIR}/${service}.key"
  SERVICE_CERT="${CERT_DIR}/${service}.crt"
  SERVICE_CSR="${CERT_DIR}/${service}.csr"

  # Generate service private key
  openssl genrsa -out "${SERVICE_KEY}" 2048

  # Generate certificate signing request
  openssl req -new -key "${SERVICE_KEY}" -out "${SERVICE_CSR}" \
    -subj "/C=US/ST=CA/L=San Francisco/O=HelixFlow/CN=${service}"

  # Generate service certificate signed by CA
  openssl x509 -req -in "${SERVICE_CSR}" -CA "${CA_CERT}" -CAkey "${CA_KEY}" \
    -CAcreateserial -out "${SERVICE_CERT}" -days "${VALIDITY_DAYS}" -sha256 \
    -extfile <(printf "subjectAltName=DNS:${service},DNS:${service}.helixflow-services.svc.cluster.local")

  # Clean up CSR
  rm "${SERVICE_CSR}"

  echo "Certificate generated for ${service}: ${SERVICE_CERT}"
done

# Generate client certificate for external API access
echo "Generating client certificate..."
CLIENT_KEY="${CERT_DIR}/client.key"
CLIENT_CERT="${CERT_DIR}/client.crt"
CLIENT_CSR="${CERT_DIR}/client.csr"

openssl genrsa -out "${CLIENT_KEY}" 2048
openssl req -new -key "${CLIENT_KEY}" -out "${CLIENT_CSR}" \
  -subj "/C=US/ST=CA/L=San Francisco/O=HelixFlow/CN=API Client"

openssl x509 -req -in "${CLIENT_CSR}" -CA "${CA_CERT}" -CAkey "${CA_KEY}" \
  -CAcreateserial -out "${CLIENT_CERT}" -days "${VALIDITY_DAYS}" -sha256

rm "${CLIENT_CSR}"

echo "Client certificate generated: ${CLIENT_CERT}"

# Create certificate bundle for applications
cat "${CA_CERT}" > "${CERT_DIR}/ca-bundle.crt"

echo "Certificate generation complete!"
echo "Certificates are in: ${CERT_DIR}"
echo ""
echo "Next steps:"
echo "1. Store certificates in Kubernetes secrets"
echo "2. Configure services to use mTLS"
echo "3. Update Istio peer authentication policies"