/* james@ustc.edu.cn 2020.05.30

syncurltofile [ -h ] [ -d ] [ -t ] [ -c ] [ -m .md5 ] remoteURL localFile

下载URL到本地文件：
-t 禁止使用HEAD请求对比文件最后修改时间和文件大小，有变化再下载
-c 校验md5
-m md5校验文件的扩展名，默认是 .md5

退出代码：

0 正常更新
1 无更新
2 MD5校验错

例子：

go run syncurltofile.go -t -c https://www.internic.net/domain/root.zone zoot.zone

*/

package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/dustin/go-humanize"
	"hash"
	"io"
	"net/http"
	"os"
	"strings"
)

var (
	h             bool
	debug         bool
	headReq       bool
	checkMD5      bool
	remoteURL     string
	localFileName string
	md5Suffix     string
	md5Ctx        hash.Hash
)

type WriteCounter struct {
	Total uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	if checkMD5 {
		md5Ctx.Write(p)
	}
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc WriteCounter) PrintProgress() {
	fmt.Printf("\r%s", strings.Repeat(" ", 50))
	fmt.Printf("\rDownloading... %s complete", humanize.Bytes(wc.Total))
}

func usage() {
	fmt.Printf("Usage:\n")
	fmt.Printf("syncurltofile [ -h ] [ -d ] [ -t ] [ -c ] [ -m .md5 ] remoteURL localFile\n")
	fmt.Printf("   -h      help\n")
	fmt.Printf("   -d      enable debug\n")
	fmt.Printf("   -t      disable HEAD request to check file size and modify time\n")
	fmt.Printf("   -c      check md5 sig\n")
	fmt.Printf("   -m .md5 md5 file suffix, default is .md5\n")
	os.Exit(5)
}

func init() {
	flag.BoolVar(&h, "h", false, "help")
	flag.BoolVar(&debug, "d", false, "debug")
	flag.BoolVar(&headReq, "t", false, "disable HEAD request to check file size and time")
	flag.BoolVar(&checkMD5, "c", false, "enable md5 check")
	flag.StringVar(&md5Suffix, "m", ".md5", "md5 file suffix")
	md5Ctx = md5.New()
}

func main() {
	var remoteFileSize int64
	flag.Parse()
	if h {
		usage()
	}
	if flag.NArg() != 2 {
		usage()
	}

	headReq = !headReq
	remoteURL := flag.Arg(0)
	localFileName := flag.Arg(1)
	fmt.Printf("remoteURL: %s\n", remoteURL)
	fmt.Printf("localFile: %s\n", localFileName)

	if debug {
		fmt.Printf("HEAD REQ: %v\n", headReq)
		fmt.Printf("checkMD5: %v\n", checkMD5)
		fmt.Printf("md5Suffix: %s\n", md5Suffix)
	}

	if headReq {
		localFileSize, localFileTime, _ := getLocalFileSizeTime(localFileName)
		if debug {
			fmt.Printf("localFileSize: %d\n", localFileSize)
			fmt.Printf("localFileTime: %d\n", localFileTime)
		}
		remoteFileSize, remoteFileTime, err := getURLSizeTime(remoteURL)
		if err != nil {
			panic(err)
		}
		if debug {
			fmt.Printf("remoteFileSize: %d\n", remoteFileSize)
			fmt.Printf("remoteFileTime: %d\n", remoteFileTime)
		}
		if localFileSize == remoteFileSize && localFileTime == remoteFileTime {
			fmt.Printf("file size and time is same, nothing to do\n")
			os.Exit(1)
		}
	}
	fmt.Println("Download " + remoteURL + " to " + localFileName + ".sync.tmp" + " Started")

	err := DownloadFile(remoteURL, localFileName+".sync.tmp")
	if err != nil {
		panic(err)
	}
	fmt.Println("Download Finished")
	if checkMD5 {
		if debug {
			fmt.Println("checking md5 sig")
		}
		cipherStr := md5Ctx.Sum(nil)
		filemd5str := hex.EncodeToString(cipherStr)
		fmt.Printf("downloaded file md5: %s\n", filemd5str)
		fmt.Println("Download " + remoteURL + md5Suffix + " to " + localFileName + md5Suffix + ".sync.tmp" + " Started")

		err := DownloadFile(remoteURL+md5Suffix, localFileName+md5Suffix+".sync.tmp")
		if err != nil {
			panic(err)
		}
		fmt.Println("Download Finished")
		if checkMD5Sig(filemd5str, localFileName+md5Suffix+".sync.tmp") == nil {
			if debug {
				fmt.Println("MD5 sig check OK")
				fmt.Println("rename " + localFileName + ".sync.tmp to " + localFileName)
				fmt.Println("rename " + localFileName + md5Suffix + ".sync.tmp to " + localFileName + md5Suffix)
			}
			err = os.Rename(localFileName+".sync.tmp", localFileName)
			if err != nil {
				panic(err)
			}
			err = os.Rename(localFileName+md5Suffix+".sync.tmp", localFileName+md5Suffix)
			if err != nil {
				panic(err)
			}
			os.Exit(0)
		} else {
			fmt.Println("ERROR: MD5 sig check failed")
			os.Exit(-1)
		}

	} else {
		if headReq {
			fi, err := os.Stat(localFileName + ".sync.tmp")
			if err != nil {
				panic(err)
			}
			if remoteFileSize != fi.Size() {
				fmt.Println("download size error: ContentLength: %d, but downloaded file lenght is: %d", remoteFileSize, fi.Size)
				os.Exit(2)
			}
		}
		if debug {
			fmt.Println("rename " + localFileName + ".sync.tmp to " + localFileName)
		}
		err = os.Rename(localFileName+".sync.tmp", localFileName)
		if err != nil {
			panic(err)
		}
		os.Exit(0)
	}
}

func checkMD5Sig(md5sig string, filepath string) error {
	file, err := os.Open(filepath)
	defer file.Close()

	if err != nil {
		return err
	}

	reader := bufio.NewReader(file)

	var line string
	for {
		line, err = reader.ReadString('\n')

		line = strings.Replace(line, "\n", "", -1)
		for _, s := range strings.Split(line, " ") {
			if debug {
				fmt.Println("md5sig:" + md5sig + ":findsig:" + s + ":")
			}
			if s == md5sig {
				return nil
			}
		}

		if err != nil {
			break
		}
	}

	return fmt.Errorf("md5 sig not match")
}

func getLocalFileSizeTime(filepath string) (int64, int64, error) {
	fi, err := os.Stat(filepath)
	if err != nil {
		return 0, 0, err
	}
	// get the size
	return fi.Size(), fi.ModTime().Unix(), nil
}

func getURLSizeTime(url string) (int64, int64, error) {
	resp, err := http.Head(url)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("HEAD request return bad status: %s", resp.Status)
	}
	t, err := http.ParseTime(resp.Header.Get("Last-Modified"))
	if err != nil {
		return 0, 0, err
	}
	return resp.ContentLength, t.Unix(), nil
}

func DownloadFile(url string, filepath string) error {

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	counter := &WriteCounter{}
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	if err != nil {
		return err
	}

	fmt.Println()

	t, err := http.ParseTime(resp.Header.Get("Last-Modified"))
	if err != nil {
		return err
	}
	fmt.Println("Changed the file time information")
	err = os.Chtimes(filepath, t, t)

	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
