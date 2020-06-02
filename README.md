
下载URL到本地文件：
```
syncurltofile [ -h ] [ -d ] [ -t ] [ -c ] [ -m .md5 ] remoteURL localFile

-d 打印调试信息
-t 禁止使用HEAD请求对比文件最后修改时间和文件大小，有变化再下载
-i 仅仅下载服务器比本地更新的文件
-c 校验md5
-m md5校验文件的扩展名，默认是 .md5

退出代码：

0 正常更新
1 无更新
2 MD5校验错
```

编译：
```
go build syncurltofile.go
```

例子：

```
./syncurltofile -c -d https://www.internic.net/domain/root.zone root.zone
```
