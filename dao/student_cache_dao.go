package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"node2/model"
	"strconv"
)

// 定义缓存键的前缀
const studentCachePrefix = "student:"

// StudentCacheDao 定义缓存层结构体实例
type StudentCacheDao struct {
	client redis.Client
}

// NewStudentCacheDao 初始化缓存层结构体实例
func NewStudentCacheDao(client *redis.Client) *StudentCacheDao {
	return &StudentCacheDao{
		client: *client,
	}
}

// AddStudent 添加学生
func (d StudentCacheDao) AddStudent(student *model.Student) error {
	ctx := context.Background()

	key := studentCachePrefix + student.ID

	// 将成绩信息序列化为 JSON 字符串
	gradeJSON, err := json.Marshal(student.Grades)
	if err != nil {
		return fmt.Errorf("StudentRedisDao.AddStudent Marshal err: %v", err)
	}

	// 构建学生的字段信息
	fields := make(map[string]interface{})
	fields["id"] = student.ID
	fields["name"] = student.Name
	fields["gender"] = student.Gender
	fields["class"] = student.Class
	fields["grade"] = gradeJSON
	fields["expiration"] = student.Expiration

	// 添加键到哈希表里面
	if err = d.client.HSet(ctx, key, fields).Err(); err != nil {
		return fmt.Errorf("StudentRedisDao.AddStudent Hset err: %w", err)
	}
	return nil
}

// GetStudent 获取学生
func (d StudentCacheDao) GetStudent(id string) (*model.Student, error) {
	ctx := context.Background()
	key := studentCachePrefix + id

	// 从缓存中获取学生的所有字段信息
	result, err := d.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("StudentRedisDao.GetStudent HGetAll err: %w", err)
	}

	// 如果结果为空，说明学生信息不存在
	if len(result) == 0 {
		return nil, fmt.Errorf("StudentRedisDao.GetStudent 缓存中不存在学生：%s", id)
	}

	// 创建一个新的 Student 对象
	student := &model.Student{}

	// 填充基本信息
	student.ID = result["id"]
	student.Name = result["name"]
	student.Gender = result["gender"]
	student.Class = result["class"]
	student.Expiration, err = strconv.ParseInt(result["expiration"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("StudentRedisDao.GetStudent ParseInt err：%w", err)
	}

	// 反序列化成绩信息
	gradeJSON := []byte(result["grade"])
	grades := make(map[string]float64)
	err = json.Unmarshal(gradeJSON, &grades)
	if err != nil {
		return nil, fmt.Errorf("StudentRedisDao.GetStudent Unmarshal err：%w", err)
	}
	student.Grades = grades

	return student, nil
}

// DeleteStudent 删除学生
func (d StudentCacheDao) DeleteStudent(id string) error {
	ctx := context.Background()

	key := studentCachePrefix + id

	// 使用 Del 命令删除缓存数据
	err := d.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("StudentRedisDao.DeleteStudent Del err: %w", err)
	}
	return nil
}

func (d StudentCacheDao) ReLoadCacheData(students []*model.Student) error {
	ctx := context.Background()
	// 删除所有 Redis 记录
	err := d.client.FlushDB(ctx).Err()
	if err != nil {
		return fmt.Errorf("StudentRedisDao.ReLoadCacheData FlushDB err: %w", err)
	}
	// 重新添加学生数据
	for _, student := range students {
		err = d.AddStudent(student)
		if err != nil {
			return fmt.Errorf("StudentRedisDao.ReLoadCacheData 加载学生时出错：%w", err)
		}
	}
	return nil
}

// GetAllStudents 获取所有学生
func (d StudentCacheDao) GetAllStudents() ([]*model.Student, error) {
	ctx := context.Background()
	var students []*model.Student

	// 使用 KEYS 命令获取所有学生的缓存键
	keys, err := d.client.Keys(ctx, "student:*").Result()
	if err != nil {
		return nil, fmt.Errorf("StudentRedisDao.GetAllStudents Keys student:* err: %w", err)
	}

	// 遍历所有学生的缓存键，先得到id，再通过id获取学生信息
	for _, key := range keys {
		studentId, err := d.client.HGet(ctx, key, "id").Result()
		if err != nil {
			return nil, fmt.Errorf("StudentRedisDao.GetAllStudents HGet err: %w", err)
		}
		student, err := d.GetStudent(studentId)
		if err != nil {
			return nil, fmt.Errorf("StudentRedisDao.GetAllStudents 通过学生id：%s获得学生时失败：%w", studentId, err)
		}
		students = append(students, student)
	}
	return students, nil
}
