package service

import (
	"encoding/json"
	"fmt"
	raftfpk "github.com/hashicorp/raft"
	"io"
	"log"
	"memoryDataBase/config"
	"memoryDataBase/interfaces"
	"memoryDataBase/model"
	"memoryDataBase/raft"
	"memoryDataBase/raft/fsm"
	"memoryDataBase/response"
	"net/http"
	"strings"
	"time"
)

// StudentService 定义学生服务层结构体
type StudentService struct {
	MdbService   *StudentMdbService
	MysqlService *StudentMysqlService
	CacheService *StudentCacheService
	raftNode     *raftfpk.Raft
	node         config.Node
	peers        []*config.Peer
}

// NewStudentService 创建并初始化 StudentService 实例
func NewStudentService(mdbService *StudentMdbService, mysqlService *StudentMysqlService, cacheService *StudentCacheService, node config.Node, peers []*config.Peer) (*StudentService, error) {

	ss := &StudentService{
		MdbService:   mdbService,
		MysqlService: mysqlService,
		CacheService: cacheService,
		raftNode:     new(raftfpk.Raft),
		node:         node,
		peers:        peers,
	}

	initializer := &raft.RaftInitializerImpl{}

	raftNode, err := initializer.InitRaft(node, peers, ss)
	if err != nil {
		return nil, fmt.Errorf("初始化 Raft 节点 %s 时出错: %w", node.NodeId, err)
	}
	ss.raftNode = raftNode
	return ss, nil
}

// 确保实现 StudentServiceInterface 接口
var _ interfaces.StudentServiceInterface = (*StudentService)(nil)

// StudentNotFoundErr 判断错误是不是没有找到学生之类的错误 如果是那就继续去下一个数据源找 不返回
func (ss *StudentService) StudentNotFoundErr(err error) bool {
	studentNotFoundErrMsg := fmt.Sprintf("不存在学生")
	return strings.Contains(err.Error(), studentNotFoundErrMsg)
}

func (ss *StudentService) StudentExists(id string) bool {
	student, err := ss.MysqlService.GetStudentFromMysql(id)
	if err != nil {
		log.Printf("查找学生：%s失败：%v", id, err)
		return false
	}
	if student != nil {
		return true
	}
	return false
}

// JoinRaftCluster 将节点加入 Raft 集群
func (ss *StudentService) JoinRaftCluster(nodeID string, nodeAddress string,nodePortAddress string) error {
	//再次确保是领导者节点才会处理加入集群的请求
	if ss.raftNode.State() == raftfpk.Leader {
		future := ss.raftNode.AddVoter(raftfpk.ServerID(nodeID), raftfpk.ServerAddress(nodeAddress), 0, 0)
		if err := future.Error(); err != nil {
			return err
		}
		log.Printf("领导者节点已将节点：%s加入集群", nodeID)

		var newPeer *config.Peer
		newPeer.NodeId=nodeID
		newPeer.Address=nodeAddress
		newPeer.PortAddress=nodePortAddress

		err := ss.ApplyRaftCommandToLeader("updatePeers",nil,"",0,newPeer)
		if err != nil {
			log.Printf("领导者节点更新所有节点的Peers失败：%v", err)
			return err
		}

		return nil
	}
	return nil
}

func (ss *StudentService)UpdatePeersInternal(peer *config.Peer){
	ss.peers=append(ss.peers,peer)
	return
}

// HandleGetLeaderPortAddressRequest 处理获取领导者地址的请求 返回领导者的端口号
func (ss *StudentService) HandleGetLeaderPortAddressRequest() string {
	if ss.raftNode.State() == raftfpk.Leader {
		log.Printf("节点：%s是领导者节点", ss.node.NodeId)
		return ss.node.PortAddress
	}
	return ""
}

