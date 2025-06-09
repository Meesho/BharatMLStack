"""
Protobuf and gRPC code generation for BharatML Stack

This script generates Python protobuf and gRPC stub files from .proto definitions.
"""

import os
import subprocess
import sys
from pathlib import Path
from typing import List, Optional


def install_requirements() -> bool:
    """Install required packages for protobuf generation"""
    try:
        print("Installing required packages...")
        result = subprocess.run([
            sys.executable, "-m", "pip", "install", 
            "grpcio-tools>=1.50.0", "protobuf>=4.24.1"
        ], capture_output=True, text=True, check=True)
        
        print("✓ Required packages installed successfully!")
        return True
        
    except subprocess.CalledProcessError as e:
        print(f"✗ Error installing packages: {e}")
        print(f"stderr: {e.stderr}")
        return False


def generate_python_protobuf(proto_dir: Path, output_dir: Path) -> bool:
    """
    Generate Python protobuf and gRPC files from .proto files
    
    Args:
        proto_dir: Directory containing .proto files
        output_dir: Output directory for generated files
        
    Returns:
        True if successful, False otherwise
    """
    if not proto_dir.exists():
        print(f"Error: Proto directory {proto_dir} does not exist")
        return False
    
    # Find all .proto files
    proto_files = list(proto_dir.glob("*.proto"))
    
    if not proto_files:
        print(f"Error: No .proto files found in {proto_dir}")
        return False
    
    print(f"Found {len(proto_files)} proto files: {[f.name for f in proto_files]}")
    
    # Create output directories for each proto file
    success = True
    for proto_file in proto_files:
        print(f"Generating Python files for {proto_file.name}...")
        
        # Create subdirectory for this proto
        proto_name = proto_file.stem
        proto_output_dir = output_dir / proto_name
        proto_output_dir.mkdir(exist_ok=True)
        
        # Create __init__.py in the subdirectory
        init_file = proto_output_dir / "__init__.py"
        init_file.write_text('"""Generated protobuf files"""\n')
        
        try:
            # Run protoc to generate Python and gRPC files
            cmd = [
                "python", "-m", "grpc_tools.protoc",
                f"--proto_path={proto_dir}",
                f"--python_out={proto_output_dir}",
                f"--grpc_python_out={proto_output_dir}",
                str(proto_file)
            ]
            
            result = subprocess.run(cmd, capture_output=True, text=True, check=True)
            print(f"✓ Generated files for {proto_file.name}")
            
            # Fix imports in generated _pb2_grpc.py files
            grpc_file = proto_output_dir / f"{proto_name}_pb2_grpc.py"
            if grpc_file.exists():
                fix_grpc_imports(grpc_file, proto_name)
            
        except subprocess.CalledProcessError as e:
            print(f"✗ Error generating files for {proto_file.name}: {e}")
            print(f"stdout: {e.stdout}")
            print(f"stderr: {e.stderr}")
            success = False
        except FileNotFoundError:
            print("Error: grpcio-tools not found. Installing...")
            if install_requirements():
                # Retry the command
                try:
                    result = subprocess.run(cmd, capture_output=True, text=True, check=True)
                    print(f"✓ Generated files for {proto_file.name}")
                except Exception as retry_e:
                    print(f"✗ Still failed after installing requirements: {retry_e}")
                    success = False
            else:
                return False
    
    return success


def fix_grpc_imports(grpc_file: Path, proto_name: str):
    """Fix relative imports in generated gRPC files"""
    try:
        content = grpc_file.read_text()
        # Fix import statement to use relative import
        old_import = f"import {proto_name}_pb2 as {proto_name}__pb2"
        new_import = f"from . import {proto_name}_pb2 as {proto_name}__pb2"
        
        if old_import in content:
            content = content.replace(old_import, new_import)
            grpc_file.write_text(content)
            print(f"  ✓ Fixed imports in {grpc_file.name}")
    except Exception as e:
        print(f"  ⚠ Warning: Could not fix imports in {grpc_file.name}: {e}")


def main():
    """Main function to generate all protobuf files"""
    current_dir = Path(__file__).parent
    proto_dir = current_dir
    output_dir = current_dir
    
    print("BharatML Stack Protobuf Generation")
    print("=" * 40)
    print(f"Proto directory: {proto_dir}")
    print(f"Output directory: {output_dir}")
    print()
    
    if generate_python_protobuf(proto_dir, output_dir):
        print("\n🎉 Protobuf generation completed successfully!")
        print("\nGenerated files:")
        for subdir in output_dir.iterdir():
            if subdir.is_dir() and subdir.name in ['persist', 'retrieve']:
                print(f"  {subdir.name}/")
                for file in subdir.glob("*.py"):
                    print(f"    {file.name}")
        
        print("\nNext steps:")
        print("1. Import the generated classes in your Python code")
        print("2. Use the gRPC stubs for service calls")
        print("3. Update SDK imports to use bharatml_common.proto")
        
        return True
    else:
        print("\n❌ Protobuf generation failed!")
        return False


if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1) 