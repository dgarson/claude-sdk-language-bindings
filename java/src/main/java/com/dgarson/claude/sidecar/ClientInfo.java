package com.dgarson.claude.sidecar;

/**
 * Identification information sent to the sidecar when attaching to a session.
 */
public record ClientInfo(String name, String version, String protocol) {

    public ClientInfo(String name, String version) {
        this(name, version, "v1");
    }

    public static Builder builder() {
        return new Builder();
    }

    public static final class Builder {
        private String name = "";
        private String version = "";
        private String protocol = "v1";

        private Builder() {}

        public Builder name(String name) {
            this.name = name;
            return this;
        }

        public Builder version(String version) {
            this.version = version;
            return this;
        }

        public Builder protocol(String protocol) {
            this.protocol = protocol;
            return this;
        }

        public ClientInfo build() {
            return new ClientInfo(name, version, protocol);
        }
    }
}
