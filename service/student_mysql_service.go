package service

import (
	"fmt"
	"gorm.io/gorm"
	"log"
	"memoryDataBase/dao"
	"memoryDataBase/model"
	"strings"
)

// StudentMysqlService 定义mysql数据库服务层结构体
type StudentMysqlService struct {
	mysqlDao *dao.StudentMysqlDao
}

// NewStudentMysqlService 创建一个新的 StudentMysqlService 实例
func NewStudentMysqlService(mysqlDao *dao.StudentMysqlDao) *StudentMysqlService {
	return &StudentMysqlService{
		mysqlDao: mysqlDao,
	}
}

// ConvertToStudent 把MySQL数据库中的学生转化为model中的学生
func (sms *StudentMysqlService) ConvertToStudent(studentDB *model.StudentDB) (*model.Student, error) {
	// 获取学生的成绩
	grades := make(map[string]float64)
	result, err := sms.mysqlDao.GetGrade(studentDB.ID)
	if err != nil {
		return nil, fmt.Errorf("StudentMysqlService.ConvertToStudent 获取学生：%s成绩失败：%v", studentDB.ID, err)
	}
	// 将成绩添加到学生中
	for _, v := range result {
		grades[v.Subject] = v.Score
	}
	return &model.Student{
		ID:     studentDB.ID,
		Name:   studentDB.Name,
		Gender: studentDB.Gender,
		Class:  studentDB.Class,
		Grades: grades,
	}, nil
}

// StudentExists 判断学生是否存在
func (sms *StudentMysqlService) StudentExists(id string) error {
	_, err := sms.mysqlDao.GetStudent(id)
	if err != nil {
		if strings.Contains(err.Error(), "数据库不存在学生") {
			return err
		}
		return err
	}
	return nil
}

// StudentCountNotExists 判断学生记录是否存在 只有确定不存在才返回true
func (sms *StudentMysqlService) StudentCountNotExists(id string) bool {
	_, err := sms.mysqlDao.GetStudentCount(id)
	if err != nil {
		if strings.Contains(err.Error(), "不存在学生记录") {
			log.Printf("不存在学生记录：%s", id)
			return true
		}
		return false
	}
	return false
}

// AddStudentToMysql 向数据库添加学生
func (sms *StudentMysqlService) AddStudentToMysql(tx *gorm.DB, student *model.Student) error {
	// 开启事务 在事务中添加学生信息 调用服务层代码
	if err := sms.mysqlDao.AddStudentToMysql(tx, student); err != nil {
		tx.Rollback()
		return fmt.Errorf("StudentMysqlService.AddStudentToMysql 向学生表添加学生：%s失败：%w", student.ID, err)
	}
	log.Printf("向数据库添加学生：%s", student.ID)

	// 在事务中添加学生成绩信息
	for k, v := range student.Grades {
		if err := sms.mysqlDao.AddGradeToMysql(tx, k, v, student.ID); err != nil {
			tx.Rollback()
			return fmt.Errorf("StudentMysqlService.AddStudentToMysql 向成绩表添加学生：%s的成绩失败：%w", student.ID, err)
		}
	}
	log.Printf("向数据库添加学生的成绩：%s", student.ID)
	return nil
}

// GetStudentFromMysql 从数据库中获取学生
func (sms *StudentMysqlService) GetStudentFromMysql(studentId string) (*model.Student, error) {
	var studentDB *model.StudentDB
	var student *model.Student
	// 调用数据层代码 获取数据库中的学生
	studentDB, err := sms.mysqlDao.GetStudent(studentId)
	if err != nil {
		return nil, fmt.Errorf("StudentMysqlService.GetStudentFromMysql 从数据库查找学生：%s失败：%w", studentId, err)
	}
	log.Printf("从数据库查找学生：%s", studentId)
	// 将数据库中的学生转化为model中的学生
	student, err = sms.ConvertToStudent(studentDB)
	if err != nil {
		return nil, fmt.Errorf("StudentMysqlService.GetStudentFromMysql 数据库中学生：%s转化出错：%w", studentId, err)
	}
	log.Printf("从数据库获取学生：%s", student.ID)
	return student, nil
}

