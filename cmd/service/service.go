/**
 * @Time : 19/11/2019 10:25 AM * @Author:ygqbasic@gmail.com * @File:service * @Software:VsCode */

package service

import (
	"flag"
	"fmt"
	kitendpoint "github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log/level"
	"github.com/fuhsicloud/shorter/pkg/endpoint"
	svchttp "github.com/fuhsicloud/shorter/pkg/http"
	"github.com/fuhsicloud/shorter/pkg/logging"
	"github.com/fuhsicloud/shorter/pkg/repository/mongodb"
	"github.com/fuhsicloud/shorter/pkg/repository/redis"
	"github.com/fuhsicloud/shorter/pkg/service"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/oklog/oklog/pkg/group"
)

var logger log.Logger

var (
	fs = flag.NewFlagSet("hello", flag.ExitOnError)
	httpAddr = fs.String("http-addr", ":8080", "HTTP listen address")
	dbDrive = fs.String("db-drive", "redis", "db drive type, default: mongo")
	mongoAddr = fs.String("mongo-addr", "mongodb://localhost:32768", "mongodb uri, default: mongodb://localhost:27017")
	redisDrive = fs.String("redis-drive", "single", "redis drive: single or cluster")
	redisHosts = fs.String("redis-hosts", "122.152.197.181:6379", "redis hosts, many ';' split")
	redisPassword = fs.String("redis-password", "", "redis password")
	redisDB = fs.String("redis-db", "3", "redis db")
	shortUri = fs.String("short-uri", "http://localhost:8080", "short url")
	logPath = fs.String("log-path", "", "logging file path.")
	logLevel = fs.String("log-level", "all", "logging level.")
	devCors = fs.String("dev-cors", "false", "is develop")
	rateBucketNum = fs.Int("rate-bucket", 10, "rate bucket num")
	err           error
)

func Run() {
	if err:= fs.Parse(os.Args[1:]); err != nil {
		panic(err)
	}

	dbDrive = envString("DB_DRIVE", dbDrive)
	mongoAddr = envString("MONGO_ADDR", mongoAddr)
	redisDrive = envString("REDIS_DRIVE", redisDrive)
	redisHosts = envString("REDIS_HOSTS", redisHosts)
	redisPassword = envString("REDIS_PASSWORD", redisPassword)
	redisDB = envString("REDIS_DB", redisDB)
	shortUri = envString("SHORT_URI", shortUri)
	logPath = envString("LOG_PATH", logPath)
	logLevel = envString("LOG_LEVEL", logLevel)
	devCors = envString("DEV_CORS", devCors)
	rateBucketNum = envInt("RATE_BUCKET", rateBucketNum)

	logger = logging.SetLogging(logger, logPath, logLevel)

	var repo service.Repository
	switch * dbDrive {
	case "mongo":
		repo, err = mongodb.NewMongoRepository( * mongoAddr, "redirect", 60)
		if err != nil {
			_ = level.Error(logger).Log("connect", "db", "err", err.Error())
			return
		}
	case "redis":
		db, _:= strconv.Atoi( * redisDB)
		repo, err = redis.NewRedisRepository(redis.RedisDrive( * redisDrive),  * redisHosts,  * redisPassword, "shorter", db)
		if err != nil {
			_ = level.Error(logger).Log("connect", "db", "err", err.Error())
			return
		}
	}

	svc:= service.New(getServiceMiddleware(logger), logger, repo,  * shortUri)
	eps:= endpoint.New(svc, getEndpointMiddleware(logger))
	g:= createService(eps)
	initCancelInterrupt(g)
	_ = logger.Log("exit", g.Run())
}

func initHttpHandler(endpoints endpoint.Endpoints, g * group.Group) {
	options:= defaultHttpOptions(logger)

	httpHandler:= svchttp.NewHTTPHandler(endpoints, options)
	httpListener, err:= net.Listen("tcp",  * httpAddr)
	if err != nil {
		_ = level.Error(logger).Log("transport", "HTTP", "during", "Listen", "err", err)
	}
	g.Add(func()error {
		_ = level.Debug(logger).Log("transport", "HTTP", "addr",  * httpAddr)
		headers:= make(map[string]string)
		if isDev, _:= strconv.ParseBool( * devCors); isDev {
			headers = map[string]string {
				"Access-Control-Allow-Origin":"http://localhost:8000",
				"Access-Control-Allow-Methods":"GET,POST,OPTIONS,PUT,DELETE", 
				"Access-Control-Allow-Headers":"Origin,Content-Type,mode,Authorization,x-requested-with,Access-Control-Allow-Origin,Access-Control-Allow-Credentials", 
			}
		}
		return http.Serve(httpListener, accessControl(httpHandler, logger, headers))
	}, func(error) {
		_ = httpListener.Close()
	})

}
func getServiceMiddleware(logger log.Logger)(mw []service.Middleware) {
	mw = []service.Middleware {}
	mw = addDefaultServiceMiddleware(logger, mw)

	return
}
func getEndpointMiddleware(logger log.Logger)(mw map[string][]kitendpoint.Middleware) {
	mw = map[string][]kitendpoint.Middleware {}
	mw = addDefaultEndpointMiddleware(logger, mw)

	return
}

func initCancelInterrupt(g * group.Group) {
	cancelInterrupt:= make(chan struct {})
	g.Add(func()error {
		c:= make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		select {
		case sig := <- c:
			return fmt.Errorf("received signal %s", sig)
		case <- cancelInterrupt:
			return nil
		}
	}, func(error) {
		close(cancelInterrupt)
	})
}

func accessControl(h http.Handler, logger log.Logger, headers map[string]string)http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r * http.Request) {
		for key, val:= range headers {
			w.Header().Set(key, val)
		}
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Connection", "keep-alive")

		if r.Method == "OPTIONS" {
			return
		}

		_ = level.Info(logger).Log("remote-addr", r.RemoteAddr, "uri", r.RequestURI, "method", r.Method, "length", r.ContentLength)
		h.ServeHTTP(w, r)
	})
}

func envString(env string, fallback * string) * string {
	e:= os.Getenv(env)
	if e == "" {
		_ = os.Setenv(env,  * fallback)
		return fallback
	}
	return & e
}

func envInt(env string, fallback * int) * int {
	e:= os.Getenv(env)
	if e == "" {
		_ = os.Setenv(env, strconv.Itoa( * fallback))
		return fallback
	}
	num, _:= strconv.Atoi(e)
	return & num
}
