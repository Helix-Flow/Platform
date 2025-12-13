#!/bin/bash

# HelixFlow SSL Certificate Generation Script
# Generates CA, server, and client certificates with proper configuration

set -e

# Configuration
CERT_DIR="/media/milosvasic/DATA4TB/Projects/HelixFlow/Platform/certs"
COUNTRY="US"
STATE="California"
LOCALITY="San Francisco"
ORGANIZATION="HelixFlow"
ORGANIZATIONAL_UNIT="AI Platform"
COMMON_NAME="helixflow.local"
SERVER_NAME="*.helixflow.local"
CLIENT_NAME="client.helixflow.local"

# Certificate validity (in days)
VALIDITY_DAYS=365

# Create certificate directory if it doesn't exist
mkdir -p "$CERT_DIR"
cd "$CERT_DIR"

echo "🔐 Generating SSL certificates for HelixFlow Platform..."

# Clean up existing certificates
rm -f *.pem *.srl *.csr *.crt *.key *.p12

# 1. Generate CA private key
echo "📋 Generating CA private key..."
openssl genrsa -out ca-key.pem 4096

# 2. Generate CA certificate
echo "📋 Generating CA certificate..."
openssl req -new -x509 -days $VALIDITY_DAYS -key ca-key.pem -sha256 -out ca.pem -subj "/C=$COUNTRY/ST=$STATE/L=$LOCALITY/O=$ORGANIZATION/OU=$ORGANIZATIONAL_UNIT/CN=HelixFlow CA"

# 3. Generate server private key
echo "📋 Generating server private key..."
openssl genrsa -out server-key.pem 2048

# 4. Generate server certificate signing request (CSR)
echo "📋 Generating server CSR..."
openssl req -new -key server-key.pem -out server.csr -subj "/C=$COUNTRY/ST=$STATE/L=$LOCALITY/O=$ORGANIZATION/OU=$ORGANIZATIONAL_UNIT/CN=$SERVER_NAME"

# 5. Create server certificate extension configuration
cat > server-ext.conf << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = helixflow.local
DNS.2 = api.helixflow.local
DNS.3 = auth.helixflow.local
DNS.4 = inference.helixflow.local
DNS.5 = monitoring.helixflow.local
DNS.6 = grpc.helixflow.local
DNS.7 = localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

# 6. Generate server certificate
echo "📋 Generating server certificate..."
openssl x509 -req -in server.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial -out server-cert.pem -days $VALIDITY_DAYS -sha256 -extfile server-ext.conf

# 7. Generate client private key
echo "📋 Generating client private key..."
openssl genrsa -out client-key.pem 2048

# 8. Generate client certificate signing request (CSR)
echo "📋 Generating client CSR..."
openssl req -new -key client-key.pem -out client.csr -subj "/C=$COUNTRY/ST=$STATE/L=$LOCALITY/O=$ORGANIZATION/OU=$ORGANIZATIONAL_UNIT/CN=$CLIENT_NAME"

# 9. Create client certificate extension configuration
cat > client-ext.conf << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = clientAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = $CLIENT_NAME
DNS.2 = localhost
IP.1 = 127.0.0.1
EOF

# 10. Generate client certificate
echo "📋 Generating client certificate..."
openssl x509 -req -in client.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial -out client-cert.pem -days $VALIDITY_DAYS -sha256 -extfile client-ext.conf

# 11. Generate PKCS12 format for client certificates (for browsers)
echo "📋 Generating PKCS12 client certificate..."
openssl pkcs12 -export -out client-cert.p12 -inkey client-key.pem -in client-cert.pem -certfile ca.pem -passout pass:helixflow123

# 12. Create certificate bundle for nginx
echo "📋 Creating certificate bundle..."
cat server-cert.pem ca.pem > server-fullchain.pem

# 13. Set appropriate permissions
echo "📋 Setting permissions..."
chmod 600 *-key.pem
chmod 644 *.pem
chmod 644 *.p12
chmod 644 *.srl

# 14. Verify certificates
echo "📋 Verifying certificates..."
echo "Verifying server certificate against CA..."
openssl verify -CAfile ca.pem server-cert.pem

echo "Verifying client certificate against CA..."
openssl verify -CAfile ca.pem client-cert.pem

# 15. Display certificate information
echo ""
echo "📄 Certificate Information:"
echo "=========================="
echo "CA Certificate:"
openssl x509 -in ca.pem -text -noout | grep -E "(Subject:|Issuer:|Not Before:|Not After:)" | head -4

