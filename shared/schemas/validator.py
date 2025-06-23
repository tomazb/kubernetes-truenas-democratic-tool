"""JSON schema validator for shared data models."""

import json
import os
from pathlib import Path
from typing import Dict, Any, List, Optional

import jsonschema
from jsonschema import Draft7Validator, ValidationError


class SchemaValidator:
    """Validator for JSON schemas used across the project."""
    
    def __init__(self, schema_dir: Optional[Path] = None):
        """Initialize the schema validator.
        
        Args:
            schema_dir: Directory containing JSON schema files
        """
        if schema_dir is None:
            schema_dir = Path(__file__).parent
        
        self.schema_dir = schema_dir
        self.schemas: Dict[str, Dict[str, Any]] = {}
        self._load_schemas()
    
    def _load_schemas(self):
        """Load all JSON schemas from the schema directory."""
        schema_files = {
            "orphaned-resources": "orphaned-resources.json",
            "storage-analysis": "storage-analysis.json",
            "config-validation": "config-validation.json",
        }
        
        for name, filename in schema_files.items():
            schema_path = self.schema_dir / filename
            if schema_path.exists():
                with open(schema_path, 'r') as f:
                    self.schemas[name] = json.load(f)
    
    def validate(self, data: Dict[str, Any], schema_name: str) -> List[str]:
        """Validate data against a named schema.
        
        Args:
            data: Data to validate
            schema_name: Name of the schema to use
            
        Returns:
            List of validation error messages (empty if valid)
        """
        if schema_name not in self.schemas:
            return [f"Unknown schema: {schema_name}"]
        
        schema = self.schemas[schema_name]
        validator = Draft7Validator(schema)
        
        errors = []
        for error in validator.iter_errors(data):
            # Build error path
            path = ".".join(str(p) for p in error.path) if error.path else "root"
            errors.append(f"{path}: {error.message}")
        
        return errors
    
    def is_valid(self, data: Dict[str, Any], schema_name: str) -> bool:
        """Check if data is valid against a schema.
        
        Args:
            data: Data to validate
            schema_name: Name of the schema to use
            
        Returns:
            True if valid, False otherwise
        """
        return len(self.validate(data, schema_name)) == 0
    
    def validate_orphaned_resources(self, data: Dict[str, Any]) -> List[str]:
        """Validate orphaned resources report data."""
        return self.validate(data, "orphaned-resources")
    
    def validate_storage_analysis(self, data: Dict[str, Any]) -> List[str]:
        """Validate storage analysis report data."""
        return self.validate(data, "storage-analysis")
    
    def validate_config_validation(self, data: Dict[str, Any]) -> List[str]:
        """Validate configuration validation report data."""
        return self.validate(data, "config-validation")


def create_orphaned_resource(
    resource_type: str,
    name: str,
    location: str,
    reason: str,
    created_at: str,
    namespace: Optional[str] = None,
    volume_handle: Optional[str] = None,
    size_bytes: Optional[int] = None,
    remediation_action: str = "manual_review",
    safe: bool = False,
    **details
) -> Dict[str, Any]:
    """Create a properly formatted orphaned resource entry.
    
    Args:
        resource_type: Type of resource (PersistentVolume, etc.)
        name: Resource name
        location: Where the resource exists (Kubernetes or TrueNAS)
        reason: Why it's considered orphaned
        created_at: ISO format timestamp
        namespace: Kubernetes namespace (if applicable)
        volume_handle: Volume handle/ID
        size_bytes: Size in bytes
        remediation_action: Suggested remediation
        safe: Whether remediation is safe to automate
        **details: Additional details
        
    Returns:
        Orphaned resource dictionary
    """
    resource = {
        "type": resource_type,
        "name": name,
        "namespace": namespace,
        "volume_handle": volume_handle,
        "created_at": created_at,
        "size_bytes": size_bytes,
        "location": location,
        "reason": reason,
        "remediation": {
            "action": remediation_action,
            "safe": safe,
        },
        "details": details,
    }
    
    # Remove None values
    resource = {k: v for k, v in resource.items() if v is not None}
    if "remediation" in resource:
        resource["remediation"] = {
            k: v for k, v in resource["remediation"].items() if v is not None
        }
    
    return resource


def create_storage_alert(
    level: str,
    category: str,
    message: str,
    resource: Optional[str] = None,
    threshold: Optional[float] = None,
    current_value: Optional[float] = None,
    **details
) -> Dict[str, Any]:
    """Create a properly formatted storage alert.
    
    Args:
        level: Alert level (info, warning, error, critical)
        category: Alert category (capacity, performance, etc.)
        message: Alert message
        resource: Related resource name
        threshold: Threshold value that triggered alert
        current_value: Current value
        **details: Additional details
        
    Returns:
        Alert dictionary
    """
    alert = {
        "level": level,
        "category": category,
        "message": message,
        "resource": resource,
        "threshold": threshold,
        "current_value": current_value,
        "details": details if details else {},
    }
    
    # Remove None values
    return {k: v for k, v in alert.items() if v is not None}