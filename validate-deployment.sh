#!/bin/bash
# Deployment Validation Script
# Validates that the deployment is ready and functional

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_header() { echo -e "\n${BLUE}=== $1 ===${NC}"; }
print_status() { echo -e "${GREEN}✅ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
print_error() { echo -e "${RED}❌ $1${NC}"; }
print_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }

ERRORS=0
WARNINGS=0

check_error() {
    if [ $? -ne 0 ]; then
        ERRORS=$((ERRORS + 1))
        print_error "$1"
        return 1
    else
        print_status "$1"
        return 0
    fi
}

echo "🔍 TrueNAS Storage Monitor - Deployment Validation"
echo "================================================="

# Check if config file exists
print_header "Configuration Check"
if [ -f "python/test-config.yaml" ]; then
    print_status "Configuration file found"
else
    print_warning "No test configuration found - create one with quick-start.sh"
    WARNINGS=$((WARNINGS + 1))
fi

# Check Python environment
print_header "Python Environment"
cd python

if [ -d "venv" ]; then
    print_status "Virtual environment exists"
    source venv/bin/activate
else
    print_warning "No virtual environment found - run quick-start.sh"
    WARNINGS=$((WARNINGS + 1))
fi

# Check Python version
python3 --version > /dev/null 2>&1
check_error "Python 3 is available"

# Check if CLI is installed
print_header "CLI Installation"
if command -v truenas-monitor > /dev/null 2>&1; then
    VERSION=$(truenas-monitor --version 2>/dev/null || echo "unknown")
    print_status "TrueNAS Monitor CLI installed: $VERSION"
else
    print_error "TrueNAS Monitor CLI not found"
    ERRORS=$((ERRORS + 1))
fi

# Check dependencies
print_header "Dependencies"
python -c "import yaml, requests, click, rich, kubernetes" 2>/dev/null
check_error "Required Python packages available"

# Test imports
print_header "Module Imports"
python -c "
from truenas_storage_monitor.cli import cli
from truenas_storage_monitor.monitor import Monitor
from truenas_storage_monitor.k8s_client import K8sClient
from truenas_storage_monitor.truenas_client import TrueNASClient
print('All imports successful')
" 2>/dev/null
check_error "Core modules import correctly"

# Test CLI functionality
print_header "CLI Functionality"
truenas-monitor --help > /dev/null 2>&1
check_error "CLI help system works"

truenas-monitor snapshots --help > /dev/null 2>&1
check_error "Snapshots command available"

# Test demo functionality
print_header "Demo Functionality"
if [ -f "test_snapshot_functionality.py" ]; then
    print_info "Running demo functionality test..."
    python test_snapshot_functionality.py > /dev/null 2>&1
    check_error "Demo functionality test passes"
else
    print_warning "Demo test script not found"
    WARNINGS=$((WARNINGS + 1))
fi

# Check for Kubernetes connectivity
print_header "Kubernetes Connectivity"
if command -v kubectl > /dev/null 2>&1; then
    print_status "kubectl found"
    if kubectl cluster-info > /dev/null 2>&1; then
        CONTEXT=$(kubectl config current-context)
        print_status "Kubernetes cluster accessible: $CONTEXT"
        
        # Check for democratic-csi namespace
        if kubectl get namespace democratic-csi > /dev/null 2>&1; then
            print_status "democratic-csi namespace exists"
        else
            print_warning "democratic-csi namespace not found"
            WARNINGS=$((WARNINGS + 1))
        fi
        
        # Check for storage classes
        SC_COUNT=$(kubectl get storageclass | grep -c democratic-csi || echo "0")
        if [ "$SC_COUNT" -gt 0 ]; then
            print_status "democratic-csi storage classes found: $SC_COUNT"
        else
            print_warning "No democratic-csi storage classes found"
            WARNINGS=$((WARNINGS + 1))
        fi
    else
        print_warning "kubectl found but cluster not accessible"
        WARNINGS=$((WARNINGS + 1))
    fi
else
    print_warning "kubectl not found - Kubernetes features unavailable"
    WARNINGS=$((WARNINGS + 1))
fi

# Test configuration if available
print_header "Configuration Test"
if [ -f "test-config.yaml" ]; then
    print_info "Testing configuration loading..."
    truenas-monitor --config test-config.yaml validate > /dev/null 2>&1 || true
    print_info "Configuration test completed (connectivity may fail)"
else
    print_info "No test configuration available"
fi

# Check container capability
print_header "Container Support"
if command -v podman > /dev/null 2>&1; then
    print_status "Podman available for container deployment"
elif command -v docker > /dev/null 2>&1; then
    print_status "Docker available for container deployment"
else
    print_warning "No container runtime found"
    WARNINGS=$((WARNINGS + 1))
fi

# File structure validation
print_header "File Structure"
cd ..

required_files=(
    "DEPLOYMENT_GUIDE.md"
    "quick-start.sh"
    "test-ci.sh"
    "python/pyproject.toml"
    "python/requirements.txt"
    "python/truenas_storage_monitor/__init__.py"
)

for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        print_status "$file exists"
    else
        print_error "$file missing"
        ERRORS=$((ERRORS + 1))
    fi
done

# Performance test
print_header "Performance Test"
if [ -f "python/test_snapshot_functionality.py" ]; then
    cd python
    source venv/bin/activate 2>/dev/null || true
    
    start_time=$(date +%s)
    python test_snapshot_functionality.py > /dev/null 2>&1 || true
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    
    if [ $duration -le 10 ]; then
        print_status "Performance test passed: ${duration}s"
    elif [ $duration -le 30 ]; then
        print_warning "Performance acceptable: ${duration}s"
        WARNINGS=$((WARNINGS + 1))
    else
        print_error "Performance test failed: ${duration}s (too slow)"
        ERRORS=$((ERRORS + 1))
    fi
    cd ..
fi

# Summary
print_header "Validation Summary"
if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    print_status "All validation checks passed! 🎉"
    print_info "The deployment is ready for use"
elif [ $ERRORS -eq 0 ]; then
    print_warning "Validation completed with $WARNINGS warnings"
    print_info "The deployment should work but may have limited functionality"
else
    print_error "Validation failed with $ERRORS errors and $WARNINGS warnings"
    print_info "Please address the errors before proceeding"
fi

# Next steps
print_header "Next Steps"
if [ $ERRORS -eq 0 ]; then
    print_info "Try these commands to get started:"
    echo "  cd python && source venv/bin/activate"
    echo "  truenas-monitor --help"
    echo "  python test_snapshot_functionality.py"
    if [ -f "python/test-config.yaml" ]; then
        echo "  truenas-monitor --config test-config.yaml validate"
        echo "  truenas-monitor --config test-config.yaml snapshots --health"
    fi
    echo ""
    print_info "For comprehensive usage, see:"
    echo "  - DEPLOYMENT_GUIDE.md for full deployment instructions"
    echo "  - python/DEMO.md for usage examples"
else
    print_info "To fix issues:"
    echo "  - Run ./quick-start.sh for interactive setup"
    echo "  - Check DEPLOYMENT_GUIDE.md troubleshooting section"
    echo "  - Ensure all prerequisites are met"
fi

echo ""
exit $ERRORS