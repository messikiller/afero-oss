package ossfs

import "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"

type OssConfigFunc func(c *oss.Config)

// Specify the endpoint.
func OssWithEndpoint(v string) OssConfigFunc {
	return func(c *oss.Config) {
		c.WithEndpoint(v)
	}
}

// Use an internal endpoint.
func OssWithUseInternalEndpoint() OssConfigFunc {
	return func(c *oss.Config) {
		c.WithUseInternalEndpoint(true)
	}
}

// Use an OSS-accelerated endpoint.
func OssWithUseAccelerateEndpoint() OssConfigFunc {
	return func(c *oss.Config) {
		c.WithUseAccelerateEndpoint(true)
	}
}

// Use a dual-stack endpoint.
func OssWithUseDualStackEndpoint() OssConfigFunc {
	return func(c *oss.Config) {
		c.WithUseDualStackEndpoint(true)
	}
}

// Access OSS by using a custom domain name.
func OssWithUseCName(domain string) OssConfigFunc {
	return func(c *oss.Config) {
		c.WithEndpoint(domain).WithUseCName(true)
	}
}

// Use path request style.
func OssWithUsePathStyle() OssConfigFunc {
	return func(c *oss.Config) {
		c.WithUsePathStyle(true)
	}
}

// Specifies that CRC-64 is disabled during object upload.
func OssWithDisableUploadCRC64Check() OssConfigFunc {
	return func(c *oss.Config) {
		c.WithDisableUploadCRC64Check(true)
	}
}

// Specifies that CRC-64 is disabled during object download.
func OssWithDisableDownloadCRC64Check() OssConfigFunc {
	return func(c *oss.Config) {
		c.WithDisableDownloadCRC64Check(true)
	}
}

// Specifies that additional headers to be signed. It's valid in V4 signature.
func OssWithAdditionalHeaders(headers []string) OssConfigFunc {
	return func(c *oss.Config) {
		c.WithAdditionalHeaders(headers)
	}
}

// Specifies user identifier appended to the User-Agent header.
func OssWithUserAgent(ua string) OssConfigFunc {
	return func(c *oss.Config) {
		c.WithUserAgent(ua)
	}
}
