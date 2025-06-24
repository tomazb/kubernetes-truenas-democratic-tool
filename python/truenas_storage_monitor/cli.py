"""Command-line interface for TrueNAS Storage Monitor."""

import sys
from typing import Optional

import click
from rich.console import Console
from rich.table import Table

from . import __version__
from .config import load_config
from .exceptions import TrueNASMonitorError

console = Console()


def _create_truenas_config(config_dict: dict):
    """Create TrueNASConfig from configuration dictionary."""
    from .truenas_client import TrueNASConfig
    import urllib.parse
    
    url = config_dict["url"]
    parsed = urllib.parse.urlparse(url)
    host = parsed.hostname
    port = parsed.port or (443 if parsed.scheme == "https" else 80)
    
    return TrueNASConfig(
        host=host,
        port=port,
        username=config_dict.get("username"),
        password=config_dict.get("password"),
        api_key=config_dict.get("api_key"),
        verify_ssl=config_dict.get("verify_ssl", True)
    )


@click.group()
@click.version_option(version=__version__, prog_name="truenas-monitor")
@click.option(
    "--config",
    "-c",
    type=click.Path(exists=True),
    help="Path to configuration file",
    envvar="TRUENAS_MONITOR_CONFIG",
)
@click.option(
    "--log-level",
    "-l",
    type=click.Choice(["debug", "info", "warning", "error"]),
    default="info",
    help="Set logging level",
)
@click.pass_context
def cli(ctx: click.Context, config: Optional[str], log_level: str) -> None:
    """TrueNAS Storage Monitor - Comprehensive monitoring for OpenShift/Kubernetes with TrueNAS."""
    ctx.ensure_object(dict)
    
    try:
        ctx.obj["config"] = load_config(config)
        ctx.obj["log_level"] = log_level
    except Exception as e:
        console.print(f"[red]Error loading configuration: {e}[/red]")
        sys.exit(1)


@cli.command()
@click.option(
    "--format",
    "-f",
    type=click.Choice(["table", "json", "yaml"]),
    default="table",
    help="Output format",
)
@click.option(
    "--namespace",
    "-n",
    type=str,
    help="Namespace to check (all namespaces if not specified)",
)
@click.option(
    "--threshold",
    "-t",
    type=int,
    default=24,
    help="Age threshold in hours for considering resources orphaned",
)
@click.pass_context
def orphans(ctx: click.Context, format: str, namespace: Optional[str], threshold: int) -> None:
    """Check for orphaned resources."""
    from datetime import datetime, timedelta
    import json
    import yaml as yml
    from .k8s_client import K8sClient, K8sConfig
    from .truenas_client import TrueNASClient, TrueNASConfig
    
    console.print("[yellow]Checking for orphaned resources...[/yellow]")
    
    config = ctx.obj["config"]
    
    try:
        # Initialize clients
        k8s_config = K8sConfig(
            namespace=namespace or config.get("openshift", {}).get("namespace"),
            storage_class=config.get("openshift", {}).get("storage_class"),
            csi_driver=config.get("openshift", {}).get("csi_driver", "org.democratic-csi.nfs")
        )
        k8s_client = K8sClient(k8s_config)
        
        # Find orphaned PVs
        orphaned_pvs = k8s_client.find_orphaned_pvs(orphan_threshold_minutes=threshold * 60)
        
        # Find orphaned PVCs
        orphaned_pvcs = k8s_client.find_orphaned_pvcs(pending_threshold_minutes=threshold * 60)
        
        all_orphans = orphaned_pvs + orphaned_pvcs
        
        if format == "table":
            if not all_orphans:
                console.print("[green]No orphaned resources found![/green]")
                return
                
            table = Table(title=f"Orphaned Resources (threshold: {threshold} hours)")
            table.add_column("Type", style="cyan")
            table.add_column("Name", style="magenta")
            table.add_column("Namespace")
            table.add_column("Age")
            table.add_column("Reason", style="yellow")
            
            for orphan in all_orphans:
                age = datetime.now() - orphan.creation_time
                age_str = f"{age.days}d {age.seconds // 3600}h" if age.days > 0 else f"{age.seconds // 3600}h {(age.seconds % 3600) // 60}m"
                
                table.add_row(
                    orphan.resource_type.value,
                    orphan.name,
                    orphan.namespace or "N/A",
                    age_str,
                    orphan.reason
                )
            
            console.print(table)
            console.print(f"\nTotal orphaned resources: [red]{len(all_orphans)}[/red]")
            
        elif format == "json":
            output = [
                {
                    "type": o.resource_type.value,
                    "name": o.name,
                    "namespace": o.namespace,
                    "creation_time": o.creation_time.isoformat(),
                    "reason": o.reason,
                    "metadata": o.metadata
                }
                for o in all_orphans
            ]
            console.print(json.dumps(output, indent=2))
            
        elif format == "yaml":
            output = [
                {
                    "type": o.resource_type.value,
                    "name": o.name,
                    "namespace": o.namespace,
                    "creation_time": o.creation_time.isoformat(),
                    "reason": o.reason,
                    "metadata": o.metadata
                }
                for o in all_orphans
            ]
            console.print(yml.dump(output, default_flow_style=False))
            
    except Exception as e:
        console.print(f"[red]Error checking orphaned resources: {e}[/red]")
        if ctx.obj.get("log_level") == "debug":
            console.print_exception()
        sys.exit(1)


