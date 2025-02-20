package config

import (
	"time"
)

// Config 定义配置结构体
type Config struct {
	MySQL struct {
		DSN string
	}
	Redis struct {
		Addr     string
		Password string
		DB       int
	}
	MemoryDB struct {
		Capacity   int
		EvictRatio float64
	}
	CachePreheating struct {
		LoadRatio float64
	}
	Server struct {
		Address                string
		ReloadInterval         time.Duration
		PeriodicDeleteInterval time.Duration
		ExamineSize            int
	}
	Raft struct {
		LocalID string
	}
}

// GetConfig 获取配置实例
func GetConfig() Config {
	return Config{
		//配置Mysql
		MySQL: struct {
			DSN string
		}{
			DSN: "root:1234@tcp(127.0.0.1:3306)/mdb?charset=utf8mb4&parseTime=True&loc=Local",
		},
		//配置redis
		Redis: struct {
			Addr     string
			Password string
			DB       int
		}{
			Addr:     "192.168.88.128:6379",
			Password: "123456",
			DB:       0,
		},
		//配置内存数据库
		MemoryDB: struct {
			Capacity   int     //内存容量
			EvictRatio float64 //内存淘汰比例
		}{
			Capacity:   10,
			EvictRatio: 0.2,
		},
		CachePreheating: struct {
			LoadRatio float64 //缓存预热时最多加载到的容量比例
		}{
			LoadRatio: 0.5,
		},
		Server: struct {
			Address                string        //服务器端口
			ReloadInterval         time.Duration //重载缓存数据的时间间隔
			PeriodicDeleteInterval time.Duration //定期删除过期键的时间间隔
			ExamineSize            int           //定期删除过期键的检测数量
		}{
			Address:                ":8080",
			ReloadInterval:         time.Hour,
			PeriodicDeleteInterval: time.Hour,
			ExamineSize:            10,
		},
		Raft: struct {
			LocalID string //Raft：服务器名称 可以起名字 也可以用ip地址代替
		}{
			LocalID: "127.0.0.1"},
	}
}
