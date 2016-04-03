package main

import "strings"

// Package represents package info, needed for this tool.
type Package struct {
	Name string
	Path string
}

// NewPackage creates new Package.
func NewPackage(name, path string) Package {
	return Package{
		Name: name,
		Path: path,
	}
}

// IsInternal returns true if subpkg is a subpackage of
// pkg.
func IsInternal(pkg, subpkg string) bool {
	// Skip if any is stdlib
	if IsStdlib(pkg) || IsStdlib(subpkg) {
		return false
	}

	// Or it is submodule
	if strings.HasPrefix(subpkg, pkg+"/") {
		return true
	}

	// Or it is on same repo nesting level (nesting > 2)
	// FIXME: this code assumes layout "server/user/repo",
	// for non-standard layouts ("gopkg.in/music.v0") it'll
	// report false negative.
	if i := strings.Count(pkg, "/"); i > 2 {
		if strings.HasPrefix(subpkg, pkg[0:i]) {
			return true
		}
	}

	return false
}

// IsStdlib attempts to check if package belongs to stdlib.
// FIXME: there is probably more idiomatic way to do this,
// I just need to find it.
func IsStdlib(path string) bool {
	for _, p := range stdPkgs {
		if p == path {
			return true
		}
	}
	return false
}

var stdPkgs = []string{
	"archive/tar",
	"archive/zip",
	"bufio",
	"bytes",
	"compress/bzip2",
	"compress/flate",
	"compress/gzip",
	"compress/lzw",
	"compress/zlib",
	"container/heap",
	"container/list",
	"container/ring",
	"crypto",
	"crypto/aes",
	"crypto/cipher",
	"crypto/des",
	"crypto/dsa",
	"crypto/ecdsa",
	"crypto/elliptic",
	"crypto/hmac",
	"crypto/md5",
	"crypto/rand",
	"crypto/rc4",
	"crypto/rsa",
	"crypto/sha1",
	"crypto/sha256",
	"crypto/sha512",
	"crypto/subtle",
	"crypto/tls",
	"crypto/x509",
	"crypto/x509/pkix",
	"database/sql",
	"database/sql/driver",
	"debug/dwarf",
	"debug/elf",
	"debug/gosym",
	"debug/macho",
	"debug/pe",
	"debug/plan9obj",
	"encoding",
	"encoding/ascii85",
	"encoding/asn1",
	"encoding/base32",
	"encoding/base64",
	"encoding/binary",
	"encoding/csv",
	"encoding/gob",
	"encoding/hex",
	"encoding/json",
	"encoding/pem",
	"encoding/xml",
	"errors",
	"expvar",
	"flag",
	"fmt",
	"go/ast",
	"go/build",
	"go/constant",
	"go/doc",
	"go/format",
	"go/importer",
	"go/internal/gccgoimporter",
	"go/internal/gcimporter",
	"go/parser",
	"go/printer",
	"go/scanner",
	"go/token",
	"go/types",
	"hash",
	"hash/adler32",
	"hash/crc32",
	"hash/crc64",
	"hash/fnv",
	"html",
	"html/template",
	"image",
	"image/color",
	"image/color/palette",
	"image/draw",
	"image/gif",
	"image/internal/imageutil",
	"image/jpeg",
	"image/png",
	"index/suffixarray",
	"internal/golang.org/x/net/http2/hpack",
	"internal/race",
	"internal/singleflight",
	"internal/testenv",
	"internal/trace",
	"io",
	"io/ioutil",
	"log",
	"log/syslog",
	"math",
	"math/big",
	"math/cmplx",
	"math/rand",
	"mime",
	"mime/multipart",
	"mime/quotedprintable",
	"net",
	"net/http",
	"net/http/cgi",
	"net/http/cookiejar",
	"net/http/fcgi",
	"net/http/httptest",
	"net/http/httputil",
	"net/http/internal",
	"net/http/pprof",
	"net/internal/socktest",
	"net/mail",
	"net/rpc",
	"net/rpc/jsonrpc",
	"net/smtp",
	"net/textproto",
	"net/url",
	"os",
	"os/exec",
	"os/signal",
	"os/user",
	"path",
	"path/filepath",
	"reflect",
	"regexp",
	"regexp/syntax",
	"runtime",
	"runtime/cgo",
	"runtime/debug",
	"runtime/internal/atomic",
	"runtime/internal/sys",
	"runtime/pprof",
	"runtime/race",
	"runtime/trace",
	"sort",
	"strconv",
	"strings",
	"sync",
	"sync/atomic",
	"syscall",
	"testing",
	"testing/iotest",
	"testing/quick",
	"text/scanner",
	"text/tabwriter",
	"text/template",
	"text/template/parse",
	"time",
	"unicode",
	"unicode/utf16",
	"unicode/utf8",
	"unsafe",
}
