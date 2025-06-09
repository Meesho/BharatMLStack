"""
gRPC Feature Client - High-performance gRPC client for feature operations

Based on Go SDK patterns for authentication, context handling, and API semantics.
"""

import asyncio
import logging
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from contextlib import asynccontextmanager
from typing import List, Dict, Any, Optional, Union, Tuple

from bharatml_commons.sdk_template import BaseSDKClient
from bharatml_commons.column_utils import clean_column_name
from bharatml_commons.feature_utils import get_fgs_to_feature_mappings
from .config import GRPCClientConfig

# Set up logging
logger = logging.getLogger(__name__)

# gRPC authentication headers (from Go SDK)
HEADER_CALLER_ID = "ONLINE-FEATURE-STORE-CALLER-ID"
HEADER_CALLER_TOKEN = "ONLINE-FEATURE-STORE-AUTH-TOKEN"

try:
    import grpc
    from grpc import RpcError
    GRPC_AVAILABLE = True
except ImportError:
    GRPC_AVAILABLE = False
    logger.warning("grpcio not available. Install with: pip install grpcio")

# Try to import protobuf files from bharatml_commons
try:
    # Import persist protobuf messages and gRPC stubs
    from bharatml_commons.proto.persist.persist_pb2 import (
        Query as PersistQuery,
        Result as PersistResult,
        FeatureGroupSchema,
        Data,
        FeatureValues,
        Values,
        Vector
    )
    from bharatml_commons.proto.persist.persist_pb2_grpc import FeatureServiceStub as PersistServiceStub
    
    # Import retrieve protobuf messages and gRPC stubs
    from bharatml_commons.proto.retrieve.retrieve_pb2 import (
        Query as RetrieveQuery,
        Result as RetrieveResult,
        DecodedResult,
        Keys,
        FeatureGroup,
        Feature,
        FeatureSchema,
        Row,
        DecodedRow
    )
    from bharatml_commons.proto.retrieve.retrieve_pb2_grpc import FeatureServiceStub as RetrieveServiceStub
    
    PROTOBUF_AVAILABLE = True
    logger.info("Successfully imported protobuf files from bharatml_commons")
except ImportError as e:
    PROTOBUF_AVAILABLE = False
    logger.warning(f"Could not import protobuf files from bharatml_commons: {e}")
    logger.warning("Run: python -m bharatml_commons.proto.generate_proto to generate protobuf files")


