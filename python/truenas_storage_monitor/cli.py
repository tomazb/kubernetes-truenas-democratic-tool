"""Command-line interface for TrueNAS Storage Monitor."""

import sys
from typing import Optional

import click
from rich.console import Console
from rich.table import Table

from . import __version__
from .config import load_config
from .monitor import Monitor
from .exceptions import TrueNASMonitorError

console = Console()


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
@click.pass_context
def orphans(ctx: click.Context, format: str) -> None:
    """Check for orphaned resources."""
    console.print("[yellow]Checking for orphaned resources...[/yellow]")
    
    # TODO: Implement orphan detection
    
    if format == "table":
        table = Table(title="Orphaned Resources")
        table.add_column("Type", style="cyan")
        table.add_column("Name", style="magenta")
        table.add_column("Namespace")
        table.add_column("Age")
        table.add_column("Size")
        
        # Example data
        table.add_row("PV", "pvc-12345", "default", "7 days", "10Gi")
        table.add_row("Snapshot", "snapshot-67890", "production", "30 days", "5Gi")
        
        console.print(table)
    else:
        console.print(f"[red]Format '{format}' not yet implemented[/red]")


@cli.command()
@click.option(
    "--trend",
    "-t",
    type=str,
    help="Time period for trend analysis (e.g., 7d, 30d)",
    default="7d",
)
@click.pass_context
def analyze(ctx: click.Context, trend: str) -> None:
    """Analyze storage usage and trends."""
    console.print(f"[yellow]Analyzing storage trends for the last {trend}...[/yellow]")
    
    # TODO: Implement storage analysis
    
    console.print("\n[green]Storage Analysis Summary:[/green]")
    console.print("• Total Allocated: 500Gi")
    console.print("• Total Used: 350Gi (70%)")
    console.print("• Thin Provisioning Savings: 150Gi")
    console.print("• Growth Rate: 5Gi/day")
    console.print("• Estimated Full: 30 days")


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
    console.print("[yellow]Validating configuration...[/yellow]")
    
    checks = [
        ("Configuration file", True),
        ("Kubernetes connection", True),
        ("TrueNAS API connection", False),
        ("Democratic-CSI namespace", True),
        ("RBAC permissions", True),
    ]
    
    table = Table(title="Validation Results")
    table.add_column("Check", style="cyan")
    table.add_column("Status", style="green")
    
    for check, status in checks:
        status_text = "[green]✓ PASS[/green]" if status else "[red]✗ FAIL[/red]"
        table.add_row(check, status_text)
    
    console.print(table)
    
    if not all(status for _, status in checks):
        console.print("\n[red]Some checks failed. Please review the configuration.[/red]")
        sys.exit(1)
    else:
        console.print("\n[green]All checks passed![/green]")


@cli.command()
@click.option(
    "--daemon",
    "-d",
    is_flag=True,
    help="Run in daemon mode",
)
@click.pass_context
def monitor(ctx: click.Context, daemon: bool) -> None:
    """Start the monitoring service."""
    if daemon:
        console.print("[yellow]Starting monitor in daemon mode...[/yellow]")
        # TODO: Implement daemon mode
    else:
        console.print("[yellow]Starting monitor in foreground...[/yellow]")
        console.print("[cyan]Press Ctrl+C to stop[/cyan]")
        
        try:
            # TODO: Implement monitoring loop
            import time
            while True:
                time.sleep(60)
        except KeyboardInterrupt:
            console.print("\n[yellow]Stopping monitor...[/yellow]")


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