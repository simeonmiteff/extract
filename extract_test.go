package extract_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/arduino/go-paths-helper"
	"github.com/codeclysm/extract/v5"
	"github.com/stretchr/testify/require"
)

type Files map[string]string

var shift = func(path string) string {
	parts := strings.Split(path, string(filepath.Separator))
	parts = parts[1:]
	return strings.Join(parts, string(filepath.Separator))
}

var subfolder = func(path string) string {
	if strings.Contains(path, "archive/folder") {
		return path
	}
	return ""
}

var ExtractCases = []struct {
	Name    string
	Archive string
	Renamer extract.Renamer
	Files   Files
}{
	{"standard bz2", "testdata/archive.tar.bz2", nil, Files{
		"":                          "dir",
		"/archive":                  "dir",
		"/archive/folder":           "dir",
		"/archive/folderlink":       "link",
		"/archive/folder/file1.txt": "folder/File1",
		"/archive/file1.txt":        "File1",
		"/archive/file2.txt":        "File2",
		"/archive/link.txt":         "File1",
	}},
	{"shift bz2", "testdata/archive.tar.bz2", shift, Files{
		"":                  "dir",
		"/folder":           "dir",
		"/folderlink":       "link",
		"/folder/file1.txt": "folder/File1",
		"/file1.txt":        "File1",
		"/file2.txt":        "File2",
		"/link.txt":         "File1",
	}},
	{"subfolder bz2", "testdata/archive.tar.bz2", subfolder, Files{
		"":                          "dir",
		"/archive":                  "dir",
		"/archive/folder":           "dir",
		"/archive/folder/file1.txt": "folder/File1",
		"/archive/folderlink":       "link",
	}},
	{"not tarred bz2", "testdata/singlefile.bz2", nil, Files{
		"": "singlefile",
	}},

	{"standard gz", "testdata/archive.tar.gz", nil, Files{
		"":                          "dir",
		"/archive":                  "dir",
		"/archive/folder":           "dir",
		"/archive/folderlink":       "link",
		"/archive/folder/file1.txt": "folder/File1",
		"/archive/file1.txt":        "File1",
		"/archive/file2.txt":        "File2",
		"/archive/link.txt":         "File1",
	}},
	{"shift gz", "testdata/archive.tar.gz", shift, Files{
		"":                  "dir",
		"/folder":           "dir",
		"/folderlink":       "link",
		"/folder/file1.txt": "folder/File1",
		"/file1.txt":        "File1",
		"/file2.txt":        "File2",
		"/link.txt":         "File1",
	}},
	{"subfolder gz", "testdata/archive.tar.gz", subfolder, Files{
		"":                          "dir",
		"/archive":                  "dir",
		"/archive/folder":           "dir",
		"/archive/folder/file1.txt": "folder/File1",
		"/archive/folderlink":       "link",
	}},
	{"not tarred gz", "testdata/singlefile.gz", nil, Files{
		"": "singlefile",
	}},
	// Note that the zip format doesn't support hard links
	{"standard zip", "testdata/archive.zip", nil, Files{
		"":                          "dir",
		"/archive":                  "dir",
		"/archive/folder":           "dir",
		"/archive/folderlink":       "link",
		"/archive/folder/file1.txt": "folder/File1",
		"/archive/file1.txt":        "File1",
		"/archive/file2.txt":        "File2",
		"/archive/link.txt":         "File1",
	}},
	{"shift zip", "testdata/archive.zip", shift, Files{
		"":                  "dir",
		"/folder":           "dir",
		"/folderlink":       "link",
		"/folder/file1.txt": "folder/File1",
		"/file1.txt":        "File1",
		"/file2.txt":        "File2",
		"/link.txt":         "File1",
	}},
	{"subfolder zip", "testdata/archive.zip", subfolder, Files{
		"":                          "dir",
		"/archive":                  "dir",
		"/archive/folder":           "dir",
		"/archive/folder/file1.txt": "folder/File1",
		"/archive/folderlink":       "link",
	}},

	{"standard inferred", "testdata/archive.mistery", nil, Files{
		"":                          "dir",
		"/archive":                  "dir",
		"/archive/folder":           "dir",
		"/archive/folderlink":       "link",
		"/archive/folder/file1.txt": "folder/File1",
		"/archive/file1.txt":        "File1",
		"/archive/file2.txt":        "File2",
		"/archive/link.txt":         "File1",
	}},
	{"shift inferred", "testdata/archive.mistery", shift, Files{
		"":                  "dir",
		"/folder":           "dir",
		"/folderlink":       "link",
		"/folder/file1.txt": "folder/File1",
		"/file1.txt":        "File1",
		"/file2.txt":        "File2",
		"/link.txt":         "File1",
	}},
	{"subfolder inferred", "testdata/archive.mistery", subfolder, Files{
		"":                          "dir",
		"/archive":                  "dir",
		"/archive/folder":           "dir",
		"/archive/folder/file1.txt": "folder/File1",
		"/archive/folderlink":       "link",
	}},

	{"standard zip with backslashes", "testdata/archive-with-backslashes.zip", nil, Files{
		"":                           "dir",
		"/AZ3166":                    "dir",
		"/AZ3166/libraries":          "dir",
		"/AZ3166/libraries/AzureIoT": "dir",
		"/AZ3166/libraries/AzureIoT/keywords.txt": "Azure",
		"/AZ3166/cores":                                   "dir",
		"/AZ3166/cores/arduino":                           "dir",
		"/AZ3166/cores/arduino/azure-iot-sdk-c":           "dir",
		"/AZ3166/cores/arduino/azure-iot-sdk-c/umqtt":     "dir",
		"/AZ3166/cores/arduino/azure-iot-sdk-c/umqtt/src": "dir",
	}},
	{"shift zip with backslashes", "testdata/archive-with-backslashes.zip", shift, Files{
		"":                                     "dir",
		"/libraries":                           "dir",
		"/libraries/AzureIoT":                  "dir",
		"/libraries/AzureIoT/keywords.txt":     "Azure",
		"/cores":                               "dir",
		"/cores/arduino":                       "dir",
		"/cores/arduino/azure-iot-sdk-c":       "dir",
		"/cores/arduino/azure-iot-sdk-c/umqtt": "dir",
		"/cores/arduino/azure-iot-sdk-c/umqtt/src": "dir",
	}},
}