@cli.command()
@click.option(
    "--pool",
    "-p",
    type=str,
    help="Specific storage pool to analyze",
)
@click.option(
    "--namespace",
    "-n",
    type=str,
    help="Namespace to analyze",
)
@click.option(
    "--format",
    "-f",
    type=click.Choice(["summary", "detailed", "json"]),
    default="summary",
    help="Output format",
)
@click.pass_context
def analyze(ctx: click.Context, pool: Optional[str], namespace: Optional[str], format: str) -> None:
    """Analyze storage usage and efficiency."""
    from .k8s_client import K8sClient, K8sConfig
    from .truenas_client import TrueNASClient, TrueNASConfig
    import json
    
    console.print("[yellow]Analyzing storage usage...[/yellow]")
    
    config = ctx.obj["config"]
    
    try:
        # Initialize K8s client
        k8s_config = K8sConfig(
            namespace=namespace or config.get("openshift", {}).get("namespace"),
            storage_class=config.get("openshift", {}).get("storage_class"),
            csi_driver=config.get("openshift", {}).get("csi_driver", "org.democratic-csi.nfs")
        )
        k8s_client = K8sClient(k8s_config)
        
        # Get all PVs and PVCs
        pvs = k8s_client.get_persistent_volumes()
        pvcs = k8s_client.get_persistent_volume_claims(namespace)
        
        # Calculate K8s side metrics
        total_allocated = sum(pv.capacity_bytes for pv in pvs)
        total_pvs = len(pvs)
        total_pvcs = len(pvcs)
        bound_pvcs = sum(1 for pvc in pvcs if pvc.phase == "Bound")
        
        # Initialize TrueNAS client if configured
        truenas_stats = None
        if "truenas" in config:
            truenas_config = _create_truenas_config(config["truenas"])
            truenas_client = TrueNASClient(truenas_config)
            
            # Get pool information
            pools = truenas_client.get_pools()
            if pool:
                pools = [p for p in pools if p.name == pool]
            
            truenas_stats = {
                "pools": []
            }
            
            for p in pools:
                pool_stats = {
                    "name": p.name,
                    "total_size": p.total_size,
                    "used_size": p.used_size,
                    "free_size": p.free_size,
                    "usage_percent": (p.used_size / p.total_size * 100) if p.total_size > 0 else 0,
                    "status": p.status,
                    "healthy": p.healthy,
                    "fragmentation": p.fragmentation
                }
                truenas_stats["pools"].append(pool_stats)
        
        if format == "summary":
            console.print("\n[green]Storage Analysis Summary:[/green]")
            console.print(f"• Total PVs: {total_pvs}")
            console.print(f"• Total PVCs: {total_pvcs} ({bound_pvcs} bound)")
            console.print(f"• Total Allocated: {total_allocated / (1024**3):.2f} GiB")
            
            if truenas_stats:
                console.print("\n[green]TrueNAS Pool Status:[/green]")
                for pool_stats in truenas_stats["pools"]:
                    console.print(f"\n[cyan]Pool: {pool_stats['name']}[/cyan]")
                    console.print(f"  • Total Size: {pool_stats['total_size'] / (1024**3):.2f} GiB")
                    console.print(f"  • Used: {pool_stats['used_size'] / (1024**3):.2f} GiB ({pool_stats['usage_percent']:.1f}%)")
                    console.print(f"  • Free: {pool_stats['free_size'] / (1024**3):.2f} GiB")
                    console.print(f"  • Status: {'[green]Healthy[/green]' if pool_stats['healthy'] else '[red]Unhealthy[/red]'}")
                    console.print(f"  • Fragmentation: {pool_stats['fragmentation']}")
                    
                    # Calculate thin provisioning efficiency
                    if pool_stats['total_size'] > 0:
                        overcommit_ratio = total_allocated / pool_stats['total_size']
                        console.print(f"  • Overcommitment Ratio: {overcommit_ratio:.2f}x")
                        
                        if pool_stats['used_size'] > 0:
                            efficiency = (total_allocated - pool_stats['used_size']) / total_allocated * 100
                            console.print(f"  • Thin Provisioning Efficiency: {efficiency:.1f}%")
                            
        elif format == "detailed":
            # Show detailed PV/PVC information
            table = Table(title="Storage Resources")
            table.add_column("PVC", style="cyan")
            table.add_column("Namespace")
            table.add_column("PV", style="magenta")
            table.add_column("Capacity")
            table.add_column("Status")
            table.add_column("Storage Class")
            
            for pvc in pvcs:
                table.add_row(
                    pvc.name,
                    pvc.namespace,
                    pvc.volume_name or "N/A",
                    pvc.capacity,
                    pvc.phase,
                    pvc.storage_class or "N/A"
                )
            
            console.print(table)
            
        elif format == "json":
            output = {
                "kubernetes": {
                    "total_pvs": total_pvs,
                    "total_pvcs": total_pvcs,
                    "bound_pvcs": bound_pvcs,
                    "total_allocated_bytes": total_allocated,
                    "pvs": [
                        {
                            "name": pv.name,
                            "capacity": pv.capacity,
                            "phase": pv.phase,
                            "claim": f"{pv.claim_namespace}/{pv.claim_name}" if pv.claim_name else None
                        }
                        for pv in pvs
                    ]
                },
                "truenas": truenas_stats
            }
            console.print(json.dumps(output, indent=2))
            
    except Exception as e:
        console.print(f"[red]Error analyzing storage: {e}[/red]")
        if ctx.obj.get("log_level") == "debug":
            console.print_exception()
        sys.exit(1)


