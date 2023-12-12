#如何多线程分片下载大文件
1. 使用[HEAD](https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Methods/HEAD "")获取下载链接的请求头
   1. 根据[Accept-Ranges](https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Headers/Accept-Ranges "")标识判断文件是否支持分片下载
   2. 读取其[Content-Length](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Length "")标头的值获取文件的大小，而无需实际下载文件，以此可以节约带宽资源
2. 确定分片的规格，构造下载文件结构体，可通过[Range](https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Headers/Range "")告知服务器返回文件的哪一部分
3. 使用goroutine并发下载分片文件
4. 合并分片文件
5. SHA-256校验[可选]