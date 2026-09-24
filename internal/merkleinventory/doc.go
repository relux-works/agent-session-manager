// Package merkleinventory builds and serves the in-process Mesh RPC 2 namespace
// inventories. It owns no network channel or durable object storage. Callers
// must provide schema-validated immutable objects through AddJSON and complete
// raw blobs through AddBlob. Host Channel dispatch and durable union state
// remain with their respective owners.
package merkleinventory
