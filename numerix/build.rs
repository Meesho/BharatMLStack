fn main() {
    tonic_build::configure()
        .out_dir("src/protos/proto_gen")
        .protoc_arg("--experimental_allow_proto3_optional")
        .compile(&["src/protos/proto/numerix.proto"], &["src/protos/proto"])
        .expect("Failed to compile proto file");
}