class GRPCFeatureClient(BaseSDKClient):
    """
    High-performance gRPC client for feature operations
    
    Implements the same API semantics as the Go SDK:
    1. persist: Store/persist features to the feature store 
    2. retrieve: Retrieve features in binary protobuf format
    3. retrieveDecoded: Retrieve features in human-readable format
    
    Features:
    - Authentication via metadata headers (same as Go SDK)
    - Batch processing for improved performance
    - Context handling with timeouts
    - Connection pooling and retry logic
    - Proper error handling and recovery
    """
    
    def __init__(self, config: Union[GRPCClientConfig, str, dict]):
        """
        Initialize gRPC Feature Client
        
        Args:
            config: Either a GRPCClientConfig object, server address string, or config dict
        """
        if not GRPC_AVAILABLE:
            raise ImportError(
                "grpcio is required for gRPC Feature Client. "
                "Install it with: pip install grpcio"
            )
        
        if not PROTOBUF_AVAILABLE:
            raise RuntimeError(
                "Protobuf files are not available. Please run: "
                "python -m bharatml_commons.proto.generate_proto to generate them"
            )
        
        # Handle different config input types
        if isinstance(config, str):
            # If string provided, assume it's server address and create minimal config
            self.config = GRPCClientConfig(
                server_address=config,
                job_id="",
                job_token=""
            )
        elif isinstance(config, dict):
            # If dict provided, create config from dict
            self.config = GRPCClientConfig(**config)
        else:
            # Assume it's already a GRPCClientConfig
            self.config = config
        
        # Initialize base SDK client
        super().__init__(
            job_id=self.config.job_id,
            job_token=self.config.job_token
        )
        
        # gRPC components
        self.channel = None
        self.persist_stub = None
        self.retrieve_stub = None
        
        # Batch configuration (same as Go SDK default)
        self.batch_size = 50
        
        # Initialize connection
        self._initialize_connection()
    
    def _initialize_connection(self):
        """Initialize gRPC channel and stubs (following Go SDK patterns)"""
        try:
            # Create channel with load balancing (same as Go SDK)
            options = self.config.get_channel_options()
            options.append(('grpc.lb_policy_name', 'round_robin'))
            
            if self.config.use_tls:
                # Use TLS with skip verify (same as Go SDK)
                credentials = grpc.ssl_channel_credentials()
                self.channel = grpc.secure_channel(
                    self.config.server_address,
                    credentials,
                    options=options
                )
            else:
                # Plain text connection
                self.channel = grpc.insecure_channel(
                    self.config.server_address,
                    options=options
                )
            
            # Initialize gRPC stubs
            self.persist_stub = PersistServiceStub(self.channel)
            self.retrieve_stub = RetrieveServiceStub(self.channel)
            
            logger.info(f"Initialized gRPC channel to {self.config.server_address}")
            
        except Exception as e:
            logger.error(f"Failed to initialize gRPC connection: {e}")
            raise
    
    def _create_metadata(self, additional_metadata: Optional[Dict[str, str]] = None) -> List[Tuple[str, str]]:
        """
        Create metadata for gRPC calls using Go SDK header conventions
        
        Args:
            additional_metadata: Optional additional metadata
            
        Returns:
            List of metadata tuples
        """
        # Use the same header names as Go SDK
        metadata = [
            (HEADER_CALLER_ID, self.config.job_id),
            (HEADER_CALLER_TOKEN, self.config.job_token),
        ]
        
        # Add any additional metadata from config
        for key, value in self.config.metadata.items():
            if key not in [HEADER_CALLER_ID, HEADER_CALLER_TOKEN]:
                metadata.append((key, value))
        
        # Add any additional metadata passed in
        if additional_metadata:
            metadata.extend(additional_metadata.items())
        
        return metadata
    
    def _create_context_with_timeout(self, timeout: Optional[float] = None):
        """
        Create gRPC context with timeout (similar to Go SDK withTimeout)
        
        Args:
            timeout: Optional timeout in seconds (defaults to config timeout)
            
        Returns:
            Tuple of (metadata, timeout_seconds)
        """
        timeout_seconds = timeout if timeout is not None else self.config.timeout_seconds
        metadata = self._create_metadata()
        
        return metadata, timeout_seconds
    
    def persist_features(
        self,
        entity_label: str,
        keys_schema: List[str],
        feature_group_schemas: List[Dict[str, Any]],
        data_rows: List[Dict[str, Any]],
        timeout: Optional[float] = None
    ) -> Dict[str, Any]:
        """
        Persist features to the feature store (following Go SDK PersistFeatures semantics)
        
        Args:
            entity_label: Label for the entity type
            keys_schema: List of key column names
            feature_group_schemas: List of feature group schemas with 'label' and 'feature_labels'
            data_rows: List of data rows to persist
            timeout: Optional timeout override
            
        Returns:
            Result dictionary with operation status
        """
        start_time = time.time()
        
        try:
            # Build feature group schemas
            fg_schemas = []
            for fg_schema in feature_group_schemas:
                if not isinstance(fg_schema, dict) or 'label' not in fg_schema or 'feature_labels' not in fg_schema:
                    raise ValueError(f"Invalid feature group schema: {fg_schema}")
                
                fg_schemas.append(FeatureGroupSchema(
                    label=fg_schema['label'],
                    feature_labels=fg_schema['feature_labels']
                ))
            
            # Build data messages
            data_messages = []
            for row in data_rows:
                # Extract key values (convert all to strings)
                key_values = [str(row.get(key, '')) for key in keys_schema]
                
                # Build feature values for each feature group
                feature_values = []
                for fg_schema in feature_group_schemas:
                    values = Values()
                    
                    # Process features for this group
                    for feature_label in fg_schema['feature_labels']:
                        if feature_label in row:
                            value = row[feature_label]
                            
                            # Handle different data types properly
                            if isinstance(value, list):
                                # Vector data - store in vector field
                                vector_values = Values()
                                if all(isinstance(v, str) for v in value):
                                    vector_values.string_values.extend(value)
                                elif all(isinstance(v, bool) for v in value):
                                    vector_values.bool_values.extend(value)
                                elif all(isinstance(v, int) for v in value):
                                    vector_values.int32_values.extend(value)
                                elif all(isinstance(v, float) for v in value):
                                    vector_values.fp32_values.extend(value)
                                values.vector.append(Vector(values=vector_values))
                            else:
                                # Scalar data
                                if isinstance(value, str):
                                    values.string_values.append(value)
                                elif isinstance(value, bool):
                                    values.bool_values.append(value)
                                elif isinstance(value, int):
                                    if -2**31 <= value < 2**31:
                                        values.int32_values.append(value)
                                    else:
                                        values.int64_values.append(value)
                                elif isinstance(value, float):
                                    values.fp32_values.append(value)
                    
                    feature_values.append(FeatureValues(values=values))
                
                data_messages.append(Data(
                    key_values=key_values,
                    feature_values=feature_values
                ))
            
            # Create query
            query = PersistQuery(
                entity_label=entity_label,
                keys_schema=keys_schema,
                feature_group_schema=fg_schemas,
                data=data_messages
            )
            
            # Make gRPC call with authentication and timeout
            metadata, timeout_seconds = self._create_context_with_timeout(timeout)
            
            response = self.persist_stub.PersistFeatures(
                query,
                metadata=metadata,
                timeout=timeout_seconds
            )
            
            elapsed_time = time.time() - start_time
            logger.info(f"Persist features completed in {elapsed_time:.2f}s for {len(data_rows)} rows")
            
            return {
                'success': True,
                'message': response.message if response else 'Features persisted successfully',
                'entity_label': entity_label,
                'rows_processed': len(data_rows),
                'elapsed_time': elapsed_time
            }
            
        except grpc.RpcError as e:
            elapsed_time = time.time() - start_time
            logger.error(f"gRPC error persisting features: {e.code()} - {e.details()}")
            return {
                'success': False,
                'error': f"gRPC error: {e.code()} - {e.details()}",
                'entity_label': entity_label,
                'elapsed_time': elapsed_time
            }
        except Exception as e:
            elapsed_time = time.time() - start_time
            logger.error(f"Error persisting features: {e}")
            return {
                'success': False,
                'error': str(e),
                'entity_label': entity_label,
                'elapsed_time': elapsed_time
            }
    
    def retrieve_features(
        self,
        entity_label: str,
        feature_groups: List[Dict[str, Any]],
        keys_schema: List[str],
        entity_keys: List[List[str]],
        timeout: Optional[float] = None
    ) -> Dict[str, Any]:
        """
        Retrieve features in binary protobuf format (following Go SDK RetrieveFeatures semantics)
        
        Args:
            entity_label: Label for the entity type
            feature_groups: List of feature groups to retrieve with 'label' and 'feature_labels'
            keys_schema: Schema for entity keys
            entity_keys: List of entity key values (each is a list of strings)
            timeout: Optional timeout override
            
        Returns:
            Result dictionary with retrieved features
        """
        start_time = time.time()
        
        try:
            # Validate inputs
            if not entity_keys:
                raise ValueError("entity_keys cannot be empty")
            
            # Create batched queries for parallel processing (Go SDK pattern)
            batched_queries = self._create_batched_queries(
                entity_label, feature_groups, keys_schema, entity_keys
            )
            
            # Execute queries in parallel (similar to Go SDK fetchResponseFromServer)
            responses = self._execute_parallel_retrieve(batched_queries, timeout)
            
            # Combine results (similar to Go SDK handleResponsesFromChannel)
            combined_result = self._combine_retrieve_responses(responses, entity_label, keys_schema)
            
            elapsed_time = time.time() - start_time
            logger.info(f"Retrieve features completed in {elapsed_time:.2f}s for {len(entity_keys)} keys")
            
            combined_result['elapsed_time'] = elapsed_time
            return combined_result
            
        except Exception as e:
            elapsed_time = time.time() - start_time
            logger.error(f"Error retrieving features: {e}")
            return {
                'success': False,
                'error': str(e),
                'entity_label': entity_label,
                'elapsed_time': elapsed_time
            }
    
    def _create_batched_queries(
        self, 
        entity_label: str, 
        feature_groups: List[Dict[str, Any]], 
        keys_schema: List[str], 
        entity_keys: List[List[str]]
    ) -> List[RetrieveQuery]:
        """
        Create batched queries for parallel processing (similar to Go SDK adapter)
        
        Returns:
            List of RetrieveQuery objects
        """
        # Build feature groups
        fg_messages = []
        for fg in feature_groups:
            if not isinstance(fg, dict) or 'label' not in fg or 'feature_labels' not in fg:
                raise ValueError(f"Invalid feature group: {fg}")
            
            fg_messages.append(FeatureGroup(
                label=fg['label'],
                feature_labels=fg['feature_labels']
            ))
        
        # Split entity_keys into batches
        batched_queries = []
        for i in range(0, len(entity_keys), self.batch_size):
            batch_keys = entity_keys[i:i + self.batch_size]
            
            # Build keys messages for this batch
            keys_messages = []
            for key_row in batch_keys:
                keys_messages.append(Keys(cols=[str(k) for k in key_row]))
            
            # Create query for this batch
            query = RetrieveQuery(
                entity_label=entity_label,
                feature_groups=fg_messages,
                keys_schema=keys_schema,
                keys=keys_messages
            )
            
            batched_queries.append(query)
        
        return batched_queries
    
    def _execute_parallel_retrieve(
        self, 
        queries: List[RetrieveQuery], 
        timeout: Optional[float]
    ) -> List[Dict[str, Any]]:
        """
        Execute retrieve queries in parallel (similar to Go SDK goroutines)
        
        Returns:
            List of response dictionaries
        """
        responses = []
        metadata, timeout_seconds = self._create_context_with_timeout(timeout)
        
        # Use ThreadPoolExecutor for parallel execution
        with ThreadPoolExecutor(max_workers=min(len(queries), 10)) as executor:
            # Submit all queries
            future_to_query = {
                executor.submit(
                    self._single_retrieve_call, 
                    query, 
                    metadata, 
                    timeout_seconds
                ): query for query in queries
            }
            
            # Collect results
            for future in as_completed(future_to_query):
                try:
                    response = future.result()
                    responses.append(response)
                except Exception as e:
                    logger.error(f"Error in parallel retrieve: {e}")
                    responses.append({
                        'success': False,
                        'error': str(e)
                    })
        
        return responses
    
    def _single_retrieve_call(
        self, 
        query: RetrieveQuery, 
        metadata: List[Tuple[str, str]], 
        timeout: float
    ) -> Dict[str, Any]:
        """
        Make a single retrieve gRPC call (similar to Go SDK contactServer)
        
        Returns:
            Response dictionary
        """
        try:
            response = self.retrieve_stub.RetrieveFeatures(
                query,
                metadata=metadata,
                timeout=timeout
            )
            
            return {
                'success': True,
                'response': response
            }
            
        except grpc.RpcError as e:
            logger.error(f"gRPC error in retrieve: {e.code()} - {e.details()}")
            return {
                'success': False,
                'error': f"gRPC error: {e.code()} - {e.details()}"
            }
        except Exception as e:
            logger.error(f"Unexpected error in retrieve: {e}")
            return {
                'success': False,
                'error': str(e)
            }
    
    def _combine_retrieve_responses(
        self, 
        responses: List[Dict[str, Any]], 
        entity_label: str, 
        keys_schema: List[str]
    ) -> Dict[str, Any]:
        """
        Combine multiple retrieve responses (similar to Go SDK handleResponsesFromChannel)
        
        Returns:
            Combined result dictionary
        """
        combined_result = {
            'success': True,
            'entity_label': entity_label,
            'keys_schema': keys_schema,
            'feature_schemas': [],
            'rows': []
        }
        
        feature_schemas_set = False
        
        for response_data in responses:
            if not response_data.get('success', False):
                # Log error but continue processing other responses
                logger.warning(f"Skipping failed response: {response_data.get('error', 'Unknown error')}")
                continue
            
            response = response_data.get('response')
            if not response:
                continue
            
            # Set feature schemas from first successful response
            if not feature_schemas_set and response.feature_schemas:
                combined_result['feature_schemas'] = [
                    {
                        'feature_group_label': fs.feature_group_label,
                        'features': [
                            {'label': f.label, 'column_idx': f.column_idx}
                            for f in fs.features
                        ]
                    }
                    for fs in response.feature_schemas
                ]
                feature_schemas_set = True
            
            # Add rows from this response
            for row in response.rows:
                combined_result['rows'].append({
                    'keys': list(row.keys),
                    'columns': list(row.columns)  # Binary data
                })
        
        # Check if we got any successful responses
        if not combined_result['rows']:
            combined_result['success'] = False
            combined_result['error'] = "No successful responses received"
        
        return combined_result
    
    def retrieve_decoded_features(
        self,
        entity_label: str,
        feature_groups: List[Dict[str, Any]],
        keys_schema: List[str],
        entity_keys: List[List[str]],
        timeout: Optional[float] = None
    ) -> Dict[str, Any]:
        """
        Retrieve features in decoded/human-readable format (following Go SDK RetrieveDecodedFeatures)
        
        Args:
            entity_label: Label for the entity type
            feature_groups: List of feature groups to retrieve with 'label' and 'feature_labels'
            keys_schema: Schema for entity keys
            entity_keys: List of entity key values (each is a list of strings)
            timeout: Optional timeout override
            
        Returns:
            Result dictionary with decoded features (strings instead of binary)
        """
        start_time = time.time()
        
        try:
            # Validate inputs
            if not entity_keys:
                raise ValueError("entity_keys cannot be empty")
            
            # Create batched queries (same as retrieve_features)
            batched_queries = self._create_batched_queries(
                entity_label, feature_groups, keys_schema, entity_keys
            )
            
            # Execute queries in parallel for decoded results
            responses = self._execute_parallel_retrieve_decoded(batched_queries, timeout)
            
            # Combine decoded results
            combined_result = self._combine_decoded_responses(responses, entity_label, keys_schema)
            
            elapsed_time = time.time() - start_time
            logger.info(f"Retrieve decoded features completed in {elapsed_time:.2f}s for {len(entity_keys)} keys")
            
            combined_result['elapsed_time'] = elapsed_time
            return combined_result
            
        except Exception as e:
            elapsed_time = time.time() - start_time
            logger.error(f"Error retrieving decoded features: {e}")
            return {
                'success': False,
                'error': str(e),
                'entity_label': entity_label,
                'elapsed_time': elapsed_time
            }
    
    def _execute_parallel_retrieve_decoded(
        self, 
        queries: List[RetrieveQuery], 
        timeout: Optional[float]
    ) -> List[Dict[str, Any]]:
        """
        Execute retrieve decoded queries in parallel
        
        Returns:
            List of response dictionaries
        """
        responses = []
        metadata, timeout_seconds = self._create_context_with_timeout(timeout)
        
        # Use ThreadPoolExecutor for parallel execution
        with ThreadPoolExecutor(max_workers=min(len(queries), 10)) as executor:
            # Submit all queries
            future_to_query = {
                executor.submit(
                    self._single_retrieve_decoded_call, 
                    query, 
                    metadata, 
                    timeout_seconds
                ): query for query in queries
            }
            
            # Collect results
            for future in as_completed(future_to_query):
                try:
                    response = future.result()
                    responses.append(response)
                except Exception as e:
                    logger.error(f"Error in parallel retrieve decoded: {e}")
                    responses.append({
                        'success': False,
                        'error': str(e)
                    })
        
        return responses
    
    def _single_retrieve_decoded_call(
        self, 
        query: RetrieveQuery, 
        metadata: List[Tuple[str, str]], 
        timeout: float
    ) -> Dict[str, Any]:
        """
        Make a single retrieve decoded gRPC call
        
        Returns:
            Response dictionary
        """
        try:
            response = self.retrieve_stub.RetrieveDecodedResult(
                query,
                metadata=metadata,
                timeout=timeout
            )
            
            return {
                'success': True,
                'response': response
            }
            
        except grpc.RpcError as e:
            logger.error(f"gRPC error in retrieve decoded: {e.code()} - {e.details()}")
            return {
                'success': False,
                'error': f"gRPC error: {e.code()} - {e.details()}"
            }
        except Exception as e:
            logger.error(f"Unexpected error in retrieve decoded: {e}")
            return {
                'success': False,
                'error': str(e)
            }
    
    def _combine_decoded_responses(
        self, 
        responses: List[Dict[str, Any]], 
        entity_label: str, 
        keys_schema: List[str]
    ) -> Dict[str, Any]:
        """
        Combine multiple decoded retrieve responses
        
        Returns:
            Combined result dictionary
        """
        combined_result = {
            'success': True,
            'entity_label': entity_label,
            'keys_schema': keys_schema,
            'feature_schemas': [],
            'rows': []
        }
        
        feature_schemas_set = False
        
        for response_data in responses:
            if not response_data.get('success', False):
                logger.warning(f"Skipping failed decoded response: {response_data.get('error', 'Unknown error')}")
                continue
            
            response = response_data.get('response')
            if not response:
                continue
            
            # Set feature schemas from first successful response
            if not feature_schemas_set and response.feature_schemas:
                combined_result['feature_schemas'] = [
                    {
                        'feature_group_label': fs.feature_group_label,
                        'features': [
                            {'label': f.label, 'column_idx': f.column_idx}
                            for f in fs.features
                        ]
                    }
                    for fs in response.feature_schemas
                ]
                feature_schemas_set = True
            
            # Add rows from this response
            for row in response.rows:
                combined_result['rows'].append({
                    'keys': list(row.keys),
                    'columns': list(row.columns)  # Decoded string data
                })
        
        # Check if we got any successful responses
        if not combined_result['rows']:
            combined_result['success'] = False
            combined_result['error'] = "No successful decoded responses received"
        
        return combined_result
    
    def set_batch_size(self, batch_size: int):
        """
        Set the batch size for parallel requests
        
        Args:
            batch_size: Number of requests to batch together
        """
        if batch_size <= 0:
            raise ValueError("batch_size must be positive")
        
        self.batch_size = batch_size
        logger.info(f"Updated batch size to {batch_size}")
    
    def get_connection_info(self) -> Dict[str, Any]:
        """
        Get information about the current connection
        
        Returns:
            Dictionary with connection details
        """
        return {
            'server_address': self.config.server_address,
            'use_tls': self.config.use_tls,
            'timeout_seconds': self.config.timeout_seconds,
            'batch_size': self.batch_size,
            'job_id': self.config.job_id,
            'channel_state': str(self.channel.get_state()) if self.channel else 'Not initialized'
        }
    
    def health_check(self) -> Dict[str, Any]:
        """
        Check if the gRPC service is healthy
        
        Returns:
            Health status dictionary
        """
        try:
            # Simple connectivity check
            grpc.channel_ready_future(self.channel).result(timeout=5.0)
            return {
                'status': 'healthy',
                'server_address': self.config.server_address,
                'connection': 'ready'
            }
        except Exception as e:
            return {
                'status': 'unhealthy',
                'server_address': self.config.server_address,
                'error': str(e)
            }
    
    def close(self):
        """Close the gRPC channel"""
        if self.channel:
            self.channel.close()
            logger.info("gRPC channel closed")
    
    def __enter__(self):
        """Context manager entry"""
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit"""
        self.close()


# Utility functions for working with feature data
def create_feature_group_schema(label: str, feature_labels: List[str]) -> Dict[str, Any]:
    """
    Helper function to create feature group schema
    
    Args:
        label: Feature group label
        feature_labels: List of feature labels
        
    Returns:
        Feature group schema dictionary
    """
    return {
        'label': label,
        'feature_labels': feature_labels
    }


def clean_feature_names(feature_names: List[str]) -> List[str]:
    """
    Clean feature names using shared utilities
    
    Args:
        feature_names: List of feature names to clean
        
    Returns:
        List of cleaned feature names
    """
    return [clean_column_name(name) for name in feature_names] 