fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Use tonic-prost-build as shown in tonic-h3 tests
    tonic_prost_build::compile_protos("proto/retrieve.proto")?;
    Ok(())
}

