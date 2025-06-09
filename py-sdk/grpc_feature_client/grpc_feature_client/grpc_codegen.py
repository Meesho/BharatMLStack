"""
gRPC Code Generation Utilities

This module provides utilities to generate gRPC Python stubs from protobuf files.
"""

import os
import subprocess
import sys
from pathlib import Path
from typing import List, Optional


def generate_grpc_stubs(
    proto_dir: str = "../spark_feature_push_client/proto",
    output_dir: Optional[str] = None
) -> bool:
    """
    Generate gRPC Python stubs from protobuf files
    
    Args:
        proto_dir: Directory containing .proto files
        output_dir: Output directory for generated files (defaults to proto_dir)
        
    Returns:
        True if successful, False otherwise
    """
    if output_dir is None:
        output_dir = proto_dir
    
    proto_path = Path(proto_dir)
    output_path = Path(output_dir)
    
    if not proto_path.exists():
        print(f"Error: Proto directory {proto_dir} does not exist")
        return False
    
    # Find all .proto files
    proto_files = list(proto_path.glob("*.proto"))
    
    if not proto_files:
        print(f"Error: No .proto files found in {proto_dir}")
        return False
    
    print(f"Found {len(proto_files)} proto files: {[f.name for f in proto_files]}")
    
    # Generate gRPC stubs for each proto file
    success = True
    for proto_file in proto_files:
        print(f"Generating gRPC stubs for {proto_file.name}...")
        
        # Determine output directory for this proto file
        proto_name = proto_file.stem
        proto_output_dir = output_path / proto_name
        proto_output_dir.mkdir(exist_ok=True)
        
        try:
            # Run protoc to generate Python and gRPC files
            cmd = [
                "python", "-m", "grpc_tools.protoc",
                f"--proto_path={proto_path}",
                f"--python_out={proto_output_dir}",
                f"--grpc_python_out={proto_output_dir}",
                str(proto_file)
            ]
            
            result = subprocess.run(cmd, capture_output=True, text=True, check=True)
            print(f"✓ Generated stubs for {proto_file.name}")
            
        except subprocess.CalledProcessError as e:
            print(f"✗ Error generating stubs for {proto_file.name}: {e}")
            print(f"stdout: {e.stdout}")
            print(f"stderr: {e.stderr}")
            success = False
        except FileNotFoundError:
            print("Error: grpcio-tools not found. Install it with: pip install grpcio-tools")
            return False
    
    if success:
        print("\n✓ All gRPC stubs generated successfully!")
        print(f"Generated files are in: {output_path}")
        print("\nTo use the generated stubs, update the imports in client.py")
    
    return success


def install_grpc_tools() -> bool:
    """
    Install grpcio-tools package required for gRPC code generation
    
    Returns:
        True if successful, False otherwise
    """
    try:
        print("Installing grpcio-tools...")
        result = subprocess.run([
            sys.executable, "-m", "pip", "install", "grpcio-tools"
        ], capture_output=True, text=True, check=True)
        
        print("✓ grpcio-tools installed successfully!")
        return True
        
    except subprocess.CalledProcessError as e:
        print(f"✗ Error installing grpcio-tools: {e}")
        print(f"stderr: {e.stderr}")
        return False


def check_grpc_tools() -> bool:
    """
    Check if grpcio-tools is available
    
    Returns:
        True if available, False otherwise
    """
    try:
        import grpc_tools
        return True
    except ImportError:
        return False


def setup_grpc_environment() -> bool:
    """
    Set up the complete gRPC development environment
    
    Returns:
        True if successful, False otherwise
    """
    print("Setting up gRPC environment for BharatML Stack...")
    
    # Check if grpcio-tools is available
    if not check_grpc_tools():
        print("grpcio-tools not found, installing...")
        if not install_grpc_tools():
            return False
    else:
        print("✓ grpcio-tools is already available")
    
    # Generate gRPC stubs
    if not generate_grpc_stubs():
        return False
    
    print("\n🎉 gRPC environment setup complete!")
    print("\nNext steps:")
    print("1. Update the imports in grpc_feature_client/client.py to use the generated stubs")
    print("2. Replace the _call_*_service placeholder methods with real gRPC calls")
    print("3. Test your gRPC client with a running gRPC server")
    
    return True


if __name__ == "__main__":
    import argparse
    
    parser = argparse.ArgumentParser(description="Generate gRPC stubs for BharatML Stack")
    parser.add_argument(
        "--proto-dir", 
        default="../spark_feature_push_client/proto",
        help="Directory containing .proto files"
    )
    parser.add_argument(
        "--output-dir",
        help="Output directory for generated files (defaults to proto-dir)"
    )
    parser.add_argument(
        "--setup",
        action="store_true",
        help="Set up complete gRPC environment (install tools + generate stubs)"
    )
    
    args = parser.parse_args()
    
    if args.setup:
        success = setup_grpc_environment()
    else:
        success = generate_grpc_stubs(args.proto_dir, args.output_dir)
    
    sys.exit(0 if success else 1) 