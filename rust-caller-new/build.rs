fn main() {
    tonic_build::configure()
        .build_server(true)
        .compile(
            &["proto/retrieve.proto"],
            &["proto"],
        )
        .unwrap();

    println!("cargo:rerun-if-changed=proto/retrieve.proto");
    println!("cargo:rerun-if-changed=proto/");
}

