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
	"sync"
)

func startNode(node config.SingleNode, cfg config.Config, index int) {
	// 初始化数据库和缓存
	if err := database.InitDB(cfg.MySQL.DSN); err != nil {
		log.Fatalf("节点：%s 初始化数据库失败: %v", node.NodeId, err)
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
	studentService, err := service.NewStudentService(studentMdbService, studentMysqlService, studentCacheService, node, cfg, index)
	if err != nil {
		log.Fatalf("节点：%s 初始化学生服务层失败：%v", node.NodeId, err)
	}

	// 初始化控制器
	studentController := controller.NewStudentController(studentService)

	//启动时加载缓存数据到内存
	if err = studentService.LoadCacheToMemory(cfg.MemoryDB.Capacity, cfg.CachePreheating.LoadRatio); err != nil {
		log.Printf("节点：%s 加载缓存到内存时失败：%v", node.NodeId, err)
		if err = studentService.LoadDateBaseToMemory(cfg.MemoryDB.Capacity, cfg.CachePreheating.LoadRatio); err != nil {
			log.Printf("节点：%s 加载数据库中的数据到内存时失败：%v", node.NodeId, err)
		}
		log.Printf("节点：%s 加载数据库到内存", node.NodeId)
	}
	log.Printf("节点：%s 加载缓存到内存", node.NodeId)

	//定期清空缓存 加载mysql中访问记录多的数据
	go func() {
		studentService.ReLoadCacheData(cfg.Server.ReloadInterval)
	}()

	//定期删除内存数据库过期键
	go func() {
		studentService.PeriodicDelete(cfg.Server.PeriodicDeleteInterval, cfg.Server.ExamineSize)
	}()

	//初始化路由
	studentRouter := routers.SetUpStudentRouter(studentController)
	serverAddress := ":" + node.PortAddress
	if err = studentRouter.Run(serverAddress); err != nil {
		log.Fatalf("节点：%s 初始化学生路由时出错：%v", node.NodeId, err)
	}
}

func main() {
	//加载配置信息
	cfg := config.GetConfig()

	var wg sync.WaitGroup
	for index, node := range cfg.Node.Nodes {
		wg.Add(1)
		go func(n config.SingleNode) {
			defer wg.Done()
			startNode(n, cfg, index)
		}(node)
	}

	wg.Wait()
}
