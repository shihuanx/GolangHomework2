项目总体使用了controller service dao层三层架构实现了一个分布式内存数据库 内存数据库用了map集合 线程模型采用读写互斥锁

使用Raft一致性协议保证了各个节点数据一致 节点间通过TCP通信 将Raft节点添加到集群通过Gin框架实现 已经实现了自动寻找领导者节点 并把命令提交给他

项目实现了学生的增删改查业务 并且用了内存数据库 redis mysql三级缓存 实现了按照内存-缓存-mysql的顺序查找学生 添加、修改、删除通过mysql事务、redis备份避免了出现异常导致的数据不一致

缓存预热的实现 通过先尝试通过缓存加载数据到内存 如果缓存加载失败了 就再尝试从mysql加载数据到内存 内存设置了最大容量 如果超过容量会停止添加 
还实现了缓存的定期删除 重新从mysql数据库中加载访问次数前几的键 这样可以增加缓存预热到内存中的键的访问命中率 以及缓存的访问命中率

过期键删除采用定期删除策略 每隔一段时间随机取一定设置了过期时间的键检查 如果过期就删除 此外在访问键时如果发现过期 也会删除 如果不设置过期时间 则永久保存

内存淘汰采用LRU算法 在内存数据库设置了一个双向列表 在添加键时检查是否满了 如果内存满了就删除列表尾部一定数量的键 其他删除 更新 查询方法只把键移动到双向链表头部

最后是一些学生增删改查的具体接口 目前项目只用了三个节点 分别占8080 8081 8082端口

查询学生：GET localhost:8080/student/1 参数：id

添加学生：POST localhost:8080/student 
参数：json形式 id：string类型，name：string类型，class：string类型，gender：string类型 grades：map[string]float64 expiration:过期时间 默认是0 即永久保存 

修改学生: PUT localhost:8080/student
参数：json形式 id：string类型 必填 其他的name，class，gender，grades选填 不填就不修改

删除学生：DELETE localhost:8080/student/id 参数：id
