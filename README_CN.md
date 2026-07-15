# Afero-OSS: 阿里云OSS的Afero文件系统实现

## 项目简介

> 基于项目：[spf13/afero](https://github.com/spf13/afero)

Afero-OSS 是一个基于 Afero 文件系统接口的阿里云对象存储（OSS）实现。允许开发者像使用本地文件系统一样操作阿里云OSS，提供了一种统一且灵活的文件存储解决方案。

## 特性

- 🔄 完全兼容 Afero 文件系统接口
- 🚀 支持阿里云OSS v2 SDK
- 📂 提供文件和目录的标准操作
- 🔒 文件句柄并发安全，同一 `Fs` 实例内检测写冲突
- 💾 可配置的预加载文件系统（内存模式默认限制 100MB）
- ⚡ 仅同步脏数据，关闭文件时自动持久化

## 安装

使用 Go 模块安装：

```bash
go get github.com/messikiller/afero-oss
```

## 快速开始

```go
package main

import (
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
    if _, err := file.WriteString("Hello, Afero-OSS!"); err != nil {
        panic(err)
    }
    // Close 会自动上传脏数据。
    if err := file.Close(); err != nil {
        panic(err)
    }
}
```

## 主要功能

- 文件创建、读取、写入、删除
- 目录管理（创建、列出）
- 文件元数据获取
- 文件预加载和同步

## 最佳实践

### 同步行为

默认禁用逐次写入自动同步，但 `Close()` 始终会上传脏数据。只有在文件保持打开期间需要提前持久化时，才需要调用 `Sync()`：

```go
file, _ := ossFs.Create("example.txt")
file.WriteString("Hello, World!")
file.Sync()  // 可选：在 Close 前提前上传
file.Close() // 没有新写入时不会重复上传
```

### 文件大小注意事项

默认内存预加载限制为 100MB。处理更大文件时，请使用磁盘文件系统并显式关闭限制：

```go
ossFs := ossfs.NewOssFs(...)
ossFs.WithPreloadFs(afero.NewBasePathFs(afero.NewOsFs(), "/tmp")).WithMaxPreloadSize(0)
```

## 局限性

- 不支持 `Chmod`、`Chown`、`Chtimes` 等文件系统特定操作
- 依赖阿里云 OSS 服务
- 写入需要预加载并重新上传完整对象，不适合频繁随机修改超大文件
- 预加载限制通过 `WithMaxPreloadSize` 配置，`0` 表示不限制
- 同一 `Fs` 实例会返回 `ErrWriteConflict` 防止旧句柄静默覆盖；跨进程写入仍需业务层协调

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

本项目基于 MIT 许可证开源。

## 依赖

- Afero: 文件系统抽象接口
- 阿里云OSS Go SDK v2