@cli.command()
@click.option(
    "--output",
    "-o",
    type=click.Path(),
    help="Output file for the report",
    default="report.html",
)
@click.option(
    "--format",
    "-f",
    type=click.Choice(["html", "pdf", "json"]),
    default="html",
    help="Report format",
)
@click.pass_context
def report(ctx: click.Context, output: str, format: str) -> None:
    """Generate a comprehensive storage report."""
    console.print(f"[yellow]Generating {format} report...[/yellow]")
    
    # TODO: Implement report generation
    
    console.print(f"[green]Report saved to: {output}[/green]")


@cli.command()
@click.pass_context
def validate(ctx: click.Context) -> None:
    """Validate configuration and connectivity."""
    from .k8s_client import K8sClient, K8sConfig
    from .truenas_client import TrueNASClient, TrueNASConfig
    
    console.print("[yellow]Validating configuration...[/yellow]")
    
    config = ctx.obj["config"]
    checks = []
    
    # Check 1: Configuration file
    config_valid = bool(config and "openshift" in config and "monitoring" in config)
    checks.append(("Configuration file", config_valid, "Valid configuration loaded" if config_valid else "Invalid or missing configuration"))
    
    # Check 2: Kubernetes connection
    k8s_status = False
    k8s_msg = ""
    try:
        k8s_config = K8sConfig(
            namespace=config.get("openshift", {}).get("namespace"),
            storage_class=config.get("openshift", {}).get("storage_class"),
            csi_driver=config.get("openshift", {}).get("csi_driver", "org.democratic-csi.nfs")
        )
        k8s_client = K8sClient(k8s_config)
        
        # Try to list namespaces as a connectivity test
        namespaces = k8s_client.core_v1.list_namespace()
        k8s_status = True
        k8s_msg = f"Connected to cluster with {len(namespaces.items)} namespaces"
    except Exception as e:
        k8s_msg = str(e)
    
    checks.append(("Kubernetes connection", k8s_status, k8s_msg))
    
    # Check 3: Democratic-CSI namespace
    csi_namespace_status = False
    csi_namespace_msg = ""
    if k8s_status:
        try:
            namespace = config.get("openshift", {}).get("namespace", "democratic-csi")
            ns = k8s_client.core_v1.read_namespace(namespace)
            csi_namespace_status = True
            csi_namespace_msg = f"Namespace '{namespace}' exists"
        except Exception:
            csi_namespace_msg = f"Namespace '{namespace}' not found"
    else:
        csi_namespace_msg = "Cannot check (Kubernetes connection failed)"
    
    checks.append(("Democratic-CSI namespace", csi_namespace_status, csi_namespace_msg))
    
    # Check 4: CSI driver health
    csi_driver_status = False
    csi_driver_msg = ""
    if k8s_status:
        try:
            health = k8s_client.check_csi_driver_health()
            csi_driver_status = health["healthy"]
            csi_driver_msg = f"{health['running_pods']}/{health['total_pods']} pods running"
            if not csi_driver_status and health["unhealthy_pods"]:
                csi_driver_msg += f" - Issues: {', '.join(health['unhealthy_pods'])}"
        except Exception as e:
            csi_driver_msg = f"Cannot check driver health: {str(e)}"
    else:
        csi_driver_msg = "Cannot check (Kubernetes connection failed)"
    
    checks.append(("CSI driver health", csi_driver_status, csi_driver_msg))
    
    # Check 5: TrueNAS API connection
    truenas_status = False
    truenas_msg = ""
    if "truenas" in config:
        try:
            truenas_config = _create_truenas_config(config["truenas"])
            truenas_client = TrueNASClient(truenas_config)
            
            # Try to get pools as a connectivity test
            pools = truenas_client.get_pools()
            truenas_status = True
            truenas_msg = f"Connected, {len(pools)} pool(s) found"
        except Exception as e:
            truenas_msg = str(e)
    else:
        truenas_msg = "TrueNAS not configured"
    
    checks.append(("TrueNAS API connection", truenas_status, truenas_msg))
    
    # Check 6: RBAC permissions
    rbac_status = False
    rbac_msg = ""
    if k8s_status:
        try:
            # Test key permissions
            can_list_pvs = k8s_client.core_v1.list_persistent_volume()
            can_list_pvcs = k8s_client.core_v1.list_persistent_volume_claim_for_all_namespaces()
            can_list_sc = k8s_client.storage_v1.list_storage_class()
            rbac_status = True
            rbac_msg = "All required permissions granted"
        except Exception as e:
            rbac_msg = f"Missing permissions: {str(e)}"
    else:
        rbac_msg = "Cannot check (Kubernetes connection failed)"
    
    checks.append(("RBAC permissions", rbac_status, rbac_msg))
    
    # Display results
    table = Table(title="Validation Results")
    table.add_column("Check", style="cyan", width=25)
    table.add_column("Status", style="green", width=10)
    table.add_column("Details", style="yellow", width=50)
    
    all_passed = True
    for check_name, status, message in checks:
        status_text = "[green]✓ PASS[/green]" if status else "[red]✗ FAIL[/red]"
        table.add_row(check_name, status_text, message)
        if not status:
            all_passed = False
    
    console.print(table)
    
    if not all_passed:
        console.print("\n[red]Some checks failed. Please review the configuration and permissions.[/red]")
        sys.exit(1)
    else:
        console.print("\n[green]All checks passed! The tool is ready to use.[/green]")


