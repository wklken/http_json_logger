http_json_logger
===========================

一个日志上报收集服务, 可以收集从浏览器/js/android/ios等通过http上报的日志, 落地为文本文件, 用作后续日志统计/分析/数据挖掘等

A http json logger, implemented by golang.

Just dump http json request into data file, than you can dosomething with the data file.

You can use the server to collect log data reported by Browser/Js/Android/ios,

e.g. tranfer by logstash to elasticsearch (the log file will be saved forever, the records in elasticsearch is temporary for 1 week?)

    http json request => http_json_logger => xxx_data.log => logstash shippter => [redis]  => logstash indexer => elasticsearch


the logger module is modified from code:https://github.com/astaxie/beego/tree/master/logs

========================

### example

make a request

    # curl
    curl -X POST -H "Content-Type: application/json"  -d '{"name":"ken", "id": 101}' http://127.0.0.1:6500/collect/ios/test

    # python requests

    import requests
    url = "http://127.0.0.1:6500/collect/ios/testa"
    data = {"name":"ken", "id": 101}
    resp = requests.post(url=url, json=data)
    print resp.status_code
    print resp.text


then you got record at `./data/ios/test.log`

    # `platform` and `doctype` add into json body
    # `ts` is the timestamp when the request happened
    {"doctype":"test","id":101,"name":"ken","platform":"ios","ts":1430749343}

each line in the file is json

    cat ./data/ios/test.log
    {"doctype":"test","id":101,"name":"ken","platform":"ios","ts":1430749343}
    {"doctype":"test","id":101,"name":"ken","platform":"ios","ts":1430750438}
    {"doctype":"test","id":101,"name":"ken","platform":"ios","ts":1430750524}
    {"doctype":"test","id":101,"name":"ken","platform":"ios","ts":1430750654}



========================

### install

    # into your golang worksapce
    cd $GOPATH/src

    git clone https://github.com/wklken/http_json_logger.git
    cd http_json_logger

    go get github.com/astaxie/beego/config
    go get github.com/gorilla/mux

    # then do build
    go build -o runserver

### config file

    bind="127.0.0.1:6500"        # server bind ip:port
    log_data_path="./data"       # log data path

    [platforms]                  # valid platforms
    list = ios;android;web;wap

    [platform.ios]               # specific platform valid doctype
    list = test

### run

    ./runserver config.ini
    Register Platforms: [ios android web wap]
    Register DocTypes: ios [test]
    Register DocTypes: android []
    Register DocTypes: web []
    Register DocTypes: wap [test demo]
    Bind to host: 127.0.0.1:6500

### url rule

    url: http://127.0.0.1:6500/collect/{platform}/{doctype}
    method: POST
    header: application/json
    body:   any json format

    platform: must be one of item in the valid platforms white list. (config.ini, section [platforms])
    doctype: must be one of item in the valid platform:doctype white list. (config.ini, section [platform:{platform}])

    response:

    204: log success

    404: url not found or wrong method(POST)
    400: invalid platform or invalid doctype
    500: Internal Server Error

### Donation

You can Buy me a coffee:)

[donation](http://www.wklken.me/pages/donation.html)

========================

wklken

2015-05-04