// GetLeaderPortAddr 获取领导者端口地址 向集群的各个节点都发送一个http请求 如果他是领导者节点 他就会把自己的端口号返回过来
func (ss *StudentService) GetLeaderPortAddr() (string, error) {
	for _, node := range ss.peers {
		url := fmt.Sprintf("http://localhost:%s/GetLeaderAddress", node.PortAddress)
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("请求出错：%v", err)
			return "", err
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("读取响应体出错：%v", err)
			return "", err
		}
		// 解析 JSON 响应
		var result response.Result
		err = json.Unmarshal(body, &result)
		if err != nil {
			fmt.Printf("解析 JSON 数据出错: %v\n", err)
			return "", err
		}

		// 提取 leaderAddr
		leaderPortAddr, ok := result.Data.(string)
		if ok {
			return leaderPortAddr, nil
		}
		return "", fmt.Errorf("领导者地址 类型断言失败")
	}
	return "", fmt.Errorf("获取领导者地址失败")
}

// ApplyRaftCommandToLeader 将命令提交给领导者处理
func (ss *StudentService) ApplyRaftCommandToLeader(operation string, student *model.Student, id string, examineSize int,peer *config.Peer) error {
	// 创建 Node 命令
	cmd := fsm.StudentCommand{
		Operation:   operation,
		Student:     student,
		Id:          id,
		ExamineSize: examineSize,
		Peer  		:peer,
	}
	// 序列化命令
	cmdData, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("ApplyRaftCommandToLeader Marshal err: %w", err)
	}
	//如果自己是领导者节点 那就处理这个命令
	if ss.raftNode.State() == raftfpk.Leader {
		// 提交命令到领导者 Node 节点
		future := ss.raftNode.Apply(cmdData, 500)
		if err = future.Error(); err != nil {
			return fmt.Errorf("ApplyRaftCommandToLeader 处理命令失败：%w", err)
		}
		// 处理响应
		result := future.Response()
		if resultErr, ok := result.(error); ok {
			return resultErr
		}
		log.Printf("领导者节点已接收并提交命令到状态机")
		return nil
	} else {
		//如果不是 那就找到领导者节点的端口 把命令交给领导者节点处理
		leaderPortAddr, err := ss.GetLeaderPortAddr()
		if err != nil {
			return fmt.Errorf("StudentService.ApplyRaftCommandToLeader 获取领导者地址失败：%w", err)
		}
		url := fmt.Sprintf("http://localhost:%s/LeaderHandleCommand?cmd=%s", leaderPortAddr, cmdData)
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("将cmd命令：%s发送给领导者失败：%v", cmdData, err)
			return fmt.Errorf("将cmd命令：%s发送给领导者失败：%v", cmdData, err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("读取响应体出错：%v", err)
			return err
		}
		// 解析 JSON 响应
		var result response.Result
		err = json.Unmarshal(body, &result)
		if err != nil {
			fmt.Printf("解析 JSON 数据出错: %v\n", err)
		}
		//把错误信息返回给前端发送的对应端口
		if result.Code != 1 {
			return fmt.Errorf("领导者节点处理命令失败：%v", result.Message)
		}
		return nil
	}
}

// LeaderHandleCommand 领导者节点会处理命令 并发送到状态机
func (ss *StudentService) LeaderHandleCommand(data string) error {
	future := ss.raftNode.Apply([]byte(data), 500)
	if err := future.Error(); err != nil {
		return fmt.Errorf("ApplyRaftCommandToLeader 处理命令失败：%w", err)
	}
	// 处理响应
	result := future.Response()
	if resultErr, ok := result.(error); ok {
		return resultErr
	}
	log.Printf("领导者节点已接收并提交命令到状态机")
	return nil
}

// RestoreCacheData 恢复缓存机制 mysql有事务可以很方便地回滚 此函数专门用于恢复缓存的数据
func (ss *StudentService) RestoreCacheData(id string) error {
	//如果要恢复数据 mysql的事务会回滚 所以这个时候找到的学生还是一开始的学生
	studentBackUp, err := ss.MysqlService.GetStudentFromMysql(id)
	if err != nil {
		return fmt.Errorf("StudentService.RestoreCacheData 尝试通过学生id：%s获取学生时失败：%w", id, err)
	}
	if err = ss.CacheService.AddStudent(studentBackUp); err != nil {
		return fmt.Errorf("StudentService.RestoreCacheData 尝试恢复缓存数据时失败: %w", err)
	}
	return nil
}

