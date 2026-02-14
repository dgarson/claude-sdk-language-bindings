import com.google.protobuf.gradle.*

plugins {
    java
    id("com.google.protobuf") version "0.9.4"
}

group = "com.dgarson.claude"
version = "0.1.0"

java {
    sourceCompatibility = JavaVersion.VERSION_17
    targetCompatibility = JavaVersion.VERSION_17
}

repositories {
    mavenCentral()
}

val grpcVersion = "1.64.0"
val protobufVersion = "4.27.1"

dependencies {
    implementation("io.grpc:grpc-netty-shaded:$grpcVersion")
    implementation("io.grpc:grpc-protobuf:$grpcVersion")
    implementation("io.grpc:grpc-stub:$grpcVersion")
    implementation("com.google.protobuf:protobuf-java:$protobufVersion")
    implementation("com.google.protobuf:protobuf-java-util:$protobufVersion")
    compileOnly("javax.annotation:javax.annotation-api:1.3.2")

    testImplementation("org.junit.jupiter:junit-jupiter:5.10.2")
    testRuntimeOnly("org.junit.platform:junit-platform-launcher")
}

protobuf {
    protoc {
        artifact = "com.google.protobuf:protoc:$protobufVersion"
    }
    plugins {
        create("grpc") {
            artifact = "io.grpc:protoc-gen-grpc-java:$grpcVersion"
        }
    }
    generateProtoTasks {
        all().forEach { task ->
            task.plugins {
                create("grpc")
            }
        }
    }
}

sourceSets {
    main {
        proto {
            srcDir("../proto")
        }
    }
}

tasks.test {
    useJUnitPlatform()
    systemProperty("sidecar.addr", System.getenv("SIDECAR_ADDR") ?: "127.0.0.1:50051")
    systemProperty("sidecar.e2e", System.getenv("SIDECAR_E2E") ?: "")
    systemProperty("sidecar.e2e.live", System.getenv("SIDECAR_E2E_LIVE") ?: "")
    systemProperty("sidecar.e2e.test_mode", System.getenv("SIDECAR_E2E_TEST_MODE") ?: "")
}
