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
	// 声明 student 变量，用于存储请求中的学生信息
	var student model.Student
	// 从请求的 JSON 数据解析学生信息到 student 变量
	if err := c.BindJSON(&student); err != nil {
		// 解析失败，记录错误并返回 400 响应
		log.Printf("StudentController.AddStudent err：%v", err.Error())
		c.JSON(http.StatusBadRequest, response.Error(err.Error()))
		// 调用服务层方法添加学生信息
	} else if err = sc.studentService.AddStudentInternal(&student); err != nil {
		// 如果出错，记录错误日志并返回错误响应
		log.Printf("StudentController.AddStudent err：%v", err.Error())
		c.JSON(http.StatusBadRequest, response.Error(err.Error()))
	} else {
		//记录日志
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
		// 如果出错，记录错误日志并返回错误响应
		log.Printf("StudentController.GetStudent err：%v", err.Error())
		c.JSON(500, response.Error(err.Error()))
	} else {
		//记录日志
		log.Printf("查询学号为：%s的学生", studentId)
		c.JSON(http.StatusOK, response.Success(resp))
	}
}

// UpdateStudent 处理更新学生信息的 HTTP 请求
func (sc *StudentController) UpdateStudent(c *gin.Context) {
	var student model.Student
	// 从请求的 JSON 数据中解析出学生信息，并绑定到student上
	if err := c.BindJSON(&student); err != nil {
		// 如果解析JSON数据失败，记录错误日志并返回
		log.Printf("StudentController.UpdateStudent err：%v", err.Error())
		c.JSON(http.StatusBadRequest, response.Error(err.Error()))
		return
	}
	// 调用服务层方法，更新学生信息
	err := sc.studentService.UpdateStudentInternal(&student)
	// 如果出错 记录错误日志并返回错误信息
	if err != nil {
		log.Printf(err.Error())
		c.JSON(http.StatusNotFound, response.Error(err.Error()))
	} else {
		// 记录日志
		log.Printf("修改学生：%s", student.ID)
		c.JSON(http.StatusOK, response.SuccessWithoutData())
	}
}

// DeleteStudent 处理删除学生信息的 HTTP 请求
func (sc *StudentController) DeleteStudent(c *gin.Context) {
	studentId := c.Param("id")
	// 调用服务层方法，删除学生信息
	err := sc.studentService.DeleteStudentInternal(studentId)
	if err != nil {
		// 如果出错，记录错误日志并返回错误响应
		log.Printf("StudentController.DeleteStudent err：%v", err.Error())
		c.JSON(http.StatusNotFound, response.Error(err.Error()))
	} else {
		// 记录日志
		log.Printf("删除学号为：%s的学生", studentId)
		c.JSON(http.StatusOK, response.SuccessWithoutData())
	}
}

// JoinRaftCluster 向领导者节点发送请求 把自身加入到集群中
func (sc *StudentController) JoinRaftCluster(c *gin.Context) {
	nodeID := c.Query("nodeID")
	nodeAddress := c.Query("nodeAddress")
	if err := sc.studentService.JoinRaftCluster(nodeID, nodeAddress); err != nil {
		log.Printf("StudentController.JoinRaftCluster err:%v", err)
		c.JSON(500, response.Error(err.Error()))
	} else {
		log.Printf("添加节点：%s成功", nodeID)
		c.JSON(http.StatusOK, response.SuccessWithoutData())
	}
}
