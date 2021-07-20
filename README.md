ETCD Adapter
============

ETCD Adapter mimics the [ETCD V3 API](https://etcd.io/docs/v3.3/rfc/) best effort.

ETCD V3 APIs Conformance
========================

Not all APIs and their options are implemented. The following table gives the details about the conformance for each API.

KV
--

### Range

Features of Range API defined in [RangeRequest](https://github.com/etcd-io/etcd/blob/main/api/etcdserverpb/rpc.proto#L404), so some feature names below are also the option name in the `RangeRequest`.

| Feature | Implemented |
| ------- | -----------|
| SortTarget | ❌ |
| Count | ❌ |
| KeysOnly | ❌ |
| CountOnly | ❌ |
| Revision specific Limitations | ❌ |
| Specific revision | ❌ |
| Range Query | ✅ |
| Exact Query | ✅ |

### Put

All related features about `PutRequest` are not yet supported.

### DeleteRange

Not yet Implemented.

### Txn

Not yet implemented.

### Compact

Not yet implemented.
