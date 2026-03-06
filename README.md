# Afero-OSS: Alibaba Cloud OSS Implementation for Afero File System

## Project Introduction

> Based on the project: [spf13/afero](https://github.com/spf13/afero)

> [中文文档](./README_CN.md)

Afero-OSS is an implementation of Alibaba Cloud Object Storage Service (OSS) based on the Afero file system interface. It allows developers to operate Alibaba Cloud OSS as easily as using a local file system, providing a unified and flexible file storage solution.

## Features

- 🔄 Full compatibility with Afero file system interface
- 🚀 Support for Alibaba Cloud OSS v2 SDK
- 📂 Provides standard file and directory operations
- 🔒 Concurrent safe design
- 💾 Memory preloading mechanism for efficient file operations (with 100MB size limit)
- ⚡ Manual sync control for optimized performance

## Installation

Install using Go modules:

```bash
go get github.com/messikiller/afero-oss
```

## Quick Start

```go
package main

import (
    "github.com/spf13/afero"
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
    defer file.Close()

    file.WriteString("Hello, Afero-OSS!")
    
    // Important: Manually sync to upload changes to OSS
    file.Sync()
}
```

## Main Functionalities

- File creation, reading, writing, and deletion
- Directory management (creation, listing)
- File metadata retrieval
- File preloading and synchronization

## Best Practices

### Manual Synchronization

By default, `autoSync` is disabled for better performance. You need to manually call `Sync()` to upload changes to OSS:

```go
file, _ := ossFs.Create("example.txt")
file.WriteString("Hello, World!")
file.Sync()  // Upload to OSS
file.Close()
```

### File Size Considerations

The preload mechanism has a 100MB size limit. For larger files, use a disk-based preload filesystem:

```go
ossFs := ossfs.NewOssFs(...)
ossFs.WithPreloadFs(afero.NewBasePathFs(afero.NewOsFs(), "/tmp"))
```

## Limitations

- Does not support file system-specific operations like `Chmod`, `Chown`, `Chtimes`
- Depends on Alibaba Cloud OSS service
- Preload size limit: 100MB per file (configurable via `WithPreloadFs`)
- Requires manual `Sync()` calls to persist changes (autoSync is disabled by default)

## Contributing

Welcome to submit Issues and Pull Requests!

## License

This project is open-sourced under the MIT License.

## Dependencies

- Afero: File system abstraction interface
- Alibaba Cloud OSS Go SDK v2