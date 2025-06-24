# Deployment Files Overview

This directory contains comprehensive deployment and testing resources for the TrueNAS Storage Monitor.

## 📁 File Structure

```
├── DEPLOYMENT_GUIDE.md     # Complete deployment guide
├── quick-start.sh          # Interactive setup script  
├── test-ci.sh             # Automated CI/CD testing
├── python/
│   ├── DEMO.md            # Usage examples and demos
│   ├── test_snapshot_functionality.py  # Demo script
│   └── test-config.yaml   # Sample configuration
└── container/             # Container build files
```

## 🚀 Quick Start Options

### Option 1: Interactive Setup (Recommended for first-time users)
```bash
git clone https://github.com/tomazb/kubernetes-truenas-democratic-tool.git
cd kubernetes-truenas-democratic-tool
./quick-start.sh
```

### Option 2: Manual Setup
```bash
cd python/
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
pip install -e .
truenas-monitor --help
```

### Option 3: Container
```bash
podman build -t truenas-monitor -f container/Containerfile.cli .
podman run -it --rm truenas-monitor:latest --help
```

### Option 4: CI/CD Testing
```bash
./test-ci.sh  # Automated testing without external dependencies
```

## 📖 Documentation

1. **[DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md)** - Complete deployment guide
   - Prerequisites and setup
   - Configuration options
   - Production deployment
   - Troubleshooting
   - Monitoring and maintenance

2. **[python/DEMO.md](python/DEMO.md)** - Usage examples
   - Command examples
   - Output samples
   - Integration patterns

## 🧪 Testing

### Quick Functionality Test
```bash
cd python/
python test_snapshot_functionality.py
```

### CI/CD Test Suite
```bash
./test-ci.sh
```

### Manual Testing
```bash
# Create test config
cat > config.yaml << EOF
openshift:
  namespace: democratic-csi
truenas:
  url: https://your-truenas.example.com
  username: admin
  password: your-password
  verify_ssl: false
EOF

# Test commands
truenas-monitor --config config.yaml validate
truenas-monitor --config config.yaml snapshots --health
```

## 🔧 Configuration

### Minimal Configuration
```yaml
openshift:
  namespace: democratic-csi
truenas:
  url: https://truenas.example.com
  username: admin
  password: password
```

### Production Configuration
```yaml
openshift:
  namespace: democratic-csi
  storage_class: democratic-csi-nfs
truenas:
  url: https://truenas.example.com
  username: truenas-monitor
  api_key: ${TRUENAS_API_KEY}
  verify_ssl: true
monitoring:
  interval: 300
  thresholds:
    snapshot_age_days: 30
alerts:
  enabled: true
```

## 🎯 Key Commands

```bash
# Health and validation
truenas-monitor validate
truenas-monitor snapshots --health

# Analysis
truenas-monitor analyze
truenas-monitor snapshots --analysis

# Monitoring
truenas-monitor orphans
truenas-monitor snapshots --orphaned

# Different output formats
truenas-monitor orphans --format json
truenas-monitor snapshots --health --format yaml
```

## 🐳 Container Deployment

### Local Testing
```bash
podman run -it --rm \
  -v ~/.kube:/root/.kube:ro \
  -v ./config.yaml:/app/config.yaml:ro \
  truenas-monitor:latest validate
```

### Kubernetes Deployment
See [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md) for complete Kubernetes manifests.

## ⚠️ Prerequisites

- **Python 3.10+** (for Python installation)
- **Kubernetes/OpenShift cluster** with democratic-csi
- **TrueNAS Scale** with API access
- **Network connectivity** between monitoring location and both systems

## 🔍 Troubleshooting

### Common Issues

1. **Permission denied**: Check RBAC configuration
2. **Connection refused**: Verify network connectivity and credentials
3. **Import errors**: Ensure all dependencies are installed
4. **Configuration errors**: Validate YAML syntax and required fields

### Debug Commands
```bash
truenas-monitor --log-level debug validate
truenas-monitor --config config.yaml validate --verbose
```

### Get Help
- Check [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md) troubleshooting section
- Review configuration examples
- Test with demo script first

## 📊 Success Indicators

After successful setup, you should see:

✅ Configuration validation passes  
✅ Demo functionality test completes  
✅ Basic commands return data  
✅ No import or dependency errors  

## 🎉 Next Steps

1. **Start with demo**: `python test_snapshot_functionality.py`
2. **Test connectivity**: `truenas-monitor validate`
3. **Run health check**: `truenas-monitor snapshots --health`
4. **Review DEMO.md** for comprehensive usage examples
5. **Plan production deployment** using DEPLOYMENT_GUIDE.md

For questions or issues, refer to the troubleshooting section in the deployment guide.