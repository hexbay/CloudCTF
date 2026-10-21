package services

import (
	"testing"

	"cloudctf/backend/internal/bootstrap"
	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

func TestGenericCRUDCreateReturnsConcreteType(t *testing.T) {
	db, err := bootstrap.OpenDB("sqlite://:memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := bootstrap.PrepareSchema(db); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}

	crud := NewGenericCRUD[models.Role](db)
	created, err := crud.Create(models.Role{
		Name:        "generic-role",
		Description: "created by generic crud",
	})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}

	if created.ID == 0 {
		t.Fatalf("expected created role id, got %+v", created)
	}
	if created.Name != "generic-role" {
		t.Fatalf("expected concrete role type back, got %+v", created)
	}
}

func TestGenericCRUDRoundTrip(t *testing.T) {
	db, err := bootstrap.OpenDB("sqlite://:memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := bootstrap.PrepareSchema(db); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}

	crud := NewGenericCRUD[models.Role](db)
	created, err := crud.Create(models.Role{
		Name:        "roundtrip-role",
		Description: "before update",
	})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}

	fetched, err := crud.GetByID(created.ID)
	if err != nil {
		t.Fatalf("get role: %v", err)
	}
	if fetched.Name != created.Name {
		t.Fatalf("expected fetched role name %q, got %q", created.Name, fetched.Name)
	}

	updated, err := crud.Update(created.ID, map[string]any{"description": "after update"})
	if err != nil {
		t.Fatalf("update role: %v", err)
	}
	if updated.Description != "after update" {
		t.Fatalf("expected updated description, got %+v", updated)
	}

	items, err := crud.List(0, 10, func(db *gorm.DB) *gorm.DB { return db.Order("id asc") })
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected listed items")
	}

	if err := crud.Delete(created.ID); err != nil {
		t.Fatalf("delete role: %v", err)
	}
	if _, err := crud.GetByID(created.ID); err == nil {
		t.Fatalf("expected deleted role lookup to fail")
	}
}
