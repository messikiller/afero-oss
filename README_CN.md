# Afero-OSS: 阿里云OSS的Afero文件系统实现

## 项目简介

> 基于项目：[spf13/afero](https://github.com/spf13/afero)

Afero-OSS 是一个基于 Afero 文件系统接口的阿里云对象存储（OSS）实现。允许开发者像使用本地文件系统一样操作阿里云OSS，提供了一种统一且灵活的文件存储解决方案。

## 特性

- 🔄 完全兼容 Afero 文件系统接口
- 🚀 支持阿里云OSS v2 SDK
- 📂 提供文件和目录的标准操作
- 🔒 并发安全设计
- 💾 内存预加载机制，高效文件操作（100MB 大小限制）
- ⚡ 手动同步控制，优化性能

## 安装

使用 Go 模块安装：

```bash
go get github.com/messikiller/afero-oss
```

## 快速开始

```go
package main

import (
    "github.com/spf13/afero"
    "github.com/messikiller/afero-oss"
)

func main() {
    // 创建OSS文件系统实例
    ossFs := ossfs.NewOssFs(
        "your-access-key-id", 
        "your-access-key-secret", 
        "your-region", 
        "your-bucket-name",
        //ossfs.OSSWithEndpoint("oss-cn-hangzhou-internal.aliyuncs.com"),
        //ossfs.OSSWithUseInternalEndpoint()
        //使用ossfs.OSSWithXXX()配置OSS客户端, 或者你自定义一个OSSOptionFunc类型的函数.
    )

    // 像使用本地文件系统一样操作OSS
    file, err := ossFs.Create("example.txt")
    if err != nil {
        panic(err)
    }
    defer file.Close()

    file.WriteString("Hello, Afero-OSS!")
    
    // 重要：手动同步以上传更改到 OSS
    file.Sync()
}
```

## 主要功能

- 文件创建、读取、写入、删除
- 目录管理（创建、列出）
- 文件元数据获取
- 文件预加载和同步

## 最佳实践

### 手动同步

默认情况下，`autoSync` 已禁用以获得更好的性能。您需要手动调用 `Sync()` 将更改上传到 OSS：

```go
file, _ := ossFs.Create("example.txt")
file.WriteString("Hello, World!")
file.Sync()  // 上传到 OSS
file.Close()
```

### 文件大小注意事项

预加载机制有 100MB 的大小限制。对于更大的文件，请使用基于磁盘的预加载文件系统：

```go
ossFs := ossfs.NewOssFs(...)
ossFs.WithPreloadFs(afero.NewBasePathFs(afero.NewOsFs(), "/tmp"))
```

## 局限性

- 不支持 `Chmod`、`Chown`、`Chtimes` 等文件系统特定操作
- 依赖阿里云 OSS 服务
- 预加载大小限制：每个文件 100MB（可通过 `WithPreloadFs` 配置）
- 需要手动调用 `Sync()` 来持久化更改（默认禁用 autoSync）

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

本项目基于 MIT 许可证开源。

## 依赖

- Afero: 文件系统抽象接口
- 阿里云OSS Go SDK v2