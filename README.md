# 短域名服务

一个简单的短域名生成跳转服务


### 安装教程

localhost: `$ make run`

### 依赖

- Golang 1.13+ [安装手册](https://golang.org/dl/)
- Docker 18.x+ [安装](https://docs.docker.com/install/)
- Mongo/Redis (主要用于存储短链信息)

## 快速开始


1. 正常启动
```
go run ./cmd/main.go -http-addr :8080
```
2. make 启动

```
$ make run
```

