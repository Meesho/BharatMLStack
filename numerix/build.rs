fn main() {
    tonic_build::configure()
        .out_dir("src/protos/proto_gen")
        .compile(
            &["src/protos/proto/numerix.proto"],
            &["src/protos/proto"]
        )
        .expect("Failed to compile proto file");
}
