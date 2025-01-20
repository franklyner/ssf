package wbcr

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/franklyner/ssf/server"
	"gorm.io/gorm"
)

var (
	ErrNotFound = server.JSONErrorResponse{
		Code:       http.StatusNotFound,
		Message:    "Entity not found",
		LogMessage: "Entity not found",
	}

	ErrAlreadyExists = server.JSONErrorResponse{
		Code:       http.StatusConflict,
		Message:    "A value with the given key already exists",
		LogMessage: "A value with the given key already exists",
	}
)

type Keyer[K comparable, V any] interface {
	SetKey(key K)
	GetKey() K
	GetValue() V
	*V // non-interface type constraint
}

type WBCacheRepository[K comparable, V any, PT Keyer[K, V]] struct {
	repo          map[K]V
	persister     Persister[K, V, PT]
	hasFetchedAll bool
}

type Persister[K comparable, V any, PT Keyer[K, V]] interface {
	Create(ctx *server.Context, value PT) (V, error)
	Update(ctx *server.Context, value PT) (V, error)
	Get(ctx *server.Context, key K) (V, error)
	GetAll(ctx *server.Context) ([]PT, error)
	Delete(ctx *server.Context, key K) error
}

type ExtendablePersister[K comparable, V any, PT Keyer[K, V]] struct {
	IntCreate func(*server.Context, K, PT) error
	IntUpdate func(*server.Context, K, PT) (V, error)
	IntGet    func(ctx *server.Context, key K) (V, error)
	IntGetAll func(ctx *server.Context) ([]PT, error)
	IntDelete func(ctx *server.Context, key K) error
}

func CreateWBCacheRepository[K comparable, V any, PT Keyer[K, V]](persister Persister[K, V, PT]) *WBCacheRepository[K, V, PT] {
	return &WBCacheRepository[K, V, PT]{
		repo:      make(map[K]V),
		persister: persister,
	}
}

func (wbcr *WBCacheRepository[K, V, PT]) GetAllKnownKeys(ctx *server.Context, ensureAllLoaded bool) (map[K]bool, error) {
	if ensureAllLoaded {
		err := wbcr.ensureLoaded(ctx)
		if err != nil {
			return nil, err
		}
	}
	keys := make(map[K]bool)
	for k := range wbcr.repo {
		keys[k] = true
	}
	return keys, nil
}

func (wbcr *WBCacheRepository[K, V, PT]) GetByKey(ctx *server.Context, key K) (V, error) {
	v, found := wbcr.repo[key]
	var err error
	if !found {
		e := new(V)
		if wbcr.hasFetchedAll {
			return *e, ErrNotFound
		}
		v, err = wbcr.persister.Get(ctx, key)
		if err != nil {
			return *e, err
		}
		wbcr.repo[key] = v
	}
	return v, nil
}

func (wbcr *WBCacheRepository[K, V, PT]) Insert(ctx *server.Context, value PT) error {
	key := value.GetKey()
	_, found := wbcr.repo[key]
	if found {
		return ErrAlreadyExists
	}

	v, err := wbcr.persister.Create(ctx, value)
	if err != nil {
		fmt.Errorf("error persisting (key: %v): %w", key, err)
	}
	wbcr.repo[key] = v
	return nil
}

func (wbcr *WBCacheRepository[K, V, PT]) Update(ctx *server.Context, value PT) error {
	key := value.GetKey()
	v, err := wbcr.persister.Update(ctx, value)
	if err != nil {
		return fmt.Errorf("error updating %v with persister: %w", key, err)
	}
	wbcr.repo[key] = v
	return nil
}

func (wbcr *WBCacheRepository[K, V, PT]) Delete(ctx *server.Context, key K) error {
	_, err := wbcr.persister.Get(ctx, key)
	found := true
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			found = false
		} else {
			return fmt.Errorf("error getting %v for deletion with persister: %w", key, err)
		}
	}
	if found {
		err = wbcr.persister.Delete(ctx, key)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return fmt.Errorf("error deleting %v with persister: %w", key, err)
		}
	}
	delete(wbcr.repo, key)
	return nil
}

func (wbcr *WBCacheRepository[K, V, PT]) Search(ctx *server.Context, searcher func(V) bool, ensureLoaded bool) ([]V, error) {
	if ensureLoaded {
		err := wbcr.ensureLoaded(ctx)
		if err != nil {
			return nil, err
		}
	}
	ret := []V{}
	for _, v := range wbcr.repo {
		if searcher(v) {
			ret = append(ret, v)
		}
	}

	return ret, nil
}

