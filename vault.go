package vault

import (
	"errors"
	"sync"
)

var (
	NotFoundError  = errors.New("key not found")
	KeyExistsError = errors.New("key exists")
)

type Keyer interface {
	Key() string
}

type Persister interface {
	Persist(map[string]Keyer) error
	Load() (map[string]Keyer, error)
}

type FilterFunc func(Keyer) bool

type Vault struct {
	mutex sync.Mutex
	vault map[string]Keyer

	persisters []Persister
}

func New() *Vault {
	return &Vault{
		vault:  make(map[string]Keyer),
		persisters: make([]Persister, 0),
	}
}

func (v *Vault) Register(s Persister) {
	v.persisters = append(v.persisters, s)
}

func (v *Vault) Persist() error {
	errs := make(chan error)
	vault := v.vault

	for _, store := range v.persisters {
		go func() {
			errs <- store.Persist(vault)
		}()
	}

	// TODO: this really should return ALL errors, not just the first one
	for range v.persisters {
		select {
		case err := <-errs:
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (v *Vault) Load() error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	for _, store := range v.persisters {
		values, err := store.Load()
		if err != nil {
			return err
		}

		for key, value := range values {
			v.vault[key] = value
		}
	}

	return nil
}

func (v *Vault) Exists(key string) bool {
	_, ok := v.vault[key]
	return ok
}

func (v *Vault) Put(items ...Keyer) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	for _, item := range items {
		v.vault[item.Key()] = item
	}

	return nil
}

func (v *Vault) Get(key string) (Keyer, error) {
	if !v.Exists(key) {
		return nil, NotFoundError
	}

	return v.vault[key], nil
}

func (v *Vault) Filter(f FilterFunc) map[string]Keyer {
	filtered := map[string]Keyer{}

	for key, value := range v.vault {
		if f(value) {
			filtered[key] = value
		}
	}

	return filtered
}
