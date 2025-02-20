package dao

import (
	"fmt"
	"gorm.io/gorm"
	"memoryDataBase/model"
)

// StudentMysqlDao 定义mysql层结构体实例
type StudentMysqlDao struct {
	DB *gorm.DB
}

// NewStudentMysqlDao 初始化mysql层结构体实例
func NewStudentMysqlDao(db *gorm.DB) *StudentMysqlDao {
	return &StudentMysqlDao{
		DB: db,
	}
}

// GetStudent 查找学生
func (d *StudentMysqlDao) GetStudent(id string) (*model.StudentDB, error) {
	var studentDB model.StudentDB
	result := d.DB.Raw("select * from student where id = ?", id).Scan(&studentDB)
	if result.Error != nil {
		return nil, fmt.Errorf("StudentMysqlDao.GetStudent err:%v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("数据库不存在学生：%s", id)
	}
	return &studentDB, nil
}

// AddStudentToMysql 添加学生（不包含成绩）
func (d *StudentMysqlDao) AddStudentToMysql(tx *gorm.DB, student *model.Student) error {
	err := tx.Exec("insert into student (id,name,gender,class,expiration) values (?,?,?,?,?)",
		student.ID, student.Name, student.Gender, student.Class, student.Expiration).Error
	if err != nil {
		return fmt.Errorf("StudentMysqlDao.AddStudentToMysql err:%w", err)
	}
	return nil
}

// AddGradeToMysql 添加成绩
func (d *StudentMysqlDao) AddGradeToMysql(tx *gorm.DB, subject string, score float64, id string) error {
	err := tx.Exec("insert into grade (subject, score, student_id) VALUES (?,?,?)",
		subject, score, id).Error
	if err != nil {
		return fmt.Errorf("StudentMysqlDao.AddGradeToMysql err:%w", err)
	}
	return nil
}

// GetGrade 获取成绩
func (d *StudentMysqlDao) GetGrade(studentId string) ([]model.Grade, error) {
	var grades []model.Grade
	err := d.DB.Raw("select * from grade where student_id = ?", studentId).Scan(&grades).Error
	if err != nil {
		return nil, fmt.Errorf("StudentMysqlDao.GetGrade err:%w", err)
	}
	return grades, nil
}

// UpdateStudent 更新学生信息
func (d *StudentMysqlDao) UpdateStudent(tx *gorm.DB, student *model.Student) error {
	sqlStmt := `
        UPDATE student
        SET
            name = IF(COALESCE(?, '') != '', ?, name),
            gender = IF(COALESCE(?, '') != '', ?, gender),
            class = IF(COALESCE(?, '') != '', ?, class)
        WHERE id = ?
    `

	err := tx.Exec(sqlStmt,
		student.Name, student.Name,
		student.Gender, student.Gender,
		student.Class, student.Class,
		student.ID).Error
	if err != nil {
		return fmt.Errorf("StudentMysqlDao.UpdateStudent err:%w", err)
	}
	return nil
}

// UpdateGrade 更新成绩
func (d *StudentMysqlDao) UpdateGrade(tx *gorm.DB, subject string, score float64, studentId string) error {
	err := tx.Exec("update grade set score=? where id=? and subject=?", score, studentId, subject).Error
	if err != nil {
		return fmt.Errorf("StudentMysqlDao.UpdateGrade err:%w", err)
	}
	return nil
}

// DeleteStudent 删除学生
func (d *StudentMysqlDao) DeleteStudent(tx *gorm.DB, id string) error {
	err := tx.Exec("delete from student where id = ?", id).Error
	if err != nil {
		return fmt.Errorf("StudentMysqlDao.DeleteStudent err:%w", err)
	}
	return nil
}

// DeleteScore 删除成绩
func (d *StudentMysqlDao) DeleteScore(tx *gorm.DB, id string) error {
	err := tx.Exec("delete from grade where student_id = ?", id).Error
	if err != nil {
		return fmt.Errorf("StudentMysqlDao.DeleteScore err:%w", err)
	}
	return nil
}

// GetGradeBySubject 通过学科和学生id获取成绩记录
func (d *StudentMysqlDao) GetGradeBySubject(id string, subject string) (*model.Grade, error) {
	var grade *model.Grade
	err := d.DB.Raw("select * from grade where subject = ? and student_id = ?", subject, id).Scan(&grade).Error
	if err != nil {
		return nil, fmt.Errorf("StudentMysqlDao.GetGradeBySubject err:%w", err)
	}
	return grade, nil
}

// GetAllStudents 获取所有学生
func (d *StudentMysqlDao) GetAllStudents() ([]model.StudentDB, error) {
	var studentDBs []model.StudentDB
	err := d.DB.Raw("select * from student").Scan(&studentDBs).Error
	if err != nil {
		return nil, fmt.Errorf("StudentMysqlDao.GetAllStudents err:%w", err)
	}
	return studentDBs, nil
}

// GetStudentCount 获取学生访问次数
func (d *StudentMysqlDao) GetStudentCount(id string) (*model.StudentCount, error) {
	var count model.StudentCount
	result := d.DB.Raw("select * from student_count where student_id = ?", id).Scan(&count)
	if result.Error != nil {
		return nil, fmt.Errorf("StudentMysqlDao.GetStudentCount err:%v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("数据库不存在学生记录：%s", id)
	}
	return &count, nil
}

// AddStudentCount 新增学生访问次数
func (d *StudentMysqlDao) AddStudentCount(id string) error {
	err := d.DB.Exec("insert into student_count (student_id) values (?)", id).Error
	if err != nil {
		return fmt.Errorf("StudentMysqlDao.AddStudentCount err:%w", err)
	}
	return nil
}

// UpdateStudentCount 更新学生访问次数
func (d *StudentMysqlDao) UpdateStudentCount(record *model.StudentCount) error {
	err := d.DB.Exec("update student_count set count=? where student_id = ?", record.Count, record.StudentId).Error
	if err != nil {
		return fmt.Errorf("StudentMysqlDao.UpdateStudentCount err:%w", err)
	}
	return nil
}

// DeleteStudentCount 删除学生访问次数
func (d *StudentMysqlDao) DeleteStudentCount(id string) error {
	err := d.DB.Exec("delete from student_count where student_id = ?", id).Error
	if err != nil {
		return fmt.Errorf("StudentMysqlDao.DeleteStudentCount err:%w", err)
	}
	return nil
}

// GetHotStudentCounts 获取访问次数前10的学生
func (d *StudentMysqlDao) GetHotStudentCounts() ([]*model.StudentCount, error) {
	var counts []*model.StudentCount
	err := d.DB.Raw("select * from student_count order by count desc limit 10").Scan(&counts).Error
	if err != nil {
		return nil, fmt.Errorf("StudentMysqlDao.GetHotStudentCounts err:%w", err)
	}
	return counts, nil
}
