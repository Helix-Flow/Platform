#!/bin/bash

set -e

echo "🔐 Generating TLS Certificates for HelixFlow"
echo "============================================="

# Certificate configuration
CERT_DIR="${CERT_DIR:-./certs}"
CA_NAME="${CA_NAME:-helixflow-ca}"
ORGANIZATION="${ORGANIZATION:-HelixFlow Inc.}"
COUNTRY="${COUNTRY:-US}"
STATE="${STATE:-California}"
LOCALITY="${LOCALITY:-San Francisco}"
VALIDITY_DAYS="${VALIDITY_DAYS:-365}"

# Service names
SERVICES=("api-gateway" "auth-service" "inference-pool" "monitoring")

# Create certificates directory
echo "Creating certificates directory..."
mkdir -p "$CERT_DIR"
cd "$CERT_DIR"

# Generate CA private key and certificate
echo "Generating Certificate Authority (CA)..."
openssl genrsa -out "${CA_NAME}-key.pem" 4096
openssl req -new -x509 -days "$VALIDITY_DAYS" -key "${CA_NAME}-key.pem" -out "${CA_NAME}.pem" \
    -subj "/C=$COUNTRY/ST=$STATE/L=$LOCALITY/O=$ORGANIZATION/CN=$CA_NAME"

# Create CA certificate chain
cat "${CA_NAME}.pem" > "${CA_NAME}-chain.pem"

echo "✅ CA certificate generated: ${CA_NAME}.pem"

# Generate server certificates for each service
for service in "${SERVICES[@]}"; do
    echo ""
    echo "Generating certificate for $service..."
    
    # Generate private key
    openssl genrsa -out "${service}-key.pem" 4096
    
    # Generate certificate signing request (CSR)
    openssl req -new -key "${service}-key.pem" -out "${service}.csr" \
        -subj "/C=$COUNTRY/ST=$STATE/L=$LOCALITY/O=$ORGANIZATION/CN=${service}.helixflow.local"
    
    # Create extensions file for SAN (Subject Alternative Names)
    cat > "${service}.ext" << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth, clientAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = ${service}.helixflow.local
DNS.2 = ${service}
DNS.3 = localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF
    
    # Sign the certificate with CA
    openssl x509 -req -in "${service}.csr" -CA "${CA_NAME}.pem" -CAkey "${CA_NAME}-key.pem" \
        -CAcreateserial -out "${service}.crt" -days "$VALIDITY_DAYS" -sha256 -extfile "${service}.ext"
    
    # Create full chain certificate
    cat "${service}.crt" "${CA_NAME}.pem" > "${service}-fullchain.crt"
    
    # Create PKCS12 format for Java applications
    openssl pkcs12 -export -out "${service}.p12" -inkey "${service}-key.pem" -in "${service}.crt" \
        -certfile "${CA_NAME}.pem" -passout pass:helixflow123
    
    # Clean up temporary files
    rm "${service}.csr" "${service}.ext"
    
    echo "✅ $service certificate generated:"
    echo "  - Private key: ${service}-key.pem"
    echo "  - Certificate: ${service}.crt"
    echo "  - Full chain: ${service}-fullchain.crt"
    echo "  - PKCS12: ${service}.p12"
done

# Generate client certificates for service-to-service communication
echo ""
echo "Generating client certificates..."
for service in "${SERVICES[@]}"; do
    echo "Generating client certificate for $service..."
    
    # Generate private key
    openssl genrsa -out "${service}-client-key.pem" 4096
    
    # Generate certificate signing request (CSR)
    openssl req -new -key "${service}-client-key.pem" -out "${service}-client.csr" \
        -subj "/C=$COUNTRY/ST=$STATE/L=$LOCALITY/O=$ORGANIZATION/CN=${service}-client.helixflow.local"
    
    # Create extensions file for client certificate
    cat > "${service}-client.ext" << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = clientAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = ${service}-client.helixflow.local
DNS.2 = localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF
    
    # Sign the client certificate with CA
    openssl x509 -req -in "${service}-client.csr" -CA "${CA_NAME}.pem" -CAkey "${CA_NAME}-key.pem" \
        -CAcreateserial -out "${service}-client.crt" -days "$VALIDITY_DAYS" -sha256 -extfile "${service}-client.ext"
    
    # Create PKCS12 format
    openssl pkcs12 -export -out "${service}-client.p12" -inkey "${service}-client-key.pem" -in "${service}-client.crt" \
        -certfile "${CA_NAME}.pem" -passout pass:helixflow123
    
    # Clean up temporary files
    rm "${service}-client.csr" "${service}-client.ext"
    
    echo "✅ $service client certificate generated:"
    echo "  - Private key: ${service}-client-key.pem"
    echo "  - Certificate: ${service}-client.crt"
    echo "  - PKCS12: ${service}-client.p12"
done

# Generate JWT signing keys
echo ""
echo "Generating JWT signing keys..."
openssl genrsa -out "jwt-private.pem" 4096
openssl rsa -in "jwt-private.pem" -pubout -out "jwt-public.pem"

# Set proper permissions
echo ""
echo "Setting file permissions..."
chmod 600 *-key.pem
chmod 600 jwt-private.pem
chmod 644 *.crt *.pem *.p12

# Create certificate bundle
cat *.crt > all-services.crt
cat *-key.pem > all-services-key.pem

# Create certificate summary
echo ""
echo "📋 Certificate Summary"
echo "====================="
echo "Certificate Authority: ${CA_NAME}.pem"
echo "Services: ${SERVICES[*]}"
echo "Certificate Directory: $CERT_DIR"
echo "Validity: $VALIDITY_DAYS days"
echo ""
echo "Generated files:"
ls -la *.pem *.crt *.p12 2>/dev/null | grep -v "^d"

echo ""
echo "🎉 TLS certificate generation completed successfully!"
echo ""
echo "Next steps:"
echo "1. Copy certificates to respective service directories"
echo "2. Update service configurations to use the certificates"
echo "3. Test mTLS connections between services"
echo "4. Set up certificate rotation mechanism"
echo ""
echo "Certificate paths for services:"
for service in "${SERVICES[@]}"; do
    echo "  $service:"
    echo "    Server cert: $CERT_DIR/${service}.crt"
    echo "    Server key: $CERT_DIR/${service}-key.pem"
    echo "    Client cert: $CERT_DIR/${service}-client.crt"
    echo "    CA cert: $CERT_DIR/${CA_NAME}.pem"
done