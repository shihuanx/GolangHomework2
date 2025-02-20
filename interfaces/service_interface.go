package interfaces

import (
	"memoryDataBase/model"
)

// StudentServiceInterface 定义学生服务接口 解决fsm依赖service service依赖fsm导致的循环导入问题。。。
type StudentServiceInterface interface {
	AddStudentInternal(student *model.Student) error
	UpdateStudentInternal(student *model.Student) error
	DeleteStudentInternal(id string) error
	ReLoadCacheDataInternal()
	PeriodicDeleteInternal(examineSize int)
}
