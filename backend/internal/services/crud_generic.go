package services

import (
	"fmt"

	"gorm.io/gorm"
)

type GenericCRUD[T any] struct {
	db *gorm.DB
}

func NewGenericCRUD[T any](db *gorm.DB) *GenericCRUD[T] {
	return &GenericCRUD[T]{db: db}
}

func (c *GenericCRUD[T]) Create(input T) (T, error) {
	var zero T
	if c == nil || c.db == nil {
		return zero, fmt.Errorf("db unavailable")
	}
	if err := c.db.Create(&input).Error; err != nil {
		return zero, err
	}
	return input, nil
}

func (c *GenericCRUD[T]) GetByID(id any) (T, error) {
	var item T
	if c == nil || c.db == nil {
		return item, fmt.Errorf("db unavailable")
	}
	if err := c.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return item, err
	}
	return item, nil
}

func (c *GenericCRUD[T]) List(offset, limit int, scopes ...func(*gorm.DB) *gorm.DB) ([]T, error) {
	items := make([]T, 0)
	if c == nil || c.db == nil {
		return items, fmt.Errorf("db unavailable")
	}

	db := c.db.Model(new(T))
	for _, scope := range scopes {
		if scope != nil {
			db = scope(db)
		}
	}
	if offset > 0 {
		db = db.Offset(offset)
	}
	if limit > 0 {
		db = db.Limit(limit)
	}
	if err := db.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (c *GenericCRUD[T]) Update(id any, patch map[string]any) (T, error) {
	var item T
	if c == nil || c.db == nil {
		return item, fmt.Errorf("db unavailable")
	}
	if err := c.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return item, err
	}
	if err := c.db.Model(&item).Updates(patch).Error; err != nil {
		return item, err
	}
	if err := c.db.First(&item, fmt.Sprint(id)).Error; err != nil {
		return item, err
	}
	return item, nil
}

func (c *GenericCRUD[T]) Delete(id any) error {
	if c == nil || c.db == nil {
		return fmt.Errorf("db unavailable")
	}
	var item T
	result := c.db.Delete(&item, fmt.Sprint(id))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