// ReLoadCacheDataInternal 重新加载缓存数据
func (ss *StudentService) ReLoadCacheDataInternal() {
	// 从 MySQL 中获取访问最多的学生
	students, err := ss.MysqlService.GetHotStudentsFromMysql()
	if err != nil {
		log.Printf("StudentService.ReLoadCacheDataInternal 获得访问最多的学生时出错：%v", err)
	}
	// 将学生添加到缓存
	err = ss.CacheService.ReLoadCacheData(students)
	if err != nil {
		log.Printf("StudentService.ReLoadCacheDataInternal 重新加载缓存失败：%v", err)
	}
	log.Printf("已重新加载缓存: %v", time.Now())
}

// PeriodicDeleteInternal 定期删除内存中的过期键 接收一个淘汰键的数量
func (ss *StudentService) PeriodicDeleteInternal(examineSize int) {
	log.Printf("定期删除内存中的过期键：%v", time.Now())
	ss.MdbService.PeriodicDelete(examineSize)
}

// LoadCacheToMemory 加载缓存到内存
func (ss *StudentService) LoadCacheToMemory(capacity int, addRadio float64) error {
	// 从缓存中获取所有学生
	students, err := ss.CacheService.GetAllStudentsFromCache()
	if err != nil {
		return fmt.Errorf("StudentService.LoadCacheToMemory 从缓存中获取所有学生时失败：%w", err)
	}
	// 定义一个已加载学生数量的计数器 如果超过了容量*加载学生占内存总容量的比例 就停止添加
	addCount := 0
	for _, student := range students {
		// 将学生添加到内存数据库
		ss.MdbService.AddStudent(student)
		addCount++
		if addCount >= int(float64(capacity)*addRadio) {
			log.Printf("从缓存加载到内存 已达到内存容量%d的%f 停止添加 共添加%d个键值对", capacity, addRadio, addCount)
			return nil
		}
	}
	return nil
}

// LoadDateBaseToMemory 加载数据库的学生到内存
func (ss *StudentService) LoadDateBaseToMemory(capacity int, addRadio float64) error {
	// 从数据库中获取热门学生
	students, err := ss.MysqlService.GetHotStudentsFromMysql()
	if err != nil {
		return fmt.Errorf("StudentService.LoadDateBaseToMemory 从数据库中获取热门学生失败：%w", err)
	}
	// 定义一个已加载学生数量的计数器 如果超过了容量*加载学生占内存总容量的比例 就停止添加
	addCount := 0
	for _, student := range students {
		// 将学生添加到内存数据库
		ss.MdbService.AddStudent(student)
		addCount++
		if addCount >= int(float64(capacity)*addRadio) {
			log.Printf("从数据库加载到内存 已达到内存容量%d的%f 停止添加 共添加%d个键值对", capacity, addRadio, addCount)
			return nil
		}
	}
	return nil
}

// AddStudentInternal 向数据库和缓存和内存添加学生
func (ss *StudentService) AddStudentInternal(student *model.Student) error {
	// 开始 MySQL 事务
	tx := ss.MysqlService.mysqlDao.DB.Begin()
	if tx.Error != nil {
		return fmt.Errorf("StudentService.AddStudentInternal 开启 MySQL 事务失败：%w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			// 发生 panic 时回滚事务
			tx.Rollback()
			log.Printf("事务已回滚：%v", r)
		}
	}()

	//如果学生已经有其他节点添加到数据库和缓存了 那本节点只更新内存即可
	if ss.StudentExists(student.ID) {
		ss.MdbService.AddStudent(student)
		return nil
	}

	// 在 MySQL 数据库事务中添加学生信息
	if err := ss.MysqlService.AddStudentToMysql(tx, student); err != nil {
		return err
	}

	// MySQL 数据库事务提交成功后，尝试添加到缓存
	if err := ss.CacheService.AddStudent(student); err != nil {
		tx.Rollback()
		//缓存中添加失败 就回滚 确保数据一致性 （添加其实好像不影响 更新和删除如果有错误肯定要回滚保持数据一致性的）
		log.Printf("添加缓存失败 回滚事务")
		return err
	}
	// 最后添加到内存数据库
	ss.MdbService.AddStudent(student)
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("StudentService.AddStudentInternal 提交事务失败：%w", err)
	}
	// 添加学生访问次数
	ss.MysqlService.AddStudentCount(student.ID)
	return nil
}

