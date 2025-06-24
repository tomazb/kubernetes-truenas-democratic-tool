#!/usr/bin/env python3
"""Run integration tests with proper environment setup and reporting.

This script runs the useful integration tests that actually validate
real functionality and user workflows.
"""

import sys
import subprocess
import time
import argparse
from pathlib import Path


def run_command(cmd, timeout=60):
    """Run a command with timeout and capture output."""
    try:
        result = subprocess.run(
            cmd,
            shell=True,
            capture_output=True,
            text=True,
            timeout=timeout
        )
        return result.returncode, result.stdout, result.stderr
    except subprocess.TimeoutExpired:
        return -1, "", f"Command timed out after {timeout}s"


def check_prerequisites():
    """Check if prerequisites are available."""
    print("🔍 Checking prerequisites...")
    
    issues = []
    
    # Check Python environment
    try:
        import truenas_storage_monitor
        print("✅ TrueNAS Storage Monitor package available")
    except ImportError:
        issues.append("❌ TrueNAS Storage Monitor package not installed")
    
    # Check pytest
    try:
        import pytest
        print("✅ pytest available")
    except ImportError:
        issues.append("❌ pytest not installed")
    
    # Check kubectl (optional)
    returncode, stdout, stderr = run_command("kubectl version --client", timeout=10)
    if returncode == 0:
        print("✅ kubectl available - Kubernetes tests will run")
    else:
        print("⚠️  kubectl not available - Kubernetes tests will be skipped")
    
    # Check if we have a cluster
    returncode, stdout, stderr = run_command("kubectl cluster-info", timeout=10)
    if returncode == 0:
        print("✅ Kubernetes cluster accessible")
    else:
        print("⚠️  No Kubernetes cluster - some tests will be skipped")
    
    return issues


def run_integration_tests(test_category=None, verbose=False):
    """Run integration tests."""
    print(f"\n🧪 Running integration tests...")
    
    # Base pytest command
    cmd = ["python", "-m", "pytest", "tests/integration/"]
    
    if verbose:
        cmd.append("-v")
    else:
        cmd.append("-q")
    
    # Add category filter if specified
    if test_category:
        if test_category == "cli":
            cmd.append("tests/integration/test_cli_workflows.py")
        elif test_category == "k8s":
            cmd.append("tests/integration/test_kubernetes_real.py")
        elif test_category == "truenas":
            cmd.append("tests/integration/test_truenas_api.py")
        elif test_category == "basic":
            cmd.append("tests/integration/test_real_functionality.py")
    
    # Add markers to skip slow tests unless requested
    cmd.extend(["-m", "not slow"])
    
    # Run without coverage failure for integration tests
    cmd.extend(["--tb=short", "--cov-fail-under=0"])
    
    print(f"Running: {' '.join(cmd)}")
    
    start_time = time.time()
    result = subprocess.run(cmd, cwd=Path(__file__).parent)
    end_time = time.time()
    
    duration = end_time - start_time
    print(f"\n⏱️  Tests completed in {duration:.2f} seconds")
    
    return result.returncode


def run_quick_smoke_tests():
    """Run quick smoke tests to verify basic functionality."""
    print("\n🚀 Running quick smoke tests...")
    
    tests = [
        ("CLI Help", "python -m truenas_storage_monitor.cli --help"),
        ("CLI Version", "python -m truenas_storage_monitor.cli --version"),
        ("Demo Script", "python test_snapshot_functionality.py"),
    ]
    
    results = []
    
    for test_name, command in tests:
        print(f"  Running {test_name}...")
        returncode, stdout, stderr = run_command(command, timeout=30)
        
        if returncode == 0:
            print(f"  ✅ {test_name} passed")
            results.append(True)
        else:
            print(f"  ❌ {test_name} failed: {stderr}")
            results.append(False)
    
    passed = sum(results)
    total = len(results)
    print(f"\n📊 Smoke tests: {passed}/{total} passed")
    
    return passed == total


def main():
    """Main function."""
    parser = argparse.ArgumentParser(description="Run TrueNAS Storage Monitor integration tests")
    parser.add_argument(
        "--category",
        choices=["all", "cli", "k8s", "truenas", "basic"],
        default="all",
        help="Test category to run"
    )
    parser.add_argument("--verbose", "-v", action="store_true", help="Verbose output")
    parser.add_argument("--quick", "-q", action="store_true", help="Run only quick tests")
    parser.add_argument("--smoke", "-s", action="store_true", help="Run only smoke tests")
    
    args = parser.parse_args()
    
    print("🔧 TrueNAS Storage Monitor - Integration Test Runner")
    print("=" * 50)
    
    # Check prerequisites
    issues = check_prerequisites()
    if issues:
        print("\n⚠️  Issues found:")
        for issue in issues:
            print(f"  {issue}")
        print("\nSome tests may not run properly.")
    
    success = True
    
    if args.smoke:
        # Run only smoke tests
        success = run_quick_smoke_tests()
    elif args.quick:
        # Run quick integration tests
        category = args.category if args.category != "all" else "basic"
        returncode = run_integration_tests(category, args.verbose)
        success = returncode == 0
    else:
        # Run smoke tests first
        smoke_success = run_quick_smoke_tests()
        
        if smoke_success:
            # Run integration tests
            category = args.category if args.category != "all" else None
            returncode = run_integration_tests(category, args.verbose)
            success = returncode == 0
        else:
            print("\n❌ Smoke tests failed, skipping integration tests")
            success = False
    
    # Summary
    print("\n" + "=" * 50)
    if success:
        print("🎉 All tests completed successfully!")
        print("\nThe TrueNAS Storage Monitor is working correctly.")
        print("You can now use it with confidence in your environment.")
    else:
        print("❌ Some tests failed.")
        print("\nThis may indicate:")
        print("  - Configuration issues")
        print("  - Network connectivity problems")
        print("  - Missing dependencies")
        print("  - Environment setup issues")
        print("\nCheck the output above for specific error details.")
    
    return 0 if success else 1


if __name__ == "__main__":
    sys.exit(main())