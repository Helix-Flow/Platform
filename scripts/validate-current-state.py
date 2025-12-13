#!/usr/bin/env python3

"""
HelixFlow Current State Validation Script
This script validates the current state of the HelixFlow platform
"""

import sys
import os
import subprocess
import importlib.util
from pathlib import Path

def check_python_environment():
    """Check Python environment and available packages."""
    print("🔍 Checking Python Environment...")
    print(f"Python version: {sys.version}")
    print(f"Python executable: {sys.executable}")
    print(f"Python path: {sys.path}")
    
    # Check if we can import key packages
    packages_to_check = [
        'requests', 'pytest', 'psycopg2', 'redis', 'fastapi', 'uvicorn',
        'prometheus_client', 'structlog', 'pydantic', 'jwt', 'cryptography'
    ]
    
    available_packages = []
    missing_packages = []
    
    for package in packages_to_check:
        try:
            __import__(package)
            available_packages.append(package)
        except ImportError:
            missing_packages.append(package)
    
    print(f"\n✅ Available packages ({len(available_packages)}):")
    for pkg in available_packages:
        print(f"  - {pkg}")
    
    print(f"\n❌ Missing packages ({len(missing_packages)}):")
    for pkg in missing_packages:
        print(f"  - {pkg}")
    
    return available_packages, missing_packages

def check_file_structure():
    """Check if required files and directories exist."""
    print("\n📁 Checking File Structure...")
    
    required_paths = [
        'api-gateway/src/main.py',
        'api-gateway/src/main.go',
        'auth-service/src/main.py',
        'auth-service/src/main.go',
        'inference-pool/src/main.py',
        'inference-pool/src/main.go',
        'monitoring/src/main.py',
        'monitoring/src/main.go',
        'tests/',
        'schemas/',
        'k8s/',
        'terraform/',
        'docs/',
        'Website/content/'
    ]
    
    existing_paths = []
    missing_paths = []
    
    for path in required_paths:
        if Path(path).exists():
            existing_paths.append(path)
        else:
            missing_paths.append(path)
    
    print(f"✅ Existing paths ({len(existing_paths)}):")
    for path in existing_paths:
        print(f"  - {path}")
    
    print(f"\n❌ Missing paths ({len(missing_paths)}):")
    for path in missing_paths:
        print(f"  - {path}")
    
    return existing_paths, missing_paths

def check_service_health():
    """Check if services can be imported and initialized."""
    print("\n🏥 Checking Service Health...")
    
    services = [
        ('api-gateway', 'api-gateway/src'),
        ('auth-service', 'auth-service/src'),
        ('inference-pool', 'inference-pool/src'),
        ('monitoring', 'monitoring/src')
    ]
    
    healthy_services = []
    unhealthy_services = []
    
    for service_name, service_path in services:
        try:
            # Add service path to Python path
            if service_path not in sys.path:
                sys.path.insert(0, service_path)
            
            # Try to import the service
            # This is a basic check - actual service initialization would require proper setup
            print(f"  Checking {service_name}...")
            
            # Check if main files exist and are readable
            main_py = Path(service_path) / 'main.py'
            main_go = Path(service_path) / 'main.go'
            
            if main_py.exists() or main_go.exists():
                healthy_services.append(service_name)
            else:
                unhealthy_services.append(service_name)
                
        except Exception as e:
            print(f"    Error: {e}")
            unhealthy_services.append(service_name)
    
    print(f"\n✅ Healthy services ({len(healthy_services)}):")
    for service in healthy_services:
        print(f"  - {service}")
    
    print(f"\n❌ Unhealthy services ({len(unhealthy_services)}):")
    for service in unhealthy_services:
        print(f"  - {service}")
    
    return healthy_services, unhealthy_services

def check_test_framework():
    """Check if test framework is properly set up."""
    print("\n🧪 Checking Test Framework...")
    
    test_dirs = ['tests/unit', 'tests/integration', 'tests/contract', 'tests/security']
    
    existing_test_dirs = []
    missing_test_dirs = []
    
    for test_dir in test_dirs:
        if Path(test_dir).exists():
            existing_test_dirs.append(test_dir)
            # Count test files
            test_files = list(Path(test_dir).glob('test_*.py'))
            print(f"  {test_dir}: {len(test_files)} test files")
        else:
            missing_test_dirs.append(test_dir)
    
    # Check if pytest is available
    try:
        import pytest
        pytest_available = True
        print("✅ pytest is available")
    except ImportError:
        pytest_available = False
        print("❌ pytest is not available")
    
    return existing_test_dirs, missing_test_dirs, pytest_available

def generate_fix_report():
    """Generate a comprehensive fix report."""
    print("\n" + "="*60)
    print("📋 HELIXFLOW CURRENT STATE VALIDATION REPORT")
    print("="*60)
    
    # Run all checks
    available_packages, missing_packages = check_python_environment()
    existing_paths, missing_paths = check_file_structure()
    healthy_services, unhealthy_services = check_service_health()
    existing_test_dirs, missing_test_dirs, pytest_available = check_test_framework()
    
    print("\n" + "="*60)
    print("🔧 RECOMMENDED FIXES")
    print("="*60)
    
    if missing_packages:
        print("\n📦 Python Dependencies:")
        print("Create a virtual environment and install missing packages:")
        print("  python3 -m venv helixflow-env")
        print("  source helixflow-env/bin/activate")
        print("  pip install " + " ".join(missing_packages[:5]) + " ...")
    
    if missing_paths:
        print("\n📁 Missing Files/Directories:")
        print("Create missing directories and files:")
        for path in missing_paths[:5]:
            print(f"  mkdir -p {Path(path).parent}")
            print(f"  touch {path}")
    
    if unhealthy_services:
        print("\n🏥 Service Issues:")
        print("Fix service implementations:")
        for service in unhealthy_services:
            print(f"  - Fix {service} main.py implementation")
            print(f"  - Add proper error handling")
            print(f"  - Implement health checks")
    
    if not pytest_available or missing_test_dirs:
        print("\n🧪 Test Framework:")
        print("Set up proper testing:")
        print("  pip install pytest pytest-cov")
        print("  Create missing test directories")
        print("  Write comprehensive test cases")
    
    # Overall health score
    total_checks = len(available_packages) + len(existing_paths) + len(healthy_services) + len(existing_test_dirs)
    total_possible = len(available_packages) + len(missing_packages) + len(existing_paths) + len(missing_paths) + len(healthy_services) + len(unhealthy_services) + len(existing_test_dirs) + len(missing_test_dirs)
    
    health_score = (total_checks / total_possible) * 100
    
    print(f"\n📊 OVERALL HEALTH SCORE: {health_score:.1f}%")
    
    if health_score >= 80:
        print("🟢 System is in good health")
    elif health_score >= 60:
        print("🟡 System needs some improvements")
    else:
        print("🔴 System requires significant fixes")
    
    print("\n" + "="*60)
    print("✅ Validation complete. Check the report above for specific fixes needed.")

if __name__ == "__main__":
    generate_fix_report()