func (wbcr *WBCacheRepository[K, V, PT]) ensureLoaded(ctx *server.Context) error {
	if !wbcr.hasFetchedAll {
		vals, err := wbcr.persister.GetAll(ctx)
		if err != nil {
			return fmt.Errorf("error fetching all values from persister: %w", err)
		}
		for _, v := range vals {
			wbcr.repo[v.GetKey()] = v.GetValue()
		}
		wbcr.hasFetchedAll = true
	}
	return nil
}

func (e *ExtendablePersister[K, V, PT]) GetAll(ctx *server.Context) ([]PT, error) {
	if e.IntGetAll == nil {
		return make([]PT, 0), nil
	}
	return e.IntGetAll(ctx)
}

func (e *ExtendablePersister[K, V, PT]) Create(ctx *server.Context, key K, value PT) error {
	if e.IntCreate == nil {
		return nil
	}
	return e.IntCreate(ctx, key, value)
}
func (e *ExtendablePersister[K, V, PT]) Update(ctx *server.Context, key K, value PT) (V, error) {
	if e.IntUpdate == nil {
		return value.GetValue(), nil
	}
	return e.IntUpdate(ctx, key, value)
}
func (e *ExtendablePersister[K, V, PT]) Get(ctx *server.Context, key K) (V, error) {
	if e.IntGet == nil {
		return *new(V), ErrNotFound
	}
	return e.IntGet(ctx, key)
}
func (e *ExtendablePersister[K, V, PT]) Delete(ctx *server.Context, key K) error {
	if e.IntDelete == nil {
		return nil
	}
	return e.IntDelete(ctx, key)
}

type GormPersister[K comparable, V any, PT Keyer[K, V]] struct {
	repository *server.Repository
}

func (p *GormPersister[K, V, PT]) SetRepository(repo *server.Repository) {
	p.repository = repo
}

func (p *GormPersister[K, V, PT]) RunMigration(repository *server.Repository) {
	p.SetRepository(repository)
	p.RunMigrationInternalRepo()
}

func (p *GormPersister[K, V, PT]) RunMigrationInternalRepo() {
	val := new(V)
	err := p.repository.DB.AutoMigrate(*val)
	if err != nil {
		panic(err) // fail fast as this happens on startup
	}
}

func (p *GormPersister[K, V, PT]) Create(ctx *server.Context, value PT) (V, error) {
	db := p.repository.DB
	res := db.Create(value)
	if res.Error != nil {
		e := new(V)
		if res.Error == gorm.ErrDuplicatedKey {
			return *e, ErrAlreadyExists
		}
		return *e, fmt.Errorf("error while inserting OU Team Mapping (%+v): %w", value, res.Error)
	}
	return value.GetValue(), nil
}
func (p *GormPersister[K, V, PT]) Update(ctx *server.Context, value PT) (V, error) {
	db := p.repository.DB
	empty := new(V)
	res := db.Save(&value) // Save is an upsert
	if res.Error != nil {
		return *empty, fmt.Errorf("error while inserting OU Team Mapping (%+v): %w", value, res.Error)
	}

	return value.GetValue(), nil
}
func (p *GormPersister[K, V, PT]) Get(ctx *server.Context, key K) (V, error) {
	db := p.repository.DB
	empty := new(V)
	value := new(PT)
	v := *value
	res := db.First(&v, key)

	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return *empty, ErrNotFound
		}
		return *empty, fmt.Errorf("couldn't find team mapping after update (%+v)", v)
	}
	return v.GetValue(), nil
}
func (p *GormPersister[K, V, PT]) GetAll(ctx *server.Context) ([]PT, error) {
	db := p.repository.DB
	all := []PT{}
	res := db.Find(&all)
	if res.Error != nil {
		return []PT{}, fmt.Errorf("error retrieving all instances: %w", res.Error)
	}
	return all, nil
}
func (p *GormPersister[K, V, PT]) Delete(ctx *server.Context, key K) error {
	db := p.repository.DB
	v := new(V)
	res := db.Delete(v, key)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("error deleting record with key %v: %w", key, res.Error)
		}
	}
	return nil
}