// UpdateStudent 从数据库中获取所有学生
func (sms *StudentMysqlService) UpdateStudent(tx *gorm.DB, student *model.Student) error {
	// 先判断是否存在
	err := sms.StudentExists(student.ID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("StudentMysqlService.UpdateStudent 更新学生：%s失败：%w", student.ID, err)
	}

	// 调用数据层代码 更新学生信息
	if err = sms.mysqlDao.UpdateStudent(tx, student); err != nil {
		tx.Rollback()
		return fmt.Errorf("StudentMysqlService.UpdateStudent 在数据库更新学生：%s失败：%w", student.ID, err)
	}
	log.Printf("在数据库更新学生：%s", student.ID)
	//向成绩表插入数据 先判断是否存在 如果存在就更新 不存在就添加
	if student.Grades != nil {
		for subject, grade := range student.Grades {
			exists, err := sms.mysqlDao.GetGradeBySubject(student.ID, subject)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("StudentMysqlService.UpdateStudent 通过学科：%s查找学生：%s的记录失败：%w", student.ID, subject, err)
			}
			if exists != nil {
				//成绩记录已存在 修改
				if err = sms.mysqlDao.UpdateGrade(tx, subject, grade, student.ID); err != nil {
					tx.Rollback()
					return fmt.Errorf("StudentMysqlService.UpdateStudent 向成绩表添加成绩失败，学生：%s，错误：%w", student.ID, err)
				}
			} else {
				//成绩记录不存在 插入
				if err = sms.mysqlDao.AddGradeToMysql(tx, subject, grade, student.ID); err != nil {
					tx.Rollback()
					return fmt.Errorf("StudentMysqlService.UpdateStudent 向成绩表添加学生：%s的成绩：%s失败：%w", student.ID, subject, err)
				}
			}
		}
	}
	log.Printf("在数据库更新学生：%s的成绩", student.ID)
	return nil
}

// DeleteStudent 删除学生
func (sms *StudentMysqlService) DeleteStudent(tx *gorm.DB, id string) error {
	// 先判断是否存在
	if err := sms.StudentExists(id); err != nil {
		tx.Rollback()
		return fmt.Errorf("StudentMysqlService.DeleteStudent 删除学生：%s失败：%w", id, err)
	}

	// 调用数据层代码 删除学生
	if err := sms.mysqlDao.DeleteStudent(tx, id); err != nil {
		tx.Rollback()
		return fmt.Errorf("StudentMysqlService.DeleteStudent 删除学生：%s失败：%w", id, err)
	}
	log.Printf("删除学生：%s", id)

	// 调用数据层代码 删除学生的成绩
	if err := sms.mysqlDao.DeleteScore(tx, id); err != nil {
		tx.Rollback()
		return fmt.Errorf("StudentMysqlService.DeleteStudent 删除学生：%s的成绩时失败：%w", id, err)
	}
	log.Printf("从数据库删除学生：%s的成绩", id)
	return nil
}

// GetHotStudentsFromMysql 获取访问次数最高的学生
func (sms *StudentMysqlService) GetHotStudentsFromMysql() ([]*model.Student, error) {
	var hotStudents []*model.StudentCount
	var students []*model.Student
	// 调用数据层代码 获取访问次数最高的学生
	hotStudents, err := sms.GetHotStudentCount()
	if err != nil {
		return nil, err
	}
	// 将数据库中的学生记录转化为model中的学生
	for _, studentRecord := range hotStudents {
		student, err := sms.GetStudentFromMysql(studentRecord.StudentId)
		if err != nil {
			return nil, fmt.Errorf("StudentMysqlService.GetHotStudentFromMysql 从数据库转化学生：%s失败：%w", studentRecord.StudentId, err)
		}
		students = append(students, student)
	}
	return students, nil
}

// AddStudentCount 添加学生访问次数
func (sms *StudentMysqlService) AddStudentCount(id string) {
	// 先判断是否存在
	// 调用数据层代码 如此不存在就添加 存在就更新
	record, err := sms.GetStudentCountFromMysql(id)
	if err != nil {
		if strings.Contains(err.Error(), "不存在学生记录") {
			if err = sms.mysqlDao.AddStudentCount(id); err != nil {
				log.Printf("添加学生：%s的记录时出错：%v", id, err)
			} else {
				log.Printf("添加学生：%s记录", id)
			}
		} else {
			log.Printf("添加学生记录：%s时失败：%v", id, err)
		}
	} else {
		record.Count++
		if err = sms.mysqlDao.UpdateStudentCount(record); err != nil {
			log.Printf("更新学生：%s访问次数时出错：%v", id, err)
		} else {
			log.Printf("更新学生：%s的访问次数：%s", id, record.StudentId)
		}
	}
}

// GetStudentCountFromMysql 获取学生访问次数
func (sms *StudentMysqlService) GetStudentCountFromMysql(id string) (*model.StudentCount, error) {
	// 调用数据层代码 获取学生访问次数
	return sms.mysqlDao.GetStudentCount(id)
}

// GetHotStudentCount 获取访问次数最高的学生
func (sms *StudentMysqlService) GetHotStudentCount() ([]*model.StudentCount, error) {
	// 调用数据层代码 获取访问次数最高的学生
	studentCounts, err := sms.mysqlDao.GetHotStudentCounts()
	if err != nil {
		return nil, fmt.Errorf("StudentMysqlService.GetHotStudentCount 获取访问最高的学生记录出错：%w", err)
	}
	return studentCounts, nil
}

// DeleteStudentCount 删除学生访问次数
func (sms *StudentMysqlService) DeleteStudentCount(id string) {
	// 先判断是否存在 存在就删除 不存在就不管了
	if !sms.StudentCountNotExists(id) {
		if err := sms.mysqlDao.DeleteStudentCount(id); err != nil {
			log.Printf("删除学生：%s记录失败：%v", id, err)
		}
	}
}
