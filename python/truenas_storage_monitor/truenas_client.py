"""TrueNAS REST API client for storage monitoring."""

import logging
from dataclasses import dataclass, field
from datetime import datetime
from typing import List, Dict, Optional, Any, Union
from urllib.parse import quote

import requests
from requests.adapters import HTTPAdapter
from requests.packages.urllib3.util.retry import Retry

from .exceptions import TrueNASMonitorError, ConnectionError as MonitorConnectionError


logger = logging.getLogger(__name__)


class TrueNASError(TrueNASMonitorError):
    """Base exception for TrueNAS client errors."""
    pass


class AuthenticationError(TrueNASError):
    """Authentication failed with TrueNAS."""
    pass


@dataclass
class TrueNASConfig:
    """Configuration for TrueNAS client."""
    host: str
    port: int = 443
    api_key: Optional[str] = None
    username: Optional[str] = None
    password: Optional[str] = None
    verify_ssl: bool = True
    timeout: int = 30
    max_retries: int = 3
    
    def __post_init__(self):
        """Validate configuration."""
        if not self.api_key and not (self.username and self.password):
            raise ValueError("Either api_key or username/password must be provided")
    
    @property
    def base_url(self) -> str:
        """Get the base URL for TrueNAS API."""
        protocol = "https" if self.port == 443 else "http"
        return f"{protocol}://{self.host}:{self.port}/api/v2.0"


@dataclass
class PoolInfo:
    """Information about a TrueNAS storage pool."""
    name: str
    status: str
    total_size: int
    used_size: int
    free_size: int
    fragmentation: str
    healthy: bool
    scan_state: Optional[str] = None
    datasets: List[str] = field(default_factory=list)


@dataclass
class DatasetInfo:
    """Information about a ZFS dataset."""
    name: str
    type: str
    used_size: int
    available_size: int
    referenced_size: int
    quota: Optional[int] = None
    compression_ratio: Optional[str] = None
    children: List[str] = field(default_factory=list)


@dataclass
class VolumeInfo:
    """Information about an iSCSI volume/extent."""
    name: str
    path: str
    size: int
    type: str
    enabled: bool
    naa: Optional[str] = None
    serial: Optional[str] = None


@dataclass
class SnapshotInfo:
    """Information about a ZFS snapshot."""
    name: str
    dataset: str
    creation_time: datetime
    used_size: int
    referenced_size: int
    full_name: str


@dataclass
class OrphanedVolume:
    """Information about an orphaned TrueNAS volume."""
    name: str
    path: str
    type: str  # 'iscsi' or 'nfs'
    size: Optional[int] = None
    creation_time: Optional[datetime] = None


