fn main() -> Result<(), Box<dyn std::error::Error>> {
    tonic_build::configure()
        .compile(&["proto/retrieve.proto", "proto/persist.proto"], &["proto"])?;
    Ok(())
} 