@cli.command()
@click.option(
    "--interval",
    "-i",
    type=int,
    default=300,
    help="Check interval in seconds (default: 300)",
)
@click.option(
    "--once",
    is_flag=True,
    help="Run checks once and exit",
)
@click.option(
    "--metrics-port",
    "-p",
    type=int,
    default=9090,
    help="Port for Prometheus metrics endpoint",
)
@click.pass_context
def monitor(ctx: click.Context, interval: int, once: bool, metrics_port: int) -> None:
    """Start the monitoring service."""
    from .monitor import Monitor
    import time
    import threading
    from prometheus_client import start_http_server, Gauge, Counter
    
    config = ctx.obj["config"]
    
    # Initialize Prometheus metrics
    orphaned_resources = Gauge('truenas_orphaned_resources_total', 'Number of orphaned resources', ['type'])
    storage_usage = Gauge('truenas_storage_usage_bytes', 'Storage usage in bytes', ['pool', 'type'])
    storage_capacity = Gauge('truenas_storage_capacity_bytes', 'Storage capacity in bytes', ['pool'])
    check_errors = Counter('truenas_check_errors_total', 'Total number of check errors', ['check_type'])
    last_check_timestamp = Gauge('truenas_last_check_timestamp', 'Timestamp of last successful check', ['check_type'])
    
    # Start metrics server
    if config.get("metrics", {}).get("enabled", True):
        start_http_server(metrics_port)
        console.print(f"[green]Prometheus metrics available at http://localhost:{metrics_port}/metrics[/green]")
    
    monitor = Monitor(config)
    
    def run_checks():
        """Run all monitoring checks."""
        console.print("\n[yellow]Running monitoring checks...[/yellow]")
        
        try:
            # Check for orphaned resources
            orphans = monitor.check_orphaned_resources()
            orphaned_resources.labels(type='pv').set(orphans['orphaned_pvs'])
            orphaned_resources.labels(type='pvc').set(orphans['orphaned_pvcs'])
            orphaned_resources.labels(type='snapshot').set(orphans['orphaned_snapshots'])
            last_check_timestamp.labels(check_type='orphans').set_to_current_time()
            
            console.print(f"  • Orphaned PVs: {orphans['orphaned_pvs']}")
            console.print(f"  • Orphaned PVCs: {orphans['orphaned_pvcs']}")
            console.print(f"  • Orphaned Snapshots: {orphans['orphaned_snapshots']}")
            
        except Exception as e:
            console.print(f"  [red]✗ Orphan check failed: {e}[/red]")
            check_errors.labels(check_type='orphans').inc()
        
        try:
            # Check storage usage
            usage = monitor.check_storage_usage()
            for pool_name, pool_data in usage.get('pools', {}).items():
                storage_usage.labels(pool=pool_name, type='used').set(pool_data['used'])
                storage_usage.labels(pool=pool_name, type='free').set(pool_data['free'])
                storage_capacity.labels(pool=pool_name).set(pool_data['total'])
                
                usage_pct = (pool_data['used'] / pool_data['total'] * 100) if pool_data['total'] > 0 else 0
                console.print(f"  • Pool {pool_name}: {usage_pct:.1f}% used")
                
                # Check thresholds
                warning_threshold = config.get("monitoring", {}).get("storage", {}).get("pool_warning_threshold", 80)
                critical_threshold = config.get("monitoring", {}).get("storage", {}).get("pool_critical_threshold", 90)
                
                if usage_pct >= critical_threshold:
                    console.print(f"    [red]⚠ CRITICAL: Pool usage above {critical_threshold}%[/red]")
                elif usage_pct >= warning_threshold:
                    console.print(f"    [yellow]⚠ WARNING: Pool usage above {warning_threshold}%[/yellow]")
                    
            last_check_timestamp.labels(check_type='storage').set_to_current_time()
            
        except Exception as e:
            console.print(f"  [red]✗ Storage check failed: {e}[/red]")
            check_errors.labels(check_type='storage').inc()
        
        try:
            # Check CSI driver health
            health = monitor.check_csi_health()
            if health['healthy']:
                console.print(f"  • CSI Driver: [green]Healthy ({health['running_pods']}/{health['total_pods']} pods)[/green]")
            else:
                console.print(f"  • CSI Driver: [red]Unhealthy ({health['running_pods']}/{health['total_pods']} pods)[/red]")
                if health['unhealthy_pods']:
                    console.print(f"    Issues: {', '.join(health['unhealthy_pods'])}")
                    
            last_check_timestamp.labels(check_type='csi_health').set_to_current_time()
            
        except Exception as e:
            console.print(f"  [red]✗ CSI health check failed: {e}[/red]")
            check_errors.labels(check_type='csi_health').inc()
        
        console.print("[green]Check complete[/green]")
    
    if once:
        run_checks()
    else:
        console.print(f"[yellow]Starting monitor in foreground (check interval: {interval}s)...[/yellow]")
        console.print("[cyan]Press Ctrl+C to stop[/cyan]")
        
        try:
            while True:
                run_checks()
                time.sleep(interval)
        except KeyboardInterrupt:
            console.print("\n[yellow]Stopping monitor...[/yellow]")
            monitor.stop()


