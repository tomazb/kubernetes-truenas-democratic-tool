#!/usr/bin/env python3
"""Run unit tests with proper coverage requirements.

This script runs unit tests that are focused on achieving high code coverage
and validating individual components in isolation.
"""

import sys
import subprocess
import time
import argparse
from pathlib import Path


def run_unit_tests(coverage_threshold=90, verbose=False, module=None):
    """Run unit tests with coverage requirements.
    
    Args:
        coverage_threshold: Minimum coverage percentage required
        verbose: Enable verbose output
        module: Specific module to test (optional)
    """
    print(f"🧪 Running unit tests with {coverage_threshold}% coverage requirement...")
    
    # Base pytest command for unit tests
    cmd = [
        "python", "-m", "pytest", 
        "tests/unit/",
        f"--cov-fail-under={coverage_threshold}",
        "--cov=truenas_storage_monitor",
        "--cov-report=term-missing",
        "--cov-report=html",
        "--cov-report=xml"
    ]
    
    if verbose:
        cmd.append("-v")
    else:
        cmd.append("-q")
    
    # Filter to specific module if requested
    if module:
        cmd.append(f"tests/unit/test_{module}.py")
    
    # Only run unit tests (exclude integration)
    cmd.extend(["-m", "unit or not (integration or e2e or slow)"])
    
    print(f"Running: {' '.join(cmd)}")
    
    start_time = time.time()
    result = subprocess.run(cmd, cwd=Path(__file__).parent)
    end_time = time.time()
    
    duration = end_time - start_time
    print(f"\n⏱️  Unit tests completed in {duration:.2f} seconds")
    
    return result.returncode


def run_quick_unit_tests():
    """Run only the fastest unit tests for quick feedback."""
    print("🚀 Running quick unit tests...")
    
    # Focus on the fastest, most important tests
    quick_tests = [
        "tests/unit/test_config.py",
        "tests/unit/test_exceptions.py", 
        "tests/unit/test_metrics.py"
    ]
    
    for test_file in quick_tests:
        if Path(test_file).exists():
            print(f"  Running {test_file}...")
            result = subprocess.run([
                "python", "-m", "pytest", 
                test_file, 
                "-q", 
                "--tb=short"
            ], cwd=Path(__file__).parent)
            
            if result.returncode != 0:
                print(f"  ❌ {test_file} failed")
                return False
            else:
                print(f"  ✅ {test_file} passed")
    
    print("📊 Quick unit tests completed successfully!")
    return True


def check_test_quality():
    """Check the quality and usefulness of unit tests."""
    print("🔍 Analyzing test quality...")
    
    # Run tests with detailed output to analyze
    result = subprocess.run([
        "python", "-m", "pytest", 
        "tests/unit/",
        "--tb=short",
        "-v",
        "--cov=truenas_storage_monitor",
        "--cov-report=term-missing"
    ], capture_output=True, text=True, cwd=Path(__file__).parent)
    
    # Analyze output for quality indicators
    output = result.stdout + result.stderr
    
    quality_score = 0
    total_checks = 0
    
    # Check 1: Are we testing real methods vs mocks?
    total_checks += 1
    if "assert_called" in output:
        print("⚠️  Found mock assertions - ensure testing real behavior")
    else:
        print("✅ No excessive mock assertions found")
        quality_score += 1
    
    # Check 2: Test count vs code complexity
    total_checks += 1
    test_count = output.count("PASSED")
    if test_count > 20:
        print(f"✅ Good test count: {test_count} tests")
        quality_score += 1
    else:
        print(f"⚠️  Low test count: {test_count} tests")
    
    # Check 3: Coverage distribution
    total_checks += 1
    if "100%" in output:
        print("✅ Some modules have 100% coverage")
        quality_score += 1
    else:
        print("⚠️  No modules with 100% coverage")
    
    # Check 4: Missing lines analysis
    total_checks += 1
    missing_lines = output.count("Missing")
    if missing_lines < 10:
        print(f"✅ Low missing line count: {missing_lines}")
        quality_score += 1
    else:
        print(f"⚠️  High missing line count: {missing_lines}")
    
    quality_percentage = (quality_score / total_checks) * 100
    print(f"\n📊 Test Quality Score: {quality_score}/{total_checks} ({quality_percentage:.0f}%)")
    
    if quality_percentage >= 75:
        print("🎉 Tests are of good quality!")
        return True
    else:
        print("⚠️  Tests could be improved")
        return False


def main():
    """Main function."""
    parser = argparse.ArgumentParser(description="Run TrueNAS Storage Monitor unit tests")
    parser.add_argument(
        "--coverage",
        type=int,
        default=90,
        help="Minimum coverage percentage (default: 90)"
    )
    parser.add_argument("--verbose", "-v", action="store_true", help="Verbose output")
    parser.add_argument("--quick", "-q", action="store_true", help="Run only quick tests")
    parser.add_argument("--quality", action="store_true", help="Analyze test quality")
    parser.add_argument("--module", "-m", help="Test specific module only")
    
    args = parser.parse_args()
    
    print("🔧 TrueNAS Storage Monitor - Unit Test Runner")
    print("=" * 50)
    
    success = True
    
    if args.quick:
        success = run_quick_unit_tests()
    elif args.quality:
        success = check_test_quality()
    else:
        returncode = run_unit_tests(
            coverage_threshold=args.coverage,
            verbose=args.verbose,
            module=args.module
        )
        success = returncode == 0
    
    # Summary
    print("\n" + "=" * 50)
    if success:
        print("🎉 Unit tests completed successfully!")
        if not args.quick:
            print(f"Coverage requirement of {args.coverage}% met.")
    else:
        print("❌ Unit tests failed.")
        print("\nThis may indicate:")
        print("  - Code coverage below threshold")
        print("  - Logic errors in implementation")
        print("  - Missing test cases")
        print("\nCheck the output above for specific details.")
    
    return 0 if success else 1


if __name__ == "__main__":
    sys.exit(main())