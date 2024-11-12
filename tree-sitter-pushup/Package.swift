// swift-tools-version:5.3
import PackageDescription

let package = Package(
    name: "TreeSitterPushup",
    products: [
        .library(name: "TreeSitterPushup", targets: ["TreeSitterPushup"]),
    ],
    dependencies: [
        .package(url: "https://github.com/ChimeHQ/SwiftTreeSitter", from: "0.8.0"),
    ],
    targets: [
        .target(
            name: "TreeSitterPushup",
            dependencies: [],
            path: ".",
            sources: [
                "src/parser.c",
                // NOTE: if your language has an external scanner, add it here.
            ],
            resources: [
                .copy("queries")
            ],
            publicHeadersPath: "bindings/swift",
            cSettings: [.headerSearchPath("src")]
        ),
        .testTarget(
            name: "TreeSitterPushupTests",
            dependencies: [
                "SwiftTreeSitter",
                "TreeSitterPushup",
            ],
            path: "bindings/swift/TreeSitterPushupTests"
        )
    ],
    cLanguageStandard: .c11
)
