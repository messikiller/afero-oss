package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) *http.Response

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	res := f(request)
	res.Request = request
	return res, nil
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func newTestManager(roundTrip roundTripFunc) *OssObjectManager {
	config := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider("ak", "sk")).
		WithRegion("cn-hangzhou").
		WithEndpoint("http://oss.test").
		WithUsePathStyle(true).
		WithDisableDownloadCRC64Check(true).
		WithDisableUploadCRC64Check(true).
		WithHttpClient(&http.Client{Transport: roundTrip})
	return &OssObjectManager{Client: oss.NewClient(config)}
}

func TestGetObjectReturns404WithoutPanic(t *testing.T) {
	manager := newTestManager(func(_ *http.Request) *http.Response {
		res := response(http.StatusNotFound, `<Error><Code>NoSuchKey</Code><Message>missing</Message></Error>`)
		res.Header.Set("Content-Type", "application/xml")
		return res
	})
	reader, cleanup, err := manager.GetObject(context.Background(), "bucket", "missing")
	require.Nil(t, reader)
	require.Nil(t, cleanup)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestGetObjectPartUsesInclusiveRange(t *testing.T) {
	var gotRange string
	manager := newTestManager(func(r *http.Request) *http.Response {
		gotRange = r.Header.Get("Range")
		res := response(http.StatusPartialContent, "bc")
		res.ContentLength = 2
		res.Header.Set("Content-Length", "2")
		return res
	})
	reader, cleanup, err := manager.GetObjectPart(context.Background(), "bucket", "file", 1, 2)
	require.NoError(t, err)
	defer cleanup()
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, "bytes=1-2", gotRange)
	require.Equal(t, "bc", string(data))
}

func TestListObjectsIncludesCommonPrefixesAndAllEntriesForNonPositiveCount(t *testing.T) {
	manager := newTestManager(func(_ *http.Request) *http.Response {
		res := response(http.StatusOK, `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult>
  <Name>bucket</Name>
  <Prefix>dir/</Prefix>
  <Delimiter>/</Delimiter>
  <MaxKeys>1000</MaxKeys>
  <KeyCount>3</KeyCount>
  <IsTruncated>false</IsTruncated>
  <Contents>
    <Key>dir/</Key><LastModified>2026-01-01T00:00:00.000Z</LastModified><Size>0</Size>
  </Contents>
  <Contents>
    <Key>dir/a.txt</Key><LastModified>2026-01-01T00:00:00.000Z</LastModified><Size>1</Size>
  </Contents>
  <Contents>
    <Key>dir/b.txt</Key><LastModified>2026-01-01T00:00:00.000Z</LastModified><Size>1</Size>
  </Contents>
  <CommonPrefixes><Prefix>dir/sub/</Prefix></CommonPrefixes>
</ListBucketResult>`)
		res.Header.Set("Content-Type", "application/xml")
		return res
	})
	entries, err := manager.ListObjects(context.Background(), "bucket", "dir/", 0)
	require.NoError(t, err)
	require.Len(t, entries, 3)
	require.Equal(t, []string{"a.txt", "b.txt", "sub"}, []string{entries[0].Name(), entries[1].Name(), entries[2].Name()})
	require.False(t, entries[0].IsDir())
	require.True(t, entries[2].IsDir())
}

func TestGetObjectPartRejectsInvalidRange(t *testing.T) {
	manager := &OssObjectManager{}
	_, _, err := manager.GetObjectPart(context.Background(), "bucket", "file", -1, 1)
	require.Error(t, err)
	_, _, err = manager.GetObjectPart(context.Background(), "bucket", "file", 2, 1)
	require.Error(t, err)
}

func TestObjectMetadataAndMutations(t *testing.T) {
	var uploaded string
	manager := newTestManager(func(request *http.Request) *http.Response {
		switch request.Method {
		case http.MethodHead:
			res := response(http.StatusOK, "")
			res.ContentLength = 4
			res.Header.Set("Content-Length", "4")
			res.Header.Set("Last-Modified", "Thu, 01 Jan 2026 00:00:00 GMT")
			res.Header.Set("ETag", `"etag"`)
			return res
		case http.MethodPut:
			if request.Header.Get("x-oss-copy-source") != "" {
				res := response(http.StatusOK, `<CopyObjectResult><LastModified>2026-01-01T00:00:00.000Z</LastModified><ETag>etag</ETag></CopyObjectResult>`)
				res.Header.Set("Content-Type", "application/xml")
				return res
			}
			body, _ := io.ReadAll(request.Body)
			uploaded = string(body)
			return response(http.StatusOK, "")
		case http.MethodDelete:
			return response(http.StatusNoContent, "")
		default:
			return response(http.StatusMethodNotAllowed, "")
		}
	})

	info, err := manager.GetObjectMeta(context.Background(), "bucket", "dir/file")
	require.NoError(t, err)
	require.Equal(t, "file", info.Name())
	require.Equal(t, int64(4), info.Size())

	exists, err := manager.IsObjectExist(context.Background(), "bucket", "dir/file")
	require.NoError(t, err)
	require.True(t, exists)

	ok, err := manager.PutObject(context.Background(), "bucket", "dir/file", strings.NewReader("data"))
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "data", uploaded)
	require.NoError(t, manager.CopyObject(context.Background(), "bucket", "dir/file", "dir/copy"))
	require.NoError(t, manager.DeleteObject(context.Background(), "bucket", "dir/file"))
}

func TestListAllObjectsKeepsFullKeys(t *testing.T) {
	manager := newTestManager(func(_ *http.Request) *http.Response {
		res := response(http.StatusOK, `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult>
  <Name>bucket</Name><Prefix>dir/</Prefix><MaxKeys>1000</MaxKeys><KeyCount>2</KeyCount><IsTruncated>false</IsTruncated>
  <Contents><Key>dir/</Key><LastModified>2026-01-01T00:00:00.000Z</LastModified><Size>0</Size></Contents>
  <Contents><Key>dir/file</Key><LastModified>2026-01-01T00:00:00.000Z</LastModified><Size>4</Size></Contents>
</ListBucketResult>`)
		res.Header.Set("Content-Type", "application/xml")
		return res
	})

	entries, err := manager.ListAllObjects(context.Background(), "bucket", "dir/")
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, "dir/", entries[0].Name())
	require.True(t, entries[0].IsDir())
	require.Equal(t, "dir/file", entries[1].Name())
}
