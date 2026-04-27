package db_test

import (
	"sync"
	"testing"

	"github.com/wheelibin/qrypad/internal/db"
)

func TestSchemaCache_TableInfo_MissAndHit(t *testing.T) {
	c := db.NewSchemaCache()
	ref := db.TableReference{Schema: "public", Name: "users"}

	_, ok := c.GetTableInfo(ref, "cols")
	if ok {
		t.Fatal("expected cache miss, got hit")
	}

	data := &db.Data{Columns: []string{"id", "name"}, Rows: nil}
	c.SetTableInfo(ref, "cols", data)

	got, ok := c.GetTableInfo(ref, "cols")
	if !ok {
		t.Fatal("expected cache hit, got miss")
	}
	if got != data {
		t.Fatal("expected same pointer")
	}
}

func TestSchemaCache_Entities_MissAndHit(t *testing.T) {
	c := db.NewSchemaCache()

	_, ok := c.GetEntities("tables")
	if ok {
		t.Fatal("expected cache miss, got hit")
	}

	data := &db.Data{Columns: []string{"name"}, Rows: nil}
	c.SetEntities("tables", data)

	got, ok := c.GetEntities("tables")
	if !ok {
		t.Fatal("expected cache hit, got miss")
	}
	if got != data {
		t.Fatal("expected same pointer")
	}
}

func TestSchemaCache_Invalidate(t *testing.T) {
	c := db.NewSchemaCache()
	ref := db.TableReference{Name: "orders"}
	data := &db.Data{Columns: []string{"id"}, Rows: nil}

	c.SetTableInfo(ref, "cols", data)
	c.SetEntities("tables", data)

	c.Invalidate()

	if _, ok := c.GetTableInfo(ref, "cols"); ok {
		t.Fatal("expected cache miss after invalidate (tableInfo)")
	}
	if _, ok := c.GetEntities("tables"); ok {
		t.Fatal("expected cache miss after invalidate (entities)")
	}
}

func TestSchemaCache_DifferentKindsStoredSeparately(t *testing.T) {
	c := db.NewSchemaCache()
	ref := db.TableReference{Name: "products"}
	cols := &db.Data{Columns: []string{"name"}}
	inds := &db.Data{Columns: []string{"index_name"}}

	c.SetTableInfo(ref, "cols", cols)
	c.SetTableInfo(ref, "inds", inds)

	gotCols, _ := c.GetTableInfo(ref, "cols")
	gotInds, _ := c.GetTableInfo(ref, "inds")

	if gotCols != cols {
		t.Fatal("cols pointer mismatch")
	}
	if gotInds != inds {
		t.Fatal("inds pointer mismatch")
	}
}

func TestSchemaCache_ConcurrentAccess(_ *testing.T) {
	c := db.NewSchemaCache()
	ref := db.TableReference{Name: "events"}
	data := &db.Data{Columns: []string{"id"}}

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(3)
		go func() {
			defer wg.Done()
			c.SetTableInfo(ref, "cols", data)
		}()
		go func() {
			defer wg.Done()
			c.GetTableInfo(ref, "cols")
		}()
		go func() {
			defer wg.Done()
			c.Invalidate()
		}()
	}
	wg.Wait()
}
