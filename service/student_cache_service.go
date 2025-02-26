package service

import (
	"fmt"
	"log"
	"memoryDataBase/dao"
	"memoryDataBase/model"
	"strings"
)

// StudentCacheService 定义缓存服务层结构体
type StudentCacheService struct {
	cacheDao *dao.StudentCacheDao
}

// NewStudentCacheService 创建一个新的 StudentCacheService 实例
func NewStudentCacheService(cacheDao *dao.StudentCacheDao) *StudentCacheService {
	return &StudentCacheService{
		cacheDao: cacheDao,
	}
}

// StudentExists 判断学生是否存在
func (scs *StudentCacheService) StudentExists(id string) error {
	_, err := scs.cacheDao.GetStudent(id)
	studentNotFoundErrMsg := fmt.Sprintf("缓存中不存在学生：%s", id)
	if err != nil {
		if strings.Contains(err.Error(), studentNotFoundErrMsg) {
			log.Printf("缓存中不存在学生：%s", id)
			return err
		}
		return err
	}
	return nil
}

// AddStudent 向缓存添加学生
func (scs *StudentCacheService) AddStudent(student *model.Student) error {
	//调用数据层代码
	if err := scs.cacheDao.AddStudent(student); err != nil {
		return fmt.Errorf("StudentCacheService.AddStudent 向缓存添加学生：%s失败：%w", student.ID, err)
	}
	log.Printf("向缓存添加学生学生：%s", student.ID)
	return nil
}

// GetStudentFromCache 从缓存中获取学生
func (scs *StudentCacheService) GetStudentFromCache(id string) (*model.Student, error) {
	//调用数据层代码
	student, err := scs.cacheDao.GetStudent(id)
	if err != nil {
		return nil, fmt.Errorf("StudentCacheService.GetStudentFromCache 从缓存中查找学生：%s失败：%w", id, err)
	}
	log.Printf("从缓存查找学生：%s", id)
	return student, nil
}

// UpdateStudent 更新学生信息
func (scs *StudentCacheService) UpdateStudent(student *model.Student) error {
	// 先判断是否存在
	err := scs.StudentExists(student.ID)
	if err != nil {
		return err
	}
	// 用新的学生信息覆盖缓存中的学生信息 （有就覆盖 没有就用旧的学生信息）
	s, _ := scs.cacheDao.GetStudent(student.ID)
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
	if err = scs.cacheDao.AddStudent(student); err != nil {
		return fmt.Errorf("StudentCacheService.UpdateStudent 向缓存中更新学生：%s失败：%w", student.ID, err)
	}
	log.Printf("向缓存中更新学生：%s", student.ID)
	return nil
}

// DeleteStudent 删除学生
func (scs *StudentCacheService) DeleteStudent(id string) error {
	// 先判断是否存在
	err := scs.StudentExists(id)
	if err != nil {
		return err
	}
	// 调用数据层代码
	if err = scs.cacheDao.DeleteStudent(id); err != nil {
		return fmt.Errorf("StudentCacheService.DeleteStudent 从缓存删除学生：%s失败：%w", id, err)
	}
	log.Printf("从缓存删除学生：%s", id)
	return nil
}

// ReLoadCacheData 重新加载缓存数据
func (scs *StudentCacheService) ReLoadCacheData(students []*model.Student) error {
	return scs.cacheDao.ReLoadCacheData(students)
}

// GetAllStudentsFromCache 从缓存中获取所有学生
func (scs *StudentCacheService) GetAllStudentsFromCache() ([]*model.Student, error) {
	return scs.cacheDao.GetAllStudents()
}
