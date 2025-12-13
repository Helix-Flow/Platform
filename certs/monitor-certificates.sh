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
