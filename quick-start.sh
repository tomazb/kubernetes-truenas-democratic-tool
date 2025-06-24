#!/bin/bash
set -e

# TrueNAS Storage Monitor - Quick Start Script
# This script helps you get started with testing the tool quickly

echo "🚀 TrueNAS Storage Monitor - Quick Start"
echo "========================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Check if we're in the right directory
if [[ ! -f "python/pyproject.toml" ]]; then
    print_error "Please run this script from the repository root directory"
    exit 1
fi

cd python

# Check Python version
python_version=$(python3 --version 2>&1 | awk '{print $2}' | cut -d. -f1-2)
required_version="3.10"

if ! python3 -c "import sys; sys.exit(0 if sys.version_info >= (3,10) else 1)" 2>/dev/null; then
    print_error "Python 3.10+ required. Found: $python_version"
    print_info "Please install Python 3.10 or later"
    exit 1
fi

print_status "Python version check passed: $python_version"

# Create virtual environment if it doesn't exist
if [[ ! -d "venv" ]]; then
    print_info "Creating Python virtual environment..."
    python3 -m venv venv
    print_status "Virtual environment created"
fi

# Activate virtual environment
print_info "Activating virtual environment..."
source venv/bin/activate

# Install dependencies
print_info "Installing dependencies..."
pip install --upgrade pip > /dev/null 2>&1
pip install -r requirements.txt > /dev/null 2>&1
pip install -e . > /dev/null 2>&1
print_status "Dependencies installed"

# Verify installation
if command -v truenas-monitor > /dev/null 2>&1; then
    print_status "TrueNAS Monitor CLI installed successfully"
else
    print_error "Installation failed - CLI not available"
    exit 1
fi

# Check for kubectl
if command -v kubectl > /dev/null 2>&1; then
    print_status "kubectl found"
    # Test kubectl connectivity
    if kubectl cluster-info > /dev/null 2>&1; then
        print_status "Kubernetes cluster accessible"
        KUBE_CONTEXT=$(kubectl config current-context)
        print_info "Current context: $KUBE_CONTEXT"
    else
        print_warning "kubectl found but cluster not accessible"
        print_info "Make sure you have a valid kubeconfig"
    fi
else
    print_warning "kubectl not found - you'll need it for Kubernetes monitoring"
fi

# Interactive configuration setup
echo ""
echo "🔧 Configuration Setup"
echo "====================="

read -p "Enter your TrueNAS hostname/IP: " TRUENAS_HOST
read -p "Enter TrueNAS username (default: admin): " TRUENAS_USER
TRUENAS_USER=${TRUENAS_USER:-admin}
read -s -p "Enter TrueNAS password: " TRUENAS_PASS
echo ""
read -p "Enter Kubernetes namespace for democratic-csi (default: democratic-csi): " K8S_NAMESPACE
K8S_NAMESPACE=${K8S_NAMESPACE:-democratic-csi}
read -p "Enter storage class name (default: democratic-csi-nfs): " STORAGE_CLASS
STORAGE_CLASS=${STORAGE_CLASS:-democratic-csi-nfs}

# Create configuration file
print_info "Creating configuration file..."
cat > test-config.yaml << EOF
openshift:
  namespace: $K8S_NAMESPACE
  storage_class: $STORAGE_CLASS
  csi_driver: org.democratic-csi.nfs

truenas:
  url: https://$TRUENAS_HOST
  username: $TRUENAS_USER
  password: $TRUENAS_PASS
  verify_ssl: false

monitoring:
  interval: 60
  thresholds:
    orphaned_pv_age_hours: 24
    pending_pvc_minutes: 60
    snapshot_age_days: 30
    pool_usage_percent: 80
    snapshot_size_gb: 100

alerts:
  enabled: false
EOF

print_status "Configuration file created: test-config.yaml"

# Test connectivity
echo ""
echo "🔍 Testing Connectivity"
echo "======================"

print_info "Testing configuration and connectivity..."
if truenas-monitor --config test-config.yaml validate; then
    print_status "Connectivity test passed!"
else
    print_warning "Connectivity test failed - but you can still run tests"
    print_info "Check your TrueNAS credentials and network connectivity"
fi

# Run demo test
echo ""
echo "🧪 Running Demo Tests"
echo "===================="

print_info "Running demonstration tests with mock data..."
if python test_snapshot_functionality.py; then
    print_status "Demo tests completed successfully!"
else
    print_error "Demo tests failed"
fi

# Show available commands
echo ""
echo "🎯 Available Commands"
echo "===================="
echo ""
print_info "Basic commands to try:"
echo "truenas-monitor --config test-config.yaml validate"
echo "truenas-monitor --config test-config.yaml orphans"
echo "truenas-monitor --config test-config.yaml analyze"
echo "truenas-monitor --config test-config.yaml snapshots --health"
echo "truenas-monitor --config test-config.yaml snapshots --analysis"
echo "truenas-monitor --config test-config.yaml snapshots --orphaned"
echo ""

print_info "Help commands:"
echo "truenas-monitor --help"
echo "truenas-monitor snapshots --help"
echo ""

print_info "Different output formats:"
echo "truenas-monitor --config test-config.yaml orphans --format json"
echo "truenas-monitor --config test-config.yaml snapshots --health --format yaml"
echo ""

# Create convenience scripts
print_info "Creating convenience scripts..."

cat > run-validation.sh << 'EOF'
#!/bin/bash
source venv/bin/activate
truenas-monitor --config test-config.yaml validate
EOF

cat > run-health-check.sh << 'EOF'
#!/bin/bash
source venv/bin/activate
echo "=== Orphaned Resources ==="
truenas-monitor --config test-config.yaml orphans
echo ""
echo "=== Storage Analysis ==="
truenas-monitor --config test-config.yaml analyze
echo ""
echo "=== Snapshot Health ==="
truenas-monitor --config test-config.yaml snapshots --health
EOF

cat > run-demo.sh << 'EOF'
#!/bin/bash
source venv/bin/activate
python test_snapshot_functionality.py
EOF

chmod +x run-validation.sh run-health-check.sh run-demo.sh

print_status "Convenience scripts created:"
print_info "  ./run-validation.sh - Test connectivity"
print_info "  ./run-health-check.sh - Run health checks"
print_info "  ./run-demo.sh - Run demo with mock data"

# Container option
echo ""
echo "🐳 Container Option"
echo "=================="
print_info "To run via container instead:"
echo "cd .."
echo "podman build -t truenas-monitor -f container/Containerfile.cli ."
echo "podman run -it --rm \\"
echo "  -v ~/.kube:/root/.kube:ro \\"
echo "  -v ./python/test-config.yaml:/app/config.yaml:ro \\"
echo "  truenas-monitor:latest validate"

# Next steps
echo ""
echo "🎉 Setup Complete!"
echo "================="
print_status "Quick start setup completed successfully!"
echo ""
print_info "Next steps:"
echo "1. Test connectivity: ./run-validation.sh"
echo "2. Run health check: ./run-health-check.sh"
echo "3. Try different commands with --help"
echo "4. Check the DEPLOYMENT_GUIDE.md for production setup"
echo "5. Review DEMO.md for comprehensive usage examples"
echo ""
print_warning "Remember to:"
echo "- Keep your TrueNAS credentials secure"
echo "- Set verify_ssl: true in production"
echo "- Review RBAC permissions for Kubernetes deployment"
echo ""
print_info "For support, check the troubleshooting section in DEPLOYMENT_GUIDE.md"
echo "Happy monitoring! 🚀"
EOF