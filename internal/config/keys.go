package config

// Etcd server hostname
const ETCD_ADDRESS = "etcd.address"

// Forces runtime container images to be pulled the first time they are used,
// even if they are locally available (true/false).
const FACTORY_REFRESH_IMAGES = "factory.images.refresh"

// Amount of memory available for the containers pool (in MB)
const POOL_MEMORY_MB = "containers.pool.memory"

// CPUs available for the containers pool (1.0 = 1 core)
const POOL_CPUS = "containers.pool.cpus"
