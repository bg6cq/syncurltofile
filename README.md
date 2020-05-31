
下载URL到本地文件：
```
syncurltofile [ -h ] [ -d ] [ -t ] [ -c ] [ -m .md5 ] remoteURL localFile

-t 禁止使用HEAD请求对比文件最后修改时间和文件大小，有变化再下载
-c 校验md5
-m md5校验文件的扩展名，默认是 .md5

退出代码：

0 正常更新
1 无更新
2 MD5校验错
```

例子：

```
go run syncurltofile.go -c https://www.internic.net/domain/root.zone root.zone
```

