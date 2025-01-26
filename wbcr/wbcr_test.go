package wbcr

import (
	"testing"

	"github.com/franklyner/ssf/server"
)

const dburi = "" // replace with real url. NEVER STORE TO GIT

type SomeMapper struct {
	SomeID    string `gorm:"primaryKey"`
	SomeValue string
}

func (sm *SomeMapper) SetKey(key string) {
	sm.SomeID = key
}
func (sm *SomeMapper) GetKey() string {
	return sm.SomeID
}
func (sm *SomeMapper) GetValue() SomeMapper {
	return *sm
}
func (sm *SomeMapper) GetKeyName() string {
	return "some_id"
}

func TestLifecycle(t *testing.T) {
	srv := server.BlankServer()
	ctx := srv.InitNonRequestContext()
	repo := server.CreateRepositoryFromParams(dburi, 1)
	p := GormPersister[string, SomeMapper, *SomeMapper]{}
	p.SetRepository(repo)
	p.RunMigrationInternalRepo()
	cache := CreateWBCacheRepository(&p)

	first := SomeMapper{
		SomeID:    "first",
		SomeValue: "haha",
	}
	err := cache.Insert(ctx, &first)
	if err != nil {
		t.Error(err)
	}
	fetched, err := cache.GetByKey(ctx, "first")
	if err != nil {
		t.Error(err)
	}
	if fetched.SomeID != first.SomeID {
		t.Error("ids don't match: ", fetched.SomeID, first.SomeID)
	}
	if fetched.SomeValue != first.SomeValue {
		t.Error("SomeValue don't match: ", fetched.SomeValue, first.SomeValue)
	}

	// re-initializing cache
	cache = CreateWBCacheRepository(&p)
	fetched, err = cache.GetByKey(ctx, "first")
	if err != nil {
		t.Error(err)
	}
	if fetched.SomeID != first.SomeID {
		t.Error("ids don't match: ", fetched.SomeID, first.SomeID)
	}
	if fetched.SomeValue != first.SomeValue {
		t.Error("SomeValue don't match: ", fetched.SomeValue, first.SomeValue)
	}

	second := SomeMapper{
		SomeID:    "second",
		SomeValue: "hoho",
	}
	err = cache.Insert(ctx, &second)
	if err != nil {
		t.Error(err)
	}

	second.SomeValue = "hihi"
	err = cache.Update(ctx, &second)
	if err != nil {
		t.Error(err)
	}

	fetched, err = cache.GetByKey(ctx, second.SomeID)
	if err != nil {
		t.Error(err)
	}
	if fetched.SomeValue != second.SomeValue {
		t.Error("updated value doesn't match: ", fetched.SomeValue, second.SomeValue)
	}

	// re-initializing cache
	cache = CreateWBCacheRepository(&p)
	keys, err := cache.GetAllKnownKeys(ctx, true)
	if err != nil {
		t.Error(err)
	}

	if len(keys) != 2 {
		t.Errorf("unexpected keys: %+v", keys)
	}

	fetched, err = cache.GetByKey(ctx, second.SomeID)
	if err != nil {
		t.Error(err)
	}

	err = cache.Delete(ctx, first.SomeID)
	if err != nil {
		t.Error(err)
	}
	err = cache.Delete(ctx, second.SomeID)
	if err != nil {
		t.Error(err)
	}
}