echo ""
echo "Server Certificate:"
openssl x509 -in server-cert.pem -text -noout | grep -E "(Subject:|Issuer:|Not Before:|Not After:)" | head -4

echo ""
echo "Client Certificate:"
openssl x509 -in client-cert.pem -text -noout | grep -E "(Subject:|Issuer:|Not Before:|Not After:)" | head -4

# 16. Create certificate rotation script
cat > rotate-certificates.sh << 'EOF'
#!/bin/bash

# HelixFlow Certificate Rotation Script
# Automates certificate rotation with minimal downtime

set -e

CERT_DIR="$(dirname "$0")"
BACKUP_DIR="$CERT_DIR/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Create backup directory
mkdir -p "$BACKUP_DIR"

echo "🔄 Starting certificate rotation..."

# Backup current certificates
echo "📦 Backing up current certificates..."
cp -p *.pem *.p12 *.srl "$BACKUP_DIR/$TIMESTAMP/" 2>/dev/null || true

# Generate new certificates
echo "🔐 Generating new certificates..."
./generate-certificates.sh

# Restart services (would be implemented in production)
echo "🔄 Services would be restarted here..."
# docker-compose restart nginx
# kubectl rollout restart deployment/nginx

echo "✅ Certificate rotation completed successfully!"
echo "📦 Backups stored in: $BACKUP_DIR/$TIMESTAMP"
EOF

chmod +x rotate-certificates.sh

# 17. Create certificate monitoring script
cat > monitor-certificates.sh << 'EOF'
#!/bin/bash

# HelixFlow Certificate Monitoring Script
# Checks certificate expiration and sends alerts

set -e

CERT_DIR="$(dirname "$0")"
WARNING_DAYS=30
CRITICAL_DAYS=7

# Function to check certificate expiration
check_cert() {
    local cert_file="$1"
    local cert_name="$2"
    
    if [[ ! -f "$cert_file" ]]; then
        echo "🚨 CRITICAL: Certificate $cert_name not found!"
        return 2
    fi
    
    local exp_date=$(openssl x509 -in "$cert_file" -noout -enddate | cut -d= -f2)
    local exp_timestamp=$(date -d "$exp_date" +%s)
    local current_timestamp=$(date +%s)
    local days_until_expiry=$(( (exp_timestamp - current_timestamp) / 86400 ))
    
    if [[ $days_until_expiry -lt $CRITICAL_DAYS ]]; then
        echo "🚨 CRITICAL: Certificate $cert_name expires in $days_until_expiry days ($exp_date)"
        return 2
    elif [[ $days_until_expiry -lt $WARNING_DAYS ]]; then
        echo "⚠️  WARNING: Certificate $cert_name expires in $days_until_expiry days ($exp_date)"
        return 1
    else
        echo "✅ OK: Certificate $cert_name is valid for $days_until_expiry days ($exp_date)"
        return 0
    fi
}

echo "🔍 Checking certificate expiration..."

# Check all certificates
check_cert "ca.pem" "CA Certificate"
check_cert "server-cert.pem" "Server Certificate"
check_cert "client-cert.pem" "Client Certificate"

echo "🔍 Certificate monitoring completed."
EOF

chmod +x monitor-certificates.sh

# Clean up temporary files
rm -f *.csr *.conf

echo ""
echo "✅ SSL certificates generated successfully!"
echo ""
echo "📁 Generated files:"
echo "   - ca-key.pem (CA private key)"
echo "   - ca.pem (CA certificate)"
echo "   - server-key.pem (Server private key)"
echo "   - server-cert.pem (Server certificate)"
echo "   - server-fullchain.pem (Server certificate + CA chain)"
echo "   - client-key.pem (Client private key)"
echo "   - client-cert.pem (Client certificate)"
echo "   - client-cert.p12 (Client certificate in PKCS12 format)"
echo ""
echo "🔧 Management scripts:"
echo "   - rotate-certificates.sh (Certificate rotation)"
echo "   - monitor-certificates.sh (Expiration monitoring)"
echo ""
echo "📋 Certificate validity: $VALIDITY_DAYS days"
echo "🌐 Server certificate covers: $SERVER_NAME and localhost"
echo ""
echo "⚠️  IMPORTANT: Keep the private keys secure and never share them!"
echo "📦 Store ca-key.pem in a secure location with limited access."
EOF

chmod +x generate-certificates.sh