// GetStudent 获取学生
func (ss *StudentService) GetStudent(id string) (*model.Student, error) {
	// 先从内存中查找学生
	student, memoryErr := ss.MdbService.GetStudent(id)
	if memoryErr != nil {
		log.Printf(memoryErr.Error())
	}
	if student != nil {
		ss.MysqlService.AddStudentCount(id)
		log.Printf("从内存中查找到了学生：%s", id)
		return student, nil
	}

	//再从缓存中查找学生
	student, cacheErr := ss.CacheService.GetStudentFromCache(id)
	if cacheErr != nil {
		log.Printf(cacheErr.Error())
	}
	if student != nil {
		ss.MysqlService.AddStudentCount(id)
		log.Printf("从缓存中查找到了学生：%s", id)
		//如果确定内存里没有这个学生 就向内存中添加学生
		if ss.StudentNotFoundErr(memoryErr) {
			ss.MdbService.AddStudent(student)
			log.Printf("从缓存向内存中添加学生：%s", id)
		}
		return student, nil
	}

	// 最后从数据库中查找学生
	student, mysqlErr := ss.MysqlService.GetStudentFromMysql(id)
	if mysqlErr != nil {
		log.Printf(mysqlErr.Error())
		return nil, mysqlErr
	}
	if student != nil {
		ss.MysqlService.AddStudentCount(id)
		log.Printf("在数据库中查找到了学生：%s", id)
		//如果确定内存和缓存没有学生 就向内存和缓存中添加学生
		if ss.StudentNotFoundErr(memoryErr) {
			ss.MdbService.AddStudent(student)
			log.Printf("从数据库向内存中添加学生：%s", id)
		}
		if ss.StudentNotFoundErr(cacheErr) {
			err := ss.CacheService.AddStudent(student)
			if err != nil {
				log.Printf("从数据库向缓存中添加学生：%s失败：%v", id, err)
			} else {
				log.Printf("从数据库向缓存中添加学生：%s", id)
			}
		}
		return student, nil
	}
	//不可到达的代码 因为不存在既没有错误又没有学生的情况 没有学生也是一个错误
	return nil, fmt.Errorf("StudentService.GetStudent 错误到达的代码")
}

// UpdateStudentInternal 更新学生
func (ss *StudentService) UpdateStudentInternal(student *model.Student) error {
	// 开始 MySQL 事务
	tx := ss.MysqlService.mysqlDao.DB.Begin()
	if tx.Error != nil {
		return fmt.Errorf("StudentService.UpdateStudentInternal 开启 MySQL 事务失败：%w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			// 发生 panic 时回滚事务
			tx.Rollback()
			log.Printf("事务已回滚：%v", r)
		}
	}()

	// 在 MySQL 数据库事务中更新学生信息
	if err := ss.MysqlService.UpdateStudent(tx, student); err != nil {
		return fmt.Errorf("StudentService.UpdateStudentInternal 更新学生：%s时失败：%w", student.ID, err)
	}
	// MySQL 数据库事务提交成功后，尝试更新缓存和内存 还要确保数据一致性
	if err := ss.CacheService.UpdateStudent(student); err != nil {
		if !ss.StudentNotFoundErr(err) {
			tx.Rollback()
			log.Printf("更新缓存失败 回滚事务")
			return fmt.Errorf("StudentService.UpdateStudentInternal 更新缓存中的学生：%s时失败：%w", student.ID, err)
		}
	}
	if err := ss.MdbService.UpdateStudent(student); err != nil {
		if !ss.StudentNotFoundErr(err) && !strings.Contains(err.Error(), "过期") {
			tx.Rollback()
			if err = ss.RestoreCacheData(student.ID); err != nil {
				return fmt.Errorf("StudentService.UpdateStudentInternal 更新内存中的学生失败后 尝试恢复缓存数据时失败：%w", err)
			}
			log.Printf("回滚数据库事务并恢复缓存")
			return fmt.Errorf("StudentService.UpdateStudentInternal 更新内存中的学生：%s失败：%w", student.ID, err)

		}
	}
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("StudentService.UpdateStudentInternal 回滚事务失败：%w", err)
	}
	// 添加学生访问次数
	ss.MysqlService.AddStudentCount(student.ID)
	return nil
}

