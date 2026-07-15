# Afero-OSS: Alibaba Cloud OSS Implementation for Afero File System

## Project Introduction

> Based on the project: [spf13/afero](https://github.com/spf13/afero)

> [中文文档](./README_CN.md)

Afero-OSS is an implementation of Alibaba Cloud Object Storage Service (OSS) based on the Afero file system interface. It allows developers to operate Alibaba Cloud OSS as easily as using a local file system, providing a unified and flexible file storage solution.

## Features

- 🔄 Full compatibility with Afero file system interface
- 🚀 Support for Alibaba Cloud OSS v2 SDK
- 📂 Provides standard file and directory operations
- 🔒 Concurrent-safe file handles with write-conflict detection within one `Fs`
- 💾 Configurable preload filesystem (100MB default limit for memory preloading)
- ⚡ Dirty-only synchronization with automatic persistence on close

## Installation

Install using Go modules:

```bash
go get github.com/messikiller/afero-oss
```

## Quick Start

```go
package main

import (
    "github.com/messikiller/afero-oss"
)

func main() {
    // Create OSS file system instance
    ossFs := ossfs.NewOssFs(
        "your-access-key-id", 
        "your-access-key-secret", 
        "your-region", 
        "your-bucket-name",
        //ossfs.OSSWithEndpoint("oss-cn-hangzhou-internal.aliyuncs.com"),
        //ossfs.OSSWithUseInternalEndpoint()
        //Configure the OSS client using ossfs.OSSWithXXX(), or you can customize an OSSOptionFunc.
    )

    // Operate OSS like a local file system
    file, err := ossFs.Create("example.txt")
    if err != nil {
        panic(err)
    }
    if _, err := file.WriteString("Hello, Afero-OSS!"); err != nil {
        panic(err)
    }
    // Close automatically uploads dirty data.
    if err := file.Close(); err != nil {
        panic(err)
    }
}
```

## Main Functionalities

- File creation, reading, writing, and deletion
- Directory management (creation, listing)
- File metadata retrieval
- File preloading and synchronization

## Best Practices

### Synchronization

Per-write automatic synchronization is disabled by default, but `Close()` always uploads dirty data. Call `Sync()` only when data must be persisted while the file remains open:

```go
file, _ := ossFs.Create("example.txt")
file.WriteString("Hello, World!")
file.Sync()  // Optional: persist before Close
file.Close() // Does not upload again when there are no new writes
```

### File Size Considerations

Memory preloading has a default 100MB limit. For larger files, use a disk-backed filesystem and explicitly disable the limit:

```go
ossFs := ossfs.NewOssFs(...)
ossFs.WithPreloadFs(afero.NewBasePathFs(afero.NewOsFs(), "/tmp")).WithMaxPreloadSize(0)
```

## Limitations

- Does not support file system-specific operations like `Chmod`, `Chown`, `Chtimes`
- Depends on Alibaba Cloud OSS service
- Writes preload and re-upload the complete object, so frequent random updates to very large files are inefficient
- Configure the limit with `WithMaxPreloadSize`; zero means unlimited
- One `Fs` returns `ErrWriteConflict` for stale local handles; writes from separate processes still require application-level coordination

## Contributing

Welcome to submit Issues and Pull Requests!

## License

This project is open-sourced under the MIT License.

## Dependencies

- Afero: File system abstraction interface
- Alibaba Cloud OSS Go SDK v2