func TestArchiveFailure(t *testing.T) {
	err := extract.Archive(context.Background(), strings.NewReader("not an archive"), "", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Not a supported archive")
}

func TestExtract(t *testing.T) {
	for _, test := range ExtractCases {
		dir, _ := os.MkdirTemp("", "")
		dir = filepath.Join(dir, "test")
		data, err := os.ReadFile(test.Archive)
		if err != nil {
			t.Fatal(err)
		}
		buffer := bytes.NewBuffer(data)

		switch filepath.Ext(test.Archive) {
		case ".bz2":
			err = extract.Bz2(context.Background(), buffer, dir, test.Renamer)
		case ".gz":
			err = extract.Gz(context.Background(), buffer, dir, test.Renamer)
		case ".zip":
			err = extract.Zip(context.Background(), buffer, dir, test.Renamer)
		case ".mistery":
			err = extract.Archive(context.Background(), buffer, dir, test.Renamer)
		default:
			t.Fatal("unknown error")
		}

		if err != nil {
			t.Fatal(test.Name, ": Should not fail: "+err.Error())
		}

		testWalk(t, dir, test.Files)

		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestExtractIdempotency(t *testing.T) {
	for _, test := range ExtractCases {
		t.Run(test.Name, func(t *testing.T) {
			dir, _ := os.MkdirTemp("", "")
			dir = filepath.Join(dir, "test")
			data, err := os.ReadFile(test.Archive)
			if err != nil {
				t.Fatal(err)
			}

			var extractFn func(context.Context, io.Reader, string, extract.Renamer) error
			switch filepath.Ext(test.Archive) {
			case ".bz2":
				extractFn = extract.Bz2
			case ".gz":
				extractFn = extract.Gz
			case ".zip":
				extractFn = extract.Zip
			case ".mistery":
				extractFn = extract.Archive
			default:
				t.Fatal("unknown error")
			}

			buffer := bytes.NewBuffer(data)
			if err = extractFn(context.Background(), buffer, dir, test.Renamer); err != nil {
				t.Fatal(test.Name, ": Should not fail first extraction: "+err.Error())
			}

			buffer = bytes.NewBuffer(data)
			if err = extractFn(context.Background(), buffer, dir, test.Renamer); err != nil {
				t.Fatal(test.Name, ": Should not fail second extraction: "+err.Error())
			}

			testWalk(t, dir, test.Files)

			err = os.RemoveAll(dir)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func BenchmarkArchive(b *testing.B) {
	dir, _ := os.MkdirTemp("", "")
	data, _ := os.ReadFile("testdata/archive.tar.bz2")

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		buffer := bytes.NewBuffer(data)
		err := extract.Archive(context.Background(), buffer, filepath.Join(dir, strconv.Itoa(i)), nil)
		if err != nil {
			b.Error(err)
		}
	}

	b.StopTimer()

	err := os.RemoveAll(dir)
	if err != nil {
		b.Error(err)
	}
}

func BenchmarkTarBz2(b *testing.B) {
	dir, _ := os.MkdirTemp("", "")
	data, _ := os.ReadFile("testdata/archive.tar.bz2")

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		buffer := bytes.NewBuffer(data)
		err := extract.Bz2(context.Background(), buffer, filepath.Join(dir, strconv.Itoa(i)), nil)
		if err != nil {
			b.Error(err)
		}
	}

	b.StopTimer()

	err := os.RemoveAll(dir)
	if err != nil {
		b.Error(err)
	}
}

func BenchmarkTarGz(b *testing.B) {
	dir, _ := os.MkdirTemp("", "")
	data, _ := os.ReadFile("testdata/archive.tar.gz")

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		buffer := bytes.NewBuffer(data)
		err := extract.Gz(context.Background(), buffer, filepath.Join(dir, strconv.Itoa(i)), nil)
		if err != nil {
			b.Error(err)
		}
	}

	b.StopTimer()

	err := os.RemoveAll(dir)
	if err != nil {
		b.Error(err)
	}
}

func BenchmarkZip(b *testing.B) {
	dir, _ := os.MkdirTemp("", "")
	data, _ := os.ReadFile("testdata/archive.zip")

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		buffer := bytes.NewBuffer(data)
		err := extract.Zip(context.Background(), buffer, filepath.Join(dir, strconv.Itoa(i)), nil)
		if err != nil {
			b.Error(err)
		}
	}

	b.StopTimer()

	err := os.RemoveAll(dir)
	if err != nil {
		b.Error(err)
	}
}

func testWalk(t *testing.T, dir string, testFiles Files) {
	files := Files{}
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		path = strings.Replace(path, dir, "", 1)

		if info.IsDir() {
			files[path] = "dir"
		} else if info.Mode()&os.ModeSymlink != 0 {
			files[path] = "link"
		} else {
			data, err := os.ReadFile(filepath.Join(dir, path))
			require.NoError(t, err)
			files[path] = strings.TrimSpace(string(data))
		}

		return nil
	})

	for file, kind := range files {
		k, ok := testFiles[file]
		if !ok {
			t.Error(file + " should not exist")
			continue
		}

		if kind != k {
			t.Error(file + " should be " + k + ", not " + kind)
			continue
		}
	}

	for file, kind := range testFiles {
		k, ok := files[file]
		if !ok {
			t.Error(file + " should exist")
			continue
		}

		if kind != k {
			t.Error(file + " should be " + kind + ", not " + k)
			continue
		}
	}
}

func TestTarGzMemoryConsumption(t *testing.T) {
	archive := paths.New("testdata/big.tar.gz")
	download(t, "http://downloads.arduino.cc/gcc-arm-none-eabi-4.8.3-2014q1-windows.tar.gz", archive)

	tmpDir, err := paths.MkTempDir("", "")
	require.NoError(t, err)
	defer tmpDir.RemoveAll()

	f, err := archive.Open()
	require.NoError(t, err)

	var m, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)

	err = extract.Archive(context.Background(), f, tmpDir.String(), nil)
	require.NoError(t, err)

	runtime.ReadMemStats(&m2)
	heapUsed := m2.HeapInuse - m.HeapInuse
	if m2.HeapInuse < m.HeapInuse {
		heapUsed = 0
	}
	fmt.Println("Heap memory used during the test:", heapUsed)
	require.True(t, heapUsed < 5000000, "heap consumption should be less than 5M but is %d", heapUsed)
}

func TestZipMemoryConsumption(t *testing.T) {
	archive := paths.New("testdata/big.zip")
	download(t, "http://downloads.arduino.cc/tools/gcc-arm-none-eabi-7-2017-q4-major-win32-arduino1.zip", archive)

	tmpDir, err := paths.MkTempDir("", "")
	require.NoError(t, err)
	defer tmpDir.RemoveAll()

	f, err := archive.Open()
	require.NoError(t, err)

	var m, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)

	err = extract.Archive(context.Background(), f, tmpDir.String(), nil)
	require.NoError(t, err)

	runtime.ReadMemStats(&m2)
	heapUsed := m2.HeapInuse - m.HeapInuse
	if m2.HeapInuse < m.HeapInuse {
		heapUsed = 0
	}
	fmt.Println("Heap memory used during the test:", heapUsed)
	require.True(t, heapUsed < 10000000, "heap consumption should be less than 10M but is %d", heapUsed)
}

func download(t require.TestingT, url string, file *paths.Path) {
	if file.Exist() {
		return
	}

	fmt.Printf("Downloading %s in %s\n", url, file)
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	out, err := file.Create()
	require.NoError(t, err)

	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		file.Remove()
	}
	require.NoError(t, err)
}
