package model

// Student 定义学生结构体
type Student struct {
	ID         string             `json:"id" validate:"required"`
	Name       string             `json:"name" validate:"required"`
	Gender     string             `json:"gender" validate:"required"`
	Class      string             `json:"class" validate:"required"`
	Grades     map[string]float64 `json:"grades"`
	Expiration int64              `json:"expiration"`
}

// StudentDB 关联mysql的学生表
type StudentDB struct {
	ID     string `json:"id" validate:"required" gorm:"primaryKey"`
	Name   string `json:"name" validate:"required"`
	Gender string `json:"gender" validate:"required"`
	Class  string `json:"class" validate:"required"`
}

// Grade 关联mysql的成绩表
type Grade struct {
	ID        string  `json:"id" validate:"required"`
	Subject   string  `json:"subject" validate:"required"`
	Score     float64 `json:"score" validate:"required"`
	StudentId string  `json:"student_id" validate:"required"`
}

// StudentCount 关联mysql的访问次数表
type StudentCount struct {
	ID        string `json:"id" validate:"required"`
	StudentId string `json:"student_id" validate:"required"`
	Count     int32  `json:"count" validate:"required"`
}
