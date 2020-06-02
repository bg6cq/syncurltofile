/* james@ustc.edu.cn 2020.05.30

syncurltofile [ -h ] [ -d ] [ -t ] [ -i ] [ -c ] [ -m .md5 ] remoteURL localFile

下载URL到本地文件：
-t 禁止使用HEAD请求对比文件最后修改时间和文件大小，有变化再下载
-i 仅仅下载服务器比本地更新的文件
-c 校验md5
-m md5校验文件的扩展名，默认是 .md5

退出代码：

0 正常更新
1 无更新
2 MD5校验错

例子：

go run syncurltofile.go -c https://www.internic.net/domain/root.zone zoot.zone

*/

package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"time"
	"strings"
)

var (
	h             bool
	debug         bool
	headReq       bool
	newerFile     bool
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
	fmt.Printf("\rDownloading... %d complete", wc.Total)
}

func usage() {
	fmt.Printf("Usage:\n")
	fmt.Printf("syncurltofile [ -h ] [ -d ] [ -t ] [ -i ] [ -c ] [ -m .md5 ] remoteURL localFile\n")
	fmt.Printf("   -h      help\n")
	fmt.Printf("   -d      enable debug\n")
	fmt.Printf("   -t      disable HEAD request for check file size and modify time\n")
	fmt.Printf("   -t      only download newer file from remote\n");
	fmt.Printf("   -c      check md5 sig\n")
	fmt.Printf("   -m .md5 md5 file suffix, default is .md5\n")
	os.Exit(5)
}

func init() {
	flag.BoolVar(&h, "h", false, "help")
	flag.BoolVar(&debug, "d", false, "debug")
	flag.BoolVar(&headReq, "t", false, "disable HEAD request for check file size and time")
	flag.BoolVar(&newerFile, "i", false, "only download newer file from remote")
	flag.BoolVar(&checkMD5, "c", false, "enable md5 check")
	flag.StringVar(&md5Suffix, "m", ".md5", "md5 file suffix")
	md5Ctx = md5.New()
}

func main() {
	var remoteFileSize, remoteFileTime int64
        var err error
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
		fmt.Printf("DEBUG: HEAD REQ: %v\n", headReq)
		fmt.Printf("DEBUG: NEWER FILE: %v\n", newerFile)
		fmt.Printf("DEBUG: checkMD5: %v\n", checkMD5)
		fmt.Printf("DEBUG: md5Suffix: %s\n", md5Suffix)
	}

	if headReq {
		localFileSize, localFileTime, _ := getLocalFileSizeTime(localFileName)
		if debug {
			fmt.Printf("DEBUG: localFileSize: %d\n", localFileSize)
			fmt.Printf("DEBUG: localFileTime: %d (%s)\n", localFileTime, time.Unix(localFileTime, 0).Format("2006-01-02 15:04:05 MST"))
		}
		remoteFileSize, remoteFileTime, err = getURLSizeTime(remoteURL)
		if err != nil {
			panic(err)
		}
		if debug {
			fmt.Printf("DEBUG: remoteFileSize: %d\n", remoteFileSize)
			fmt.Printf("DEBUG: remoteFileTime: %d (%s)\n", remoteFileTime, time.Unix(remoteFileTime, 0).Format("2006-01-02 15:04:05 MST"))
		}
		if localFileSize == remoteFileSize && localFileTime == remoteFileTime {
			fmt.Printf("file size and time is same, nothing to do\n")
			os.Exit(1)
		}
		if newerFile && localFileTime > remoteFileTime {
			fmt.Printf("local file newer than remote, nothing to do\n")
			os.Exit(1)
		}
	}
	fmt.Println("Download " + remoteURL + " to " + localFileName + ".sync.tmp" + " Started")

	err = DownloadFile(remoteURL, localFileName + ".sync.tmp")
	if err != nil {
		panic(err)
	}
	fmt.Println("Download Finished")
	if checkMD5 {
		if debug {
			fmt.Println("DEBUG: Checking md5 checksum")
		}
		cipherStr := md5Ctx.Sum(nil)
		filemd5str := hex.EncodeToString(cipherStr)
		fmt.Println("Download " + remoteURL + md5Suffix + " to " + localFileName + md5Suffix + ".sync.tmp" + " Started")

		err := DownloadFile(remoteURL + md5Suffix, localFileName + md5Suffix + ".sync.tmp")
		if err != nil {
			panic(err)
		}
		fmt.Println("Download Finished")
		if checkMD5checksum(filemd5str, localFileName + md5Suffix + ".sync.tmp") {
			if debug {
				fmt.Println("DEBUG: MD5 checksum OK")
				fmt.Println("Rename " + localFileName + ".sync.tmp to " + localFileName)
				fmt.Println("Rename " + localFileName + md5Suffix + ".sync.tmp to " + localFileName + md5Suffix)
			}
			err = os.Rename(localFileName + ".sync.tmp", localFileName)
			if err != nil {
				panic(err)
			}
			err = os.Rename(localFileName + md5Suffix + ".sync.tmp", localFileName + md5Suffix)
			if err != nil {
				panic(err)
			}
			os.Exit(0)
		} else {
			fmt.Println("ERROR: MD5 checksum failed")
			os.Exit(2)
		}

	} else {
		if headReq {
			fi, err := os.Stat(localFileName + ".sync.tmp")
			if err != nil {
				panic(err)
			}
			if remoteFileSize != fi.Size() {
				fmt.Printf("ERROR: ContentLength: %d, but downloaded file lenght is: %d\n", remoteFileSize, fi.Size)
				os.Exit(2)
			}
		}
		if debug {
			fmt.Println("DEBUG: Rename " + localFileName + ".sync.tmp to " + localFileName)
		}
		err = os.Rename(localFileName + ".sync.tmp", localFileName)
		if err != nil {
			panic(err)
		}
		os.Exit(0)
	}
}

func checkMD5checksum(md5sig string, filepath string) bool {
	var line string
	file, err := os.Open(filepath)
	defer file.Close()

	if err != nil {
		return false
	}

	reader := bufio.NewReader(file)

	for {
		line, err = reader.ReadString('\n')

		line = strings.Replace(line, "\n", "", -1)
		for _, s := range strings.Split(line, " ") {
			if debug {
				fmt.Println("DEBUG: md5checksum:" + md5sig + ":findchecksum:" + s + ":")
			}
			if s == md5sig {
				return true
			}
		}

		if err != nil {
			break
		}
	}
	return false
}

func getLocalFileSizeTime(filepath string) (int64, int64, error) {
	fi, err := os.Stat(filepath)
	if err != nil {
		return 0, 0, err
	}
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
		fmt.Printf("ERROR: bad http status: %s", resp.Status)
		os.Exit(2)
	}

	if debug {
		t, err := http.ParseTime(resp.Header.Get("Last-Modified"))
        	if err == nil {
                	fmt.Printf("DEBUG: ContentLength: %d Last-Modified: %d (%s)\n", resp.ContentLength, t.Unix(), resp.Header.Get("Last-Modified"));
        	} else {
                	fmt.Printf("DEBUG: ContentLength: %d\n", resp.ContentLength);
		}
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
