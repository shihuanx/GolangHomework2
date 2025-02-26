package main

import (
	"log"
	"memoryDataBase/cache"
	"memoryDataBase/config"
	"memoryDataBase/controller"
	"memoryDataBase/dao"
	"memoryDataBase/database"
	"memoryDataBase/routers"
	"memoryDataBase/service"
)

func main() {
	// 初始化数据库和缓存
	cfg := config.GetConfig()

	if err := database.InitDB(cfg.MySQL.DSN); err != nil {
		log.Fatalf("节点：%s 初始化数据库失败: %v", cfg.Node.NodeId, err)
	}
	cache.InitRedis(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)

	// 初始化 DAO
	studentCacheDao := dao.NewStudentCacheDao(cache.RedisClient)
	studentMysqlDao := dao.NewStudentMysqlDao(database.DB)
	memoryDBDao := dao.NewMemoryDBDao(cfg.MemoryDB.Capacity, cfg.MemoryDB.EvictRatio)

	// 初始化服务
	studentCacheService := service.NewStudentCacheService(studentCacheDao)
	studentMysqlService := service.NewStudentMysqlService(studentMysqlDao)
	studentMdbService := service.NewStudentMdbService(memoryDBDao)
	studentService, err := service.NewStudentService(studentMdbService, studentMysqlService, studentCacheService, cfg.Node, cfg.Peers)
	if err != nil {
		log.Fatalf("节点：%s 初始化学生服务层失败：%v", cfg.Node.NodeId, err)
	}

	// 初始化控制器
	studentController := controller.NewStudentController(studentService)

	//启动时加载缓存数据到内存
	if err = studentService.LoadCacheToMemory(cfg.MemoryDB.Capacity, cfg.CachePreheating.LoadRatio); err != nil {
		log.Printf("节点：%s 加载缓存到内存时失败：%v", cfg.Node.NodeId, err)
		if err = studentService.LoadDateBaseToMemory(cfg.MemoryDB.Capacity, cfg.CachePreheating.LoadRatio); err != nil {
			log.Printf("节点：%s 加载数据库中的数据到内存时失败：%v", cfg.Node.NodeId, err)
		}
		log.Printf("节点：%s 加载数据库到内存", cfg.Node.NodeId)
	}
	log.Printf("节点：%s 加载缓存到内存", cfg.Node.NodeId)

	//定期清空缓存 定期清除内存中的过期键 让领导者节点提交命令给所有节点
	leaderPortAddr, err := studentService.GetLeaderPortAddr()
	if err != nil {
		log.Fatalf("节点：%s 获取领导者端口地址失败：%v", cfg.Node.NodeId, err)
	}
		go func() {
			if cfg.Node.PortAddress == leaderPortAddr{
				studentService.ReLoadCacheData(cfg.Server.ReloadInterval)
			}
		}()

		//定期删除内存数据库过期键
		go func() {
			if cfg.Node.PortAddress == leaderPortAddr{
				studentService.PeriodicDelete(cfg.Server.PeriodicDeleteInterval, cfg.Server.ExamineSize)
			}
		}()
	//初始化路由
	studentRouter := routers.SetUpStudentRouter(studentController)
	serverAddress := ":" + cfg.Node.PortAddress
	if err = studentRouter.Run(serverAddress); err != nil {
		log.Fatalf("节点：%s 初始化学生路由时出错：%v", cfg.Node.NodeId, err)
	}
}
