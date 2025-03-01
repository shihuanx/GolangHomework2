package routers

import (
	"github.com/gin-gonic/gin"
	"node2/controller"
)

func SetUpStudentRouter(studentController *controller.StudentController) *gin.Engine {
	r := gin.Default()
	// 创建一个学生组
	studentGroup := r.Group("/student")

	studentGroup.POST("", studentController.AddStudent)
	studentGroup.GET("/:id", studentController.GetStudent)
	studentGroup.PUT("", studentController.UpdateStudent)
	studentGroup.DELETE("/:id", studentController.DeleteStudent)

	r.GET("/JoinRaftCluster", studentController.JoinRaftCluster)

	r.GET("/LeaderHandleCommand", studentController.LeaderHandleCommand)

	r.GET("/GetLeaderAddress", studentController.GetLeaderPortAddress)

	return r

}
