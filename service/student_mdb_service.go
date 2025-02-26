package service

import (
	"errors"
	"fmt"
	"log"
	"node2/dao"
	"node2/model"
	"strings"
)

// StudentMdbService 定义内存数据库服务层结构体
type StudentMdbService struct {
	memoryDBDao *dao.MemoryDBDao
}

// NewStudentMdbService 创建一个新的 StudentMdbService 实例
func NewStudentMdbService(db *dao.MemoryDBDao) *StudentMdbService {
	return &StudentMdbService{
		memoryDBDao: db,
	}
}

// StudentExists 判断学生是否存在
func (smdbs *StudentMdbService) StudentExists(studentId string) error {
	_, err := smdbs.GetStudent(studentId)
	if err != nil {
		//保装错误信息
		if strings.Contains(err.Error(), "学生不存在") {
			return fmt.Errorf("不存在学生：%s", err)
		}
		return err
	}
	return nil
}

// AddStudent 向内存添加学生
func (smdbs *StudentMdbService) AddStudent(student *model.Student) {
	log.Printf("向内存添加学生：%s", student.ID)
	smdbs.memoryDBDao.Set(student.ID, student, student.Expiration)
}

// GetStudent 从内存中获取学生
func (smdbs *StudentMdbService) GetStudent(studentId string) (*model.Student, error) {
	value, exists := smdbs.memoryDBDao.Get(studentId)
	if exists {
		student, ok := value.(*model.Student)
		if !ok {
			return nil, errors.New("StudentMdbService.GetStudent 类型断言失败")
		}
		log.Printf("从内存中查找学生：%s", studentId)
		log.Printf("%v", student)
		return student, nil
	}
	return nil, fmt.Errorf("StudentMdbService.GetStudent 内存中不存在学生：%s", studentId)
}

// UpdateStudent 更新学生信息
func (smdbs *StudentMdbService) UpdateStudent(student *model.Student) error {
	// 先判断是否存在
	err := smdbs.StudentExists(student.ID)
	if err != nil {
		return fmt.Errorf("StudentMdbService.UpdateStudent 在内存更新学生：%s失败：%w", student.ID, err)
	}
	// 用新的学生信息更新原来的学生信息
	s, _ := smdbs.GetStudent(student.ID)
	for k, v := range student.Grades {
		s.Grades[k] = v
	}
	student.Grades = s.Grades

	if student.Name == "" {
		student.Name = s.Name
	}
	if student.Class == "" {
		student.Class = s.Class
	}
	if student.Gender == "" {
		student.Gender = s.Gender
	}

	// 调用数据层代码
	smdbs.memoryDBDao.Update(student.ID, student)
	log.Printf("在内存中更新学生：%s", student.ID)
	return nil
}

// DeleteStudent 删除学生
func (smdbs *StudentMdbService) DeleteStudent(studentId string) error {
	// 先判断是否存在
	if err := smdbs.StudentExists(studentId); err != nil {
		return fmt.Errorf("StudentMdbService.DeleteStudent 在内存删除学生：%s失败：%w", studentId, err)
	}
	// 调用数据层代码
	smdbs.memoryDBDao.Delete(studentId)
	log.Printf("在内存中删除学生：%s", studentId)
	return nil
}

// PeriodicDelete 定期删除内存中的过期键
func (smdbs *StudentMdbService) PeriodicDelete(examineSize int) {
	smdbs.memoryDBDao.PeriodicDelete(examineSize)
}
