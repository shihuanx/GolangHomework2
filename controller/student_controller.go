package controller

import (
	"github.com/gin-gonic/gin"
	"log"
	"memoryDataBase/model"
	"memoryDataBase/response"
	"memoryDataBase/service"
	"net/http"
)

// StudentController 定义控制层结构体实例
type StudentController struct {
	studentService *service.StudentService
}

func NewStudentController(studentService *service.StudentService) *StudentController {
	return &StudentController{
		studentService: studentService,
	}
}

// AddStudent 处理添加学生信息的 HTTP 请求
func (sc *StudentController) AddStudent(c *gin.Context) {
	var student model.Student
	if err := c.BindJSON(&student); err != nil {
		log.Printf("StudentController.AddStudent err：%v", err.Error())
		c.JSON(http.StatusBadRequest, response.Error(err.Error()))
		// 调用服务层方法添加学生信息
	} else if err = sc.studentService.AddStudent(&student); err != nil {
		log.Printf("StudentController.AddStudent err：%v", err.Error())
		c.JSON(http.StatusBadRequest, response.Error(err.Error()))
	} else {
		log.Printf("添加学号为：%s的学生", student.ID)
		c.JSON(http.StatusOK, response.SuccessWithoutData())
	}
}

// GetStudent 处理获取学生信息的 HTTP 请求
func (sc *StudentController) GetStudent(c *gin.Context) {
	studentId := c.Param("id")
	// 调用服务层方法获取学生信息
	resp, err := sc.studentService.GetStudent(studentId)
	if err != nil {
		log.Printf("StudentController.GetStudent err：%v", err.Error())
		c.JSON(500, response.Error(err.Error()))
	} else {
		log.Printf("查询学号为：%s的学生", studentId)
		c.JSON(http.StatusOK, response.Success(resp))
	}
}

// UpdateStudent 处理更新学生信息的 HTTP 请求
func (sc *StudentController) UpdateStudent(c *gin.Context) {
	var student model.Student
	// 从请求的 JSON 数据中解析出学生信息，并绑定到student上
	if err := c.BindJSON(&student); err != nil {
		log.Printf("StudentController.UpdateStudent err：%v", err.Error())
		c.JSON(http.StatusBadRequest, response.Error(err.Error()))
		return
	}
	// 调用服务层方法，更新学生信息
	err := sc.studentService.UpdateStudent(&student)
	if err != nil {
		log.Printf(err.Error())
		c.JSON(http.StatusNotFound, response.Error(err.Error()))
	} else {
		log.Printf("修改学生：%s", student.ID)
		c.JSON(http.StatusOK, response.SuccessWithoutData())
	}
}

// DeleteStudent 处理删除学生信息的 HTTP 请求
func (sc *StudentController) DeleteStudent(c *gin.Context) {
	studentId := c.Param("id")
	// 调用服务层方法，删除学生信息
	err := sc.studentService.DeleteStudent(studentId)
	if err != nil {
		log.Printf("StudentController.DeleteStudent err：%v", err.Error())
		c.JSON(http.StatusNotFound, response.Error(err.Error()))
	} else {
		log.Printf("删除学号为：%s的学生", studentId)
		c.JSON(http.StatusOK, response.SuccessWithoutData())
	}
}

// JoinRaftCluster 向领导者节点发送请求 把自身加入到集群中
func (sc *StudentController) JoinRaftCluster(c *gin.Context) {
	nodeID := c.Query("nodeID")
	nodeAddress := c.Query("nodeAddress")
	nodePortAddress := c.Query("portAddress")
	if err := sc.studentService.JoinRaftCluster(nodeID, nodeAddress, nodePortAddress); err != nil {
		log.Printf("StudentController.JoinRaftCluster err:%v", err)
		c.JSON(500, response.Error(err.Error()))
	} else {
		log.Printf("添加节点：%s成功", nodeID)
		c.JSON(http.StatusOK, response.SuccessWithoutData())
	}
}

// LeaderHandleCommand 在找到领导者的端口后会向领导者端口发送命令 这个接口会处理这些命令
func (sc *StudentController) LeaderHandleCommand(c *gin.Context) {
	cmdData := c.Query("cmd")
	if err := sc.studentService.LeaderHandleCommand(cmdData); err != nil {
		log.Printf("StudentController.LeaderHandleCommand err:%v", err)
		c.JSON(500, response.Error(err.Error()))
	} else {
		log.Printf("领导者节点已处理命令")
		c.JSON(http.StatusOK, response.SuccessWithoutData())
	}
}

// GetLeaderPortAddress 获取领导者端口的地址 方法是向所有节点都通过此端口发送请求 领导者端口会返回自己的端口地址
func (sc *StudentController) GetLeaderPortAddress(c *gin.Context) {
	leaderAddr := sc.studentService.HandleGetLeaderPortAddressRequest()
	if leaderAddr != "" {
		c.JSON(http.StatusOK, response.Success(leaderAddr))
	}
}