// DeleteStudentInternal 删除学生 分别删除三个数据库的数据 然后再提交事务 保证数据一致性
func (ss *StudentService) DeleteStudentInternal(id string) error {
	// 开始 MySQL 事务
	tx := ss.MysqlService.mysqlDao.DB.Begin()
	if tx.Error != nil {
		return fmt.Errorf("StudentService.DeleteStudentInternal 开启 MySQL 事务失败：%w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			// 发生 panic 时回滚事务
			tx.Rollback()
			log.Printf("事务已回滚：%v", r)
		}
	}()

	if err := ss.CacheService.DeleteStudent(id); err != nil {
		if !ss.StudentNotFoundErr(err) {
			tx.Rollback()
			log.Printf("删除缓存数据失败 回滚事务")
			return fmt.Errorf("StudentService.DeleteStudentInternal 从缓存中删除学生：%s失败：%w", id, err)
		}
	}

	if err := ss.MdbService.DeleteStudent(id); err != nil {
		if !ss.StudentNotFoundErr(err) {
			tx.Rollback()
			log.Printf("从内存中删除学生：%s失败：%v", id, err)
			if err = ss.RestoreCacheData(id); err != nil {
				return fmt.Errorf("StudentService.DeleteStudentInternal 从内存中删除学生失败后 尝试恢复缓存数据失败：%w", err)
			}
			log.Printf("回滚数据库事务并恢复缓存")
		}
	}

	if err := ss.MysqlService.DeleteStudent(tx, id); err != nil {
		return fmt.Errorf("StudentService.DeleteStudentInternal err: %w", err)
	}
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("StudentService.DeleteStudentInternal 提交事务失败: %w", err)
	}
	// 删除学生访问次数
	ss.MysqlService.DeleteStudentCount(id)
	return nil
}

// ReLoadCacheData 重新加载缓存 并提交给Raft节点
func (ss *StudentService) ReLoadCacheData(interval time.Duration) {
	// 每隔一段时间重新加载缓存
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := ss.ApplyRaftCommandToLeader("reloadCacheData", nil, "", 0,nil)
			if err != nil {
				log.Printf("StudentService.ReLoadCacheData 分布式加载缓存数据失败: %v，跳过这次操作", err)
				continue
			}
		}
	}
}

// PeriodicDelete 定期删除内存中的过期键 并提交给Raft节点
func (ss *StudentService) PeriodicDelete(interval time.Duration, examineSize int) {
	// 每隔一段时间删除内存中的过期键
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := ss.ApplyRaftCommandToLeader("periodicDelete", nil, "", examineSize,nil)
			if err != nil {
				log.Printf("StudentService.PeriodicDelete 分布式删除内存数据库过期键失败：: %v，跳过这次操作", err)
				continue
			}
		}
	}
}

// AddStudent 接收添加学生命令 提交给Raft节点
func (ss *StudentService) AddStudent(student *model.Student) error {
	return ss.ApplyRaftCommandToLeader("add", student, "", 0,nil)
}

// UpdateStudent 接收更新学生命令 提交给Raft节点
func (ss *StudentService) UpdateStudent(student *model.Student) error {
	return ss.ApplyRaftCommandToLeader("update", student, "", 0,nil)
}

// DeleteStudent 接收删除学生命令 提交给Raft节点
func (ss *StudentService) DeleteStudent(id string) error {
	return ss.ApplyRaftCommandToLeader("delete", nil, id, 0,nil)
}