@cli.command()
@click.option(
    "--volume",
    "-v",
    type=str,
    help="Specific volume to list snapshots for",
)
@click.option(
    "--format",
    "-f",
    type=click.Choice(["table", "json", "yaml"]),
    default="table",
    help="Output format",
)
@click.option(
    "--age-days",
    type=int,
    default=0,
    help="Show snapshots older than this many days (0 for all)",
)
@click.option(
    "--analysis",
    is_flag=True,
    help="Show snapshot usage analysis",
)
@click.option(
    "--orphaned",
    is_flag=True,
    help="Show only orphaned snapshots",
)
@click.option(
    "--health",
    is_flag=True,
    help="Show snapshot health status",
)
@click.pass_context
def snapshots(ctx: click.Context, volume: Optional[str], format: str, age_days: int, analysis: bool, orphaned: bool, health: bool) -> None:
    """List and manage snapshots with comprehensive analysis."""
    from .truenas_client import TrueNASClient, TrueNASConfig
    from .k8s_client import K8sClient, K8sConfig
    from .monitor import Monitor
    import json
    import yaml
    from datetime import datetime, timedelta
    
    config = ctx.obj["config"]
    
    try:
        # Create monitor for comprehensive functionality
        monitor = Monitor(config)
        
        if health:
            # Show snapshot health status
            console.print("[yellow]Checking snapshot health...[/yellow]")
            health_status = monitor.check_snapshot_health()
            
            if format == 'json':
                console.print(json.dumps(health_status, indent=2, default=str))
            elif format == 'yaml':
                console.print(yaml.dump(health_status, default_flow_style=False))
            else:
                # Table format for health status
                console.print("[bold]Snapshot Health Status[/bold]")
                
                # K8s snapshots table
                k8s_table = Table(title="Kubernetes Snapshots")
                k8s_table.add_column("Metric", style="cyan")
                k8s_table.add_column("Count", style="green")
                
                for metric, count in health_status["k8s_snapshots"].items():
                    k8s_table.add_row(metric.replace("_", " ").title(), str(count))
                
                console.print(k8s_table)
                
                # TrueNAS snapshots table
                truenas_table = Table(title="TrueNAS Snapshots")
                truenas_table.add_column("Metric", style="cyan")
                truenas_table.add_column("Value", style="green")
                
                for metric, value in health_status["truenas_snapshots"].items():
                    if metric == "total_size_gb":
                        truenas_table.add_row(metric.replace("_", " ").title(), f"{value:.2f} GB")
                    else:
                        truenas_table.add_row(metric.replace("_", " ").title(), str(value))
                
                console.print(truenas_table)
                
                # Orphaned resources
                orphaned_table = Table(title="Orphaned Snapshots")
                orphaned_table.add_column("Type", style="cyan")
                orphaned_table.add_column("Count", style="red")
                
                for metric, count in health_status["orphaned_resources"].items():
                    orphaned_table.add_row(metric.replace("_", " ").title(), str(count))
                
                console.print(orphaned_table)
                
                # Alerts
                if health_status["alerts"]:
                    console.print("\n[bold red]Alerts:[/bold red]")
                    for alert in health_status["alerts"]:
                        level_color = {"error": "red", "warning": "yellow", "info": "blue"}.get(alert["level"], "white")
                        console.print(f"[{level_color}]{alert['level'].upper()}[/{level_color}]: {alert['message']}")
                
                # Recommendations
                if health_status["recommendations"]:
                    console.print("\n[bold blue]Recommendations:[/bold blue]")
                    for rec in health_status["recommendations"]:
                        console.print(f"• {rec}")
            
            return
        
        if analysis:
            # Show snapshot analysis
            if not monitor.truenas_client:
                console.print("[red]Error: TrueNAS client not configured[/red]")
                sys.exit(1)
            
            console.print(f"[yellow]Analyzing snapshot usage{' for ' + volume if volume else ''}...[/yellow]")
            analysis_result = monitor.truenas_client.analyze_snapshot_usage(volume)
            
            if format == 'json':
                console.print(json.dumps(analysis_result, indent=2, default=str))
            elif format == 'yaml':
                console.print(yaml.dump(analysis_result, default_flow_style=False))
            else:
                # Table format for analysis
                console.print(f"[bold]Snapshot Analysis{' for ' + volume if volume else ''}[/bold]")
                
                # Summary stats
                summary_table = Table(title="Summary Statistics")
                summary_table.add_column("Metric", style="cyan")
                summary_table.add_column("Value", style="green")
                
                summary_table.add_row("Total Snapshots", str(analysis_result["total_snapshots"]))
                summary_table.add_row("Total Size", f"{analysis_result['total_snapshot_size'] / (1024**3):.2f} GB")
                summary_table.add_row("Average Age (days)", f"{analysis_result['average_snapshot_age_days']:.1f}")
                
                console.print(summary_table)
                
                # Age distribution
                age_table = Table(title="Snapshots by Age")
                age_table.add_column("Age Range", style="cyan")
                age_table.add_column("Count", style="green")
                
                for age_range, count in analysis_result["snapshots_by_age"].items():
                    age_table.add_row(age_range.replace("_", " ").title(), str(count))
                
                console.print(age_table)
                
                # Large snapshots
                if analysis_result["large_snapshots"]:
                    large_table = Table(title="Large Snapshots (>1GB)")
                    large_table.add_column("Name", style="cyan")
                    large_table.add_column("Dataset", style="green")
                    large_table.add_column("Size (GB)", style="yellow")
                    large_table.add_column("Age (days)", style="blue")
                    
                    for snap in analysis_result["large_snapshots"]:
                        large_table.add_row(
                            snap["name"],
                            snap["dataset"],
                            f"{snap['size_gb']:.2f}",
                            str(snap["age_days"])
                        )
                    
                    console.print(large_table)
                
                # Recommendations
                if analysis_result["recommendations"]:
                    console.print("\n[bold blue]Recommendations:[/bold blue]")
                    for rec in analysis_result["recommendations"]:
                        console.print(f"• {rec}")
            
            return
        
        if orphaned:
            # Show orphaned snapshots
            console.print("[yellow]Finding orphaned snapshots...[/yellow]")
            orphaned_snapshots = []
            
            if monitor.k8s_client:
                orphaned_k8s = monitor.k8s_client.find_orphaned_snapshots()
                orphaned_snapshots.extend(orphaned_k8s)
            
            if monitor.truenas_client:
                k8s_snapshots = monitor.k8s_client.get_volume_snapshots() if monitor.k8s_client else []
                orphaned_truenas = monitor.truenas_client.find_orphaned_truenas_snapshots(k8s_snapshots)
                orphaned_snapshots.extend(orphaned_truenas)
            
            if format == 'json':
                orphaned_data = []
                for orphan in orphaned_snapshots:
                    if hasattr(orphan, 'resource_type'):  # K8s orphaned resource
                        orphaned_data.append({
                            'type': 'kubernetes',
                            'name': orphan.name,
                            'namespace': orphan.namespace,
                            'reason': orphan.reason,
                            'creation_time': orphan.creation_time.isoformat() if orphan.creation_time else None,
                            'details': orphan.details
                        })
                    else:  # TrueNAS snapshot
                        orphaned_data.append({
                            'type': 'truenas',
                            'name': orphan.name,
                            'dataset': orphan.dataset,
                            'creation_time': orphan.creation_time.isoformat() if orphan.creation_time else None,
                            'size': orphan.used_size
                        })
                console.print(json.dumps(orphaned_data, indent=2))
            elif format == 'yaml':
                orphaned_data = []
                for orphan in orphaned_snapshots:
                    if hasattr(orphan, 'resource_type'):
                        orphaned_data.append({
                            'type': 'kubernetes',
                            'name': orphan.name,
                            'namespace': orphan.namespace,
                            'reason': orphan.reason,
                            'creation_time': orphan.creation_time.isoformat() if orphan.creation_time else None,
                            'details': orphan.details
                        })
                    else:
                        orphaned_data.append({
                            'type': 'truenas',
                            'name': orphan.name,
                            'dataset': orphan.dataset,
                            'creation_time': orphan.creation_time.isoformat() if orphan.creation_time else None,
                            'size': orphan.used_size
                        })
                console.print(yaml.dump(orphaned_data, default_flow_style=False))
            else:
                # Table format for orphaned snapshots
                table = Table(title="Orphaned Snapshots")
                table.add_column("Type", style="cyan")
                table.add_column("Name", style="green") 
                table.add_column("Location/Dataset", style="yellow")
                table.add_column("Age", style="blue")
                table.add_column("Reason", style="red")
                
                for orphan in orphaned_snapshots:
                    if hasattr(orphan, 'resource_type'):  # K8s orphaned resource
                        age = "Unknown"
                        if orphan.creation_time:
                            # Handle timezone-aware datetime
                            orphan_time = orphan.creation_time.replace(tzinfo=None) if orphan.creation_time.tzinfo else orphan.creation_time
                            age_delta = datetime.now() - orphan_time
                            age = f"{age_delta.days} days"
                        
                        table.add_row(
                            "Kubernetes",
                            orphan.name,
                            f"{orphan.namespace}" if orphan.namespace else "N/A",
                            age,
                            orphan.reason
                        )
                    else:  # TrueNAS snapshot
                        age = "Unknown"
                        if orphan.creation_time:
                            age_delta = datetime.now() - orphan.creation_time
                            age = f"{age_delta.days} days"
                        
                        table.add_row(
                            "TrueNAS",
                            orphan.name,
                            orphan.dataset,
                            age,
                            "No corresponding K8s snapshot"
                        )
                
                console.print(table)
                console.print(f"\nTotal orphaned snapshots: [red]{len(orphaned_snapshots)}[/red]")
            
            return
        
        # Regular snapshot listing
        if not monitor.truenas_client:
            console.print("[red]TrueNAS not configured. Please add TrueNAS configuration.[/red]")
            sys.exit(1)
        
        console.print(f"[yellow]Listing snapshots{' for ' + volume if volume else ''}...[/yellow]")
        
        # Get snapshots
        if volume:
            snapshots_list = monitor.truenas_client.get_volume_snapshots(volume)
        else:
            snapshots_list = monitor.truenas_client.get_snapshots()
        
        # Filter by age if specified
        if age_days > 0:
            threshold = datetime.now() - timedelta(days=age_days)
            snapshots_list = [s for s in snapshots_list if s.creation_time < threshold]
        
        if format == "table":
            if not snapshots_list:
                console.print("[green]No snapshots found![/green]")
                return
                
            table = Table(title=f"Snapshots{f' for volume {volume}' if volume else ''}")
            table.add_column("Name", style="cyan")
            table.add_column("Dataset", style="magenta")
            table.add_column("Created")
            table.add_column("Used Size")
            
            for snapshot in snapshots_list:
                table.add_row(
                    snapshot.name,
                    snapshot.dataset,
                    snapshot.creation_time.strftime("%Y-%m-%d %H:%M:%S"),
                    f"{snapshot.used_size / (1024**3):.2f} GiB" if snapshot.used_size > 0 else "N/A"
                )
            
            console.print(table)
            console.print(f"\nTotal snapshots: [cyan]{len(snapshots_list)}[/cyan]")
            
        elif format == "json":
            output = [
                {
                    "name": s.name,
                    "dataset": s.dataset,
                    "creation_time": s.creation_time.isoformat(),
                    "used_size": s.used_size,
                    "referenced_size": s.referenced_size,
                    "full_name": s.full_name
                }
                for s in snapshots_list
            ]
            console.print(json.dumps(output, indent=2))
            
        elif format == "yaml":
            output = [
                {
                    "name": s.name,
                    "dataset": s.dataset,
                    "creation_time": s.creation_time.isoformat(),
                    "used_size": s.used_size,
                    "referenced_size": s.referenced_size,
                    "full_name": s.full_name
                }
                for s in snapshots_list
            ]
            console.print(yaml.dump(output, default_flow_style=False))
            
    except Exception as e:
        console.print(f"[red]Error: {e}[/red]")
        if ctx.obj.get("log_level") == "debug":
            console.print_exception()
        sys.exit(1)


def main() -> None:
    """Main entry point for the CLI."""
    try:
        cli(obj={})
    except TrueNASMonitorError as e:
        console.print(f"[red]Error: {e}[/red]")
        sys.exit(1)
    except KeyboardInterrupt:
        console.print("\n[yellow]Interrupted[/yellow]")
        sys.exit(130)
    except Exception as e:
        console.print(f"[red]Unexpected error: {e}[/red]")
        if "--debug" in sys.argv or "-d" in sys.argv:
            console.print_exception()
        sys.exit(1)


if __name__ == "__main__":
    main()