class TrueNASClient:
    """Client for TrueNAS REST API."""

    def __init__(self, config: TrueNASConfig):
        """Initialize TrueNAS client.
        
        Args:
            config: TrueNAS client configuration
        """
        self.config = config
        self.base_url = config.base_url
        
        # Setup session with retry logic
        self.session = requests.Session()
        
        # Configure retries
        retry = Retry(
            total=config.max_retries,
            backoff_factor=0.3,
            status_forcelist=[500, 502, 503, 504],
        )
        adapter = HTTPAdapter(max_retries=retry)
        self.session.mount("http://", adapter)
        self.session.mount("https://", adapter)
        
        # Set authentication
        if config.api_key:
            self.session.headers["Authorization"] = f"Bearer {config.api_key}"
        else:
            self.session.auth = (config.username, config.password)
        
        # SSL verification
        self.session.verify = config.verify_ssl
        
        # Set default headers
        self.session.headers.update({
            "Accept": "application/json",
            "Content-Type": "application/json",
        })

    def test_connection(self) -> bool:
        """Test connection to TrueNAS API.
        
        Returns:
            True if connection successful
            
        Raises:
            AuthenticationError: If authentication fails
            TrueNASError: For other connection errors
        """
        try:
            response = self.session.get(
                f"{self.base_url}/auth/me",
                timeout=self.config.timeout
            )
            
            if response.status_code == 401:
                raise AuthenticationError("Authentication failed with TrueNAS")
            
            response.raise_for_status()
            logger.info(f"Successfully connected to TrueNAS at {self.config.host}")
            return True
            
        except requests.exceptions.RequestException as e:
            raise TrueNASError(f"Failed to connect to TrueNAS: {str(e)}")

    def get_pools(self) -> List[PoolInfo]:
        """Get all storage pools.
        
        Returns:
            List of PoolInfo objects
        """
        try:
            response = self.session.get(
                f"{self.base_url}/pool",
                timeout=self.config.timeout
            )
            response.raise_for_status()
            
            pools = []
            for pool_data in response.json():
                pool = PoolInfo(
                    name=pool_data["name"],
                    status=pool_data.get("status", "UNKNOWN"),
                    total_size=pool_data.get("size", 0),
                    used_size=pool_data.get("allocated", 0),
                    free_size=pool_data.get("free", 0),
                    fragmentation=pool_data.get("fragmentation", "0%"),
                    healthy=pool_data.get("healthy", False),
                    scan_state=pool_data.get("scan", {}).get("state"),
                    datasets=[],  # Will be populated separately if needed
                )
                pools.append(pool)
            
            logger.info(f"Found {len(pools)} storage pools")
            return pools
            
        except requests.exceptions.RequestException as e:
            raise TrueNASError(f"Failed to get pools: {str(e)}")

    def get_datasets(self, pool: Optional[str] = None) -> List[DatasetInfo]:
        """Get all datasets, optionally filtered by pool.
        
        Args:
            pool: Pool name to filter datasets
            
        Returns:
            List of DatasetInfo objects
        """
        try:
            params = {}
            if pool:
                params["pool"] = pool
            
            response = self.session.get(
                f"{self.base_url}/pool/dataset",
                params=params,
                timeout=self.config.timeout
            )
            response.raise_for_status()
            
            datasets = []
            for ds_data in response.json():
                dataset = DatasetInfo(
                    name=ds_data["id"],
                    type=ds_data.get("type", "FILESYSTEM"),
                    used_size=ds_data.get("used", {}).get("value", 0),
                    available_size=ds_data.get("available", {}).get("value", 0),
                    referenced_size=ds_data.get("referenced", {}).get("value", 0),
                    quota=ds_data.get("quota", {}).get("value"),
                    compression_ratio=ds_data.get("compressratio"),
                    children=[],
                )
                datasets.append(dataset)
            
            logger.info(f"Found {len(datasets)} datasets")
            return datasets
            
        except requests.exceptions.RequestException as e:
            raise TrueNASError(f"Failed to get datasets: {str(e)}")

    def get_volumes(self) -> List[VolumeInfo]:
        """Get all iSCSI volumes (extents).
        
        Returns:
            List of VolumeInfo objects
        """
        try:
            response = self.session.get(
                f"{self.base_url}/iscsi/extent",
                timeout=self.config.timeout
            )
            response.raise_for_status()
            
            volumes = []
            for extent_data in response.json():
                volume = VolumeInfo(
                    name=extent_data["name"],
                    path=extent_data.get("path", ""),
                    size=extent_data.get("filesize", 0),
                    type=extent_data.get("type", "FILE"),
                    enabled=extent_data.get("enabled", False),
                    naa=extent_data.get("naa"),
                    serial=extent_data.get("serial"),
                )
                volumes.append(volume)
            
            logger.info(f"Found {len(volumes)} iSCSI volumes")
            return volumes
            
        except requests.exceptions.RequestException as e:
            raise TrueNASError(f"Failed to get volumes: {str(e)}")

    def get_nfs_shares(self) -> List[Dict[str, Any]]:
        """Get all NFS shares.
        
        Returns:
            List of NFS share information
        """
        try:
            response = self.session.get(
                f"{self.base_url}/sharing/nfs",
                timeout=self.config.timeout
            )
            response.raise_for_status()
            
            shares = response.json()
            logger.info(f"Found {len(shares)} NFS shares")
            return shares
            
        except requests.exceptions.RequestException as e:
            raise TrueNASError(f"Failed to get NFS shares: {str(e)}")

    def get_snapshots(self, dataset: Optional[str] = None) -> List[SnapshotInfo]:
        """Get all snapshots, optionally filtered by dataset.
        
        Args:
            dataset: Dataset name to filter snapshots
            
        Returns:
            List of SnapshotInfo objects
        """
        try:
            params = {}
            if dataset:
                params["dataset"] = dataset
            
            response = self.session.get(
                f"{self.base_url}/zfs/snapshot",
                params=params,
                timeout=self.config.timeout
            )
            response.raise_for_status()
            
            snapshots = []
            for snap_data in response.json():
                # Parse creation time
                creation_timestamp = snap_data.get("properties", {}).get("creation", {}).get("value", "0")
                creation_time = datetime.fromtimestamp(int(creation_timestamp))
                
                snapshot = SnapshotInfo(
                    name=snap_data.get("snapshot_name", ""),
                    dataset=snap_data.get("dataset", ""),
                    creation_time=creation_time,
                    used_size=int(snap_data.get("properties", {}).get("used", {}).get("value", "0")),
                    referenced_size=int(snap_data.get("properties", {}).get("referenced", {}).get("value", "0")),
                    full_name=snap_data.get("id", ""),
                )
                snapshots.append(snapshot)
            
            logger.info(f"Found {len(snapshots)} snapshots")
            return snapshots
            
        except requests.exceptions.RequestException as e:
            raise TrueNASError(f"Failed to get snapshots: {str(e)}")

    def get_volume_snapshots(self, volume_name: str) -> List[SnapshotInfo]:
        """Get snapshots for a specific volume.
        
        Args:
            volume_name: Name of the volume (e.g., 'pvc-abc123')
            
        Returns:
            List of SnapshotInfo objects
        """
        # Look for snapshots in common dataset paths
        dataset_paths = [
            f"tank/k8s/volumes/{volume_name}",
            f"tank/k8s/nfs/{volume_name}",
            f"pool0/k8s/volumes/{volume_name}",
            f"pool0/k8s/nfs/{volume_name}",
        ]
        
        all_snapshots = []
        for dataset_path in dataset_paths:
            try:
                params = {"dataset__startswith": dataset_path}
                response = self.session.get(
                    f"{self.base_url}/zfs/snapshot",
                    params=params,
                    timeout=self.config.timeout
                )
                
                if response.status_code == 200:
                    for snap_data in response.json():
                        creation_timestamp = snap_data.get("properties", {}).get("creation", {}).get("value", "0")
                        creation_time = datetime.fromtimestamp(int(creation_timestamp))
                        
                        snapshot = SnapshotInfo(
                            name=snap_data.get("snapshot_name", ""),
                            dataset=snap_data.get("dataset", ""),
                            creation_time=creation_time,
                            used_size=int(snap_data.get("properties", {}).get("used", {}).get("value", "0")),
                            referenced_size=int(snap_data.get("properties", {}).get("referenced", {}).get("value", "0")),
                            full_name=snap_data.get("id", ""),
                        )
                        all_snapshots.append(snapshot)
            except Exception:
                continue
        
        return all_snapshots

    def create_snapshot(self, dataset: str, name: str, recursive: bool = False) -> Dict[str, Any]:
        """Create a ZFS snapshot.
        
        Args:
            dataset: Dataset to snapshot
            name: Snapshot name
            recursive: Whether to create recursive snapshot
            
        Returns:
            Snapshot information
        """
        try:
            data = {
                "dataset": dataset,
                "name": name,
                "recursive": recursive,
            }
            
            response = self.session.post(
                f"{self.base_url}/zfs/snapshot",
                json=data,
                timeout=self.config.timeout
            )
            response.raise_for_status()
            
            logger.info(f"Created snapshot {dataset}@{name}")
            return response.json()
            
        except requests.exceptions.RequestException as e:
            raise TrueNASError(f"Failed to create snapshot: {str(e)}")

    def delete_snapshot(self, snapshot_id: str) -> bool:
        """Delete a ZFS snapshot.
        
        Args:
            snapshot_id: Full snapshot ID (e.g., 'tank/dataset@snapshot')
            
        Returns:
            True if successful
        """
        try:
            # URL encode the snapshot ID
            encoded_id = quote(snapshot_id, safe='')
            
            response = self.session.delete(
                f"{self.base_url}/zfs/snapshot/id/{encoded_id}",
                timeout=self.config.timeout
            )
            response.raise_for_status()
            
            logger.info(f"Deleted snapshot {snapshot_id}")
            return True
            
        except requests.exceptions.RequestException as e:
            raise TrueNASError(f"Failed to delete snapshot: {str(e)}")

    def get_dataset_usage(self, dataset: str) -> Dict[str, Any]:
        """Get detailed usage information for a dataset.
        
        Args:
            dataset: Dataset name
            
        Returns:
            Usage information dictionary
        """
        try:
            response = self.session.get(
                f"{self.base_url}/pool/dataset",
                params={"id": dataset},
                timeout=self.config.timeout
            )
            response.raise_for_status()
            
            data = response.json()
            if not data:
                raise TrueNASError(f"Dataset {dataset} not found")
            
            dataset_info = data[0]
            return {
                "used": dataset_info.get("used", {}).get("value", 0),
                "available": dataset_info.get("available", {}).get("value", 0),
                "referenced": dataset_info.get("referenced", {}).get("value", 0),
                "quota": dataset_info.get("quota", {}).get("value"),
                "children": dataset_info.get("children", []),
            }
            
        except requests.exceptions.RequestException as e:
            raise TrueNASError(f"Failed to get dataset usage: {str(e)}")

    def find_orphaned_volumes(self, k8s_volume_names: List[str]) -> List[OrphanedVolume]:
        """Find TrueNAS volumes that don't have corresponding K8s volumes.
        
        Args:
            k8s_volume_names: List of volume names from Kubernetes
            
        Returns:
            List of orphaned volumes
        """
        orphans = []
        k8s_names_set = set(k8s_volume_names)
        
        # Check iSCSI volumes
        try:
            volumes = self.get_volumes()
            for volume in volumes:
                if volume.name not in k8s_names_set:
                    orphan = OrphanedVolume(
                        name=volume.name,
                        path=volume.path,
                        type="iscsi",
                        size=volume.size,
                    )
                    orphans.append(orphan)
        except Exception as e:
            logger.error(f"Failed to check iSCSI volumes: {str(e)}")
        
        # Check NFS shares
        try:
            shares = self.get_nfs_shares()
            for share in shares:
                path = share.get("path", "")
                # Extract volume name from path (e.g., /mnt/tank/k8s/nfs/pvc-xxx)
                if "/k8s/nfs/" in path:
                    volume_name = path.split("/k8s/nfs/")[-1]
                    if volume_name and volume_name not in k8s_names_set:
                        orphan = OrphanedVolume(
                            name=volume_name,
                            path=path,
                            type="nfs",
                        )
                        orphans.append(orphan)
        except Exception as e:
            logger.error(f"Failed to check NFS shares: {str(e)}")
        
        logger.info(f"Found {len(orphans)} orphaned volumes in TrueNAS")
        return orphans

    def _get_all_pages(self, endpoint: str, params: Optional[Dict] = None) -> List[Dict]:
        """Get all pages from a paginated endpoint.
        
        Args:
            endpoint: API endpoint
            params: Query parameters
            
        Returns:
            Combined list of all items
        """
        if params is None:
            params = {}
        
        all_items = []
        offset = 0
        limit = 50  # TrueNAS default page size
        
        while True:
            params.update({"offset": offset, "limit": limit})
            
            response = self.session.get(
                f"{self.base_url}{endpoint}",
                params=params,
                timeout=self.config.timeout
            )
            response.raise_for_status()
            
            items = response.json()
            if not items:
                break
            
            all_items.extend(items)
            
            # Check if there are more pages
            total_count = response.headers.get("X-Total-Count")
            if total_count and offset + limit >= int(total_count):
                break
            
            offset += limit
        
        